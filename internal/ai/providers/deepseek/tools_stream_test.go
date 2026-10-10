package deepseek

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"merlincode/internal/ai/transport"
	"merlincode/internal/domain"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func withFakeStreamTransport(t *testing.T, sseBody string) {
	t.Helper()
	orig := transport.StreamClient.Transport
	transport.StreamClient.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(sseBody)),
			Header:     make(http.Header),
		}, nil
	})
	t.Cleanup(func() { transport.StreamClient.Transport = orig })
}

// Fixture SSE en el formato compatible con OpenAI que usa DeepSeek: la tool call llega
// fragmentada en varias piezas de delta.tool_calls[].function.arguments (mismo índice).
const deepSeekToolCallSSEFixture = `data: {"choices":[{"delta":{"content":"Voy a leer "},"finish_reason":null}]}

data: {"choices":[{"delta":{"content":"el archivo."},"finish_reason":null}]}

data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_ds1","function":{"name":"read_file","arguments":""}}]},"finish_reason":null}]}

data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"pa"}}]},"finish_reason":null}]}

data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"th\": \"src/"}}]},"finish_reason":null}]}

data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"main.go\"}"}}]},"finish_reason":null}]}

data: {"choices":[{"delta":{},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":30,"completion_tokens":12}}

data: [DONE]

`

func TestStreamChatWithTools_AccumulatesFragmentedToolCallArguments(t *testing.T) {
	withFakeStreamTransport(t, deepSeekToolCallSSEFixture)

	var gotContent strings.Builder
	var gotToolCalls []domain.ToolCall

	result, err := Client{}.StreamChatWithTools(
		context.Background(),
		"sk-deepseek-test-key",
		"deepseek-v4-pro",
		[]domain.ChatMessage{{Role: domain.ChatRoleUser, Content: "lee src/main.go"}},
		[]domain.ToolDefinition{{Name: "read_file", Description: "lee un archivo"}},
		func(chunk StreamChunk) error {
			if chunk.Type == domain.ChunkTypeContent {
				gotContent.WriteString(chunk.Text)
			}
			if chunk.Type == domain.ChunkTypeToolCall && chunk.ToolCall != nil {
				gotToolCalls = append(gotToolCalls, *chunk.ToolCall)
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("StreamChatWithTools falló: %v", err)
	}

	if gotContent.String() != "Voy a leer el archivo." {
		t.Fatalf("contenido acumulado inesperado: %q", gotContent.String())
	}
	if result.Content != "Voy a leer el archivo." {
		t.Fatalf("result.Content inesperado: %q", result.Content)
	}

	if len(gotToolCalls) != 1 {
		t.Fatalf("se esperaba 1 tool call emitida por chunk, obtenidas: %d", len(gotToolCalls))
	}
	tc := gotToolCalls[0]
	if tc.ID != "call_ds1" || tc.Name != "read_file" {
		t.Fatalf("tool call inesperada: %+v", tc)
	}
	wantArgs := `{"path": "src/main.go"}`
	if tc.Arguments != wantArgs {
		t.Fatalf("argumentos ensamblados inesperados: obtenido %q, esperado %q", tc.Arguments, wantArgs)
	}

	if len(result.ToolCalls) != 1 || result.ToolCalls[0].Arguments != wantArgs {
		t.Fatalf("ToolCalls del resultado final inesperadas: %+v", result.ToolCalls)
	}

	if result.TokensPrompt != 30 || result.TokensCompletion != 12 {
		t.Fatalf("conteo de tokens inesperado: prompt=%d completion=%d", result.TokensPrompt, result.TokensCompletion)
	}
}

func TestStreamChatWithTools_MultipleToolCallsAreDistinguishedByIndex(t *testing.T) {
	sse := `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","function":{"name":"read_file","arguments":"{\"path\":\"a.txt\"}"}}]},"finish_reason":null}]}

data: {"choices":[{"delta":{"tool_calls":[{"index":1,"id":"call_2","function":{"name":"read_file","arguments":"{\"path\":\"b.txt\"}"}}]},"finish_reason":null}]}

data: {"choices":[{"delta":{},"finish_reason":"tool_calls"}]}

data: [DONE]

`
	withFakeStreamTransport(t, sse)

	result, err := Client{}.StreamChatWithTools(
		context.Background(),
		"sk-deepseek-test-key",
		"deepseek-v4-pro",
		[]domain.ChatMessage{{Role: domain.ChatRoleUser, Content: "lee ambos archivos"}},
		[]domain.ToolDefinition{{Name: "read_file"}},
		nil,
	)
	if err != nil {
		t.Fatalf("StreamChatWithTools falló: %v", err)
	}
	if len(result.ToolCalls) != 2 {
		t.Fatalf("se esperaban 2 tool calls, obtenidas: %d", len(result.ToolCalls))
	}
	if result.ToolCalls[0].ID != "call_1" || result.ToolCalls[0].Arguments != `{"path":"a.txt"}` {
		t.Errorf("primera tool call inesperada: %+v", result.ToolCalls[0])
	}
	if result.ToolCalls[1].ID != "call_2" || result.ToolCalls[1].Arguments != `{"path":"b.txt"}` {
		t.Errorf("segunda tool call inesperada: %+v", result.ToolCalls[1])
	}
}

func TestStreamChatWithTools_InvalidAssembledJSONDefaultsToEmptyObject(t *testing.T) {
	sse := `data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_x","function":{"name":"list_files","arguments":"{esto no es json"}}]},"finish_reason":null}]}

data: {"choices":[{"delta":{},"finish_reason":"tool_calls"}]}

data: [DONE]

`
	withFakeStreamTransport(t, sse)

	result, err := Client{}.StreamChatWithTools(
		context.Background(),
		"sk-deepseek-test-key",
		"deepseek-v4-pro",
		[]domain.ChatMessage{{Role: domain.ChatRoleUser, Content: "lista los archivos"}},
		[]domain.ToolDefinition{{Name: "list_files"}},
		nil,
	)
	if err != nil {
		t.Fatalf("StreamChatWithTools falló: %v", err)
	}
	if len(result.ToolCalls) != 1 || result.ToolCalls[0].Arguments != "{}" {
		t.Fatalf("se esperaba Arguments='{}' por JSON inválido, obtenido: %+v", result.ToolCalls)
	}
}

func TestStreamChatWithTools_RejectsEmptyAPIKey(t *testing.T) {
	_, err := Client{}.StreamChatWithTools(
		context.Background(), "", "deepseek-v4-pro",
		[]domain.ChatMessage{{Role: domain.ChatRoleUser, Content: "hola"}},
		nil, nil,
	)
	if err == nil {
		t.Fatal("se esperaba error con clave de API vacía")
	}
}
