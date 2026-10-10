package anthropic

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

// withFakeStreamTransport sustituye transport.StreamClient.Transport por uno que responde con el
// cuerpo SSE fijo dado, y lo restaura al terminar la prueba.
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

// Fixture SSE real de Anthropic (formato de docs.anthropic.com) para un turno que responde con texto
// y luego invoca una tool "read_file" cuyos argumentos llegan fragmentados en varios input_json_delta.
const toolUseSSEFixture = `event: message_start
data: {"type":"message_start","message":{"usage":{"input_tokens":42}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Voy a leer "}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"el archivo."}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: content_block_start
data: {"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"toolu_01ABC","name":"read_file"}}

event: content_block_delta
data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"pa"}}

event: content_block_delta
data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"th\": \"src/"}}

event: content_block_delta
data: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"main.go\"}"}}

event: content_block_stop
data: {"type":"content_block_stop","index":1}

event: message_delta
data: {"type":"message_delta","usage":{"output_tokens":17}}

event: message_stop
data: {"type":"message_stop"}

`

func TestStreamChatWithTools_AccumulatesFragmentedToolCallArguments(t *testing.T) {
	withFakeStreamTransport(t, toolUseSSEFixture)

	var gotContent strings.Builder
	var gotToolCalls []domain.ToolCall

	result, err := Client{}.StreamChatWithTools(
		context.Background(),
		"sk-ant-test-key",
		"claude-sonnet-5",
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
	if tc.ID != "toolu_01ABC" || tc.Name != "read_file" {
		t.Fatalf("tool call inesperada: %+v", tc)
	}
	wantArgs := `{"path": "src/main.go"}`
	if tc.Arguments != wantArgs {
		t.Fatalf("argumentos ensamblados inesperados: obtenido %q, esperado %q", tc.Arguments, wantArgs)
	}

	if len(result.ToolCalls) != 1 || result.ToolCalls[0].Arguments != wantArgs {
		t.Fatalf("ToolCalls del resultado final inesperadas: %+v", result.ToolCalls)
	}

	if result.TokensPrompt != 42 || result.TokensCompletion != 17 {
		t.Fatalf("conteo de tokens inesperado: prompt=%d completion=%d", result.TokensPrompt, result.TokensCompletion)
	}
}

func TestStreamChatWithTools_ToolCallWithEmptyArgumentsDefaultsToEmptyObject(t *testing.T) {
	sse := `event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_02","name":"list_files"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_stop
data: {"type":"message_stop"}

`
	withFakeStreamTransport(t, sse)

	result, err := Client{}.StreamChatWithTools(
		context.Background(),
		"sk-ant-test-key",
		"claude-sonnet-5",
		[]domain.ChatMessage{{Role: domain.ChatRoleUser, Content: "lista los archivos"}},
		[]domain.ToolDefinition{{Name: "list_files"}},
		nil,
	)
	if err != nil {
		t.Fatalf("StreamChatWithTools falló: %v", err)
	}
	if len(result.ToolCalls) != 1 || result.ToolCalls[0].Arguments != "{}" {
		t.Fatalf("se esperaba Arguments='{}' por defecto, obtenido: %+v", result.ToolCalls)
	}
}

func TestStreamChatWithTools_RejectsEmptyAPIKey(t *testing.T) {
	_, err := Client{}.StreamChatWithTools(
		context.Background(), "", "claude-sonnet-5",
		[]domain.ChatMessage{{Role: domain.ChatRoleUser, Content: "hola"}},
		nil, nil,
	)
	if err == nil {
		t.Fatal("se esperaba error con clave de API vacía")
	}
}
