package google

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

// Fixture SSE en el formato de Gemini: a diferencia de OpenAI/Anthropic, functionCall.args llega
// completo en una sola parte (sin fragmentar), y trae thoughtSignature que debe conservarse.
const geminiToolCallSSEFixture = `data: {"candidates":[{"content":{"role":"model","parts":[{"text":"Voy a leer "}]}}]}

data: {"candidates":[{"content":{"role":"model","parts":[{"text":"el archivo."}]}}]}

data: {"candidates":[{"content":{"role":"model","parts":[{"functionCall":{"name":"read_file","args":{"path":"src/main.go"}},"thoughtSignature":"sig-abc"}]}}]}

data: {"candidates":[{"content":{"role":"model","parts":[]}}],"usageMetadata":{"promptTokenCount":25,"candidatesTokenCount":9,"totalTokenCount":34}}

`

func TestStreamChatWithTools_ParsesUnfragmentedFunctionCallArgs(t *testing.T) {
	withFakeStreamTransport(t, geminiToolCallSSEFixture)

	var gotContent strings.Builder
	var gotToolCalls []domain.ToolCall

	result, err := Client{}.StreamChatWithTools(
		context.Background(),
		"AIzaSyTestKey",
		"gemini-3.8-flash",
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
	if tc.ID != "gemini-tool-0" {
		t.Fatalf("se esperaba un ID sintetizado gemini-tool-0, obtenido: %q", tc.ID)
	}
	if tc.Name != "read_file" {
		t.Fatalf("nombre de tool call inesperado: %q", tc.Name)
	}
	if tc.Arguments != `{"path":"src/main.go"}` {
		t.Fatalf("argumentos inesperados: %q", tc.Arguments)
	}
	if tc.ThoughtSignature != "sig-abc" {
		t.Fatalf("thoughtSignature no se conservó: %q", tc.ThoughtSignature)
	}

	if len(result.ToolCalls) != 1 || result.ToolCalls[0].ID != "gemini-tool-0" {
		t.Fatalf("ToolCalls del resultado final inesperadas: %+v", result.ToolCalls)
	}

	if result.TokensPrompt != 25 || result.TokensCompletion != 9 {
		t.Fatalf("conteo de tokens inesperado: prompt=%d completion=%d", result.TokensPrompt, result.TokensCompletion)
	}
}

func TestStreamChatWithTools_MultipleFunctionCallsGetSequentialSynthesizedIDs(t *testing.T) {
	sse := `data: {"candidates":[{"content":{"role":"model","parts":[{"functionCall":{"name":"read_file","args":{"path":"a.txt"}}}]}}]}

data: {"candidates":[{"content":{"role":"model","parts":[{"functionCall":{"name":"read_file","args":{"path":"b.txt"}}}]}}]}

`
	withFakeStreamTransport(t, sse)

	result, err := Client{}.StreamChatWithTools(
		context.Background(),
		"AIzaSyTestKey",
		"gemini-3.8-flash",
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
	if result.ToolCalls[0].ID != "gemini-tool-0" || result.ToolCalls[1].ID != "gemini-tool-1" {
		t.Fatalf("IDs sintetizados no son secuenciales: %+v", result.ToolCalls)
	}
}

func TestStreamChatWithTools_ThinkingPartsAreSeparatedFromContent(t *testing.T) {
	sse := `data: {"candidates":[{"content":{"role":"model","parts":[{"text":"razonando...","thought":true}]}}]}

data: {"candidates":[{"content":{"role":"model","parts":[{"text":"respuesta final"}]}}]}

`
	withFakeStreamTransport(t, sse)

	result, err := Client{}.StreamChatWithTools(
		context.Background(),
		"AIzaSyTestKey",
		"gemini-3.8-flash",
		[]domain.ChatMessage{{Role: domain.ChatRoleUser, Content: "hola"}},
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("StreamChatWithTools falló: %v", err)
	}
	if result.Thinking != "razonando..." {
		t.Fatalf("Thinking inesperado: %q", result.Thinking)
	}
	if result.Content != "respuesta final" {
		t.Fatalf("Content inesperado: %q", result.Content)
	}
}

func TestStreamChatWithTools_RejectsEmptyAPIKey(t *testing.T) {
	_, err := Client{}.StreamChatWithTools(
		context.Background(), "", "gemini-3.8-flash",
		[]domain.ChatMessage{{Role: domain.ChatRoleUser, Content: "hola"}},
		nil, nil,
	)
	if err == nil {
		t.Fatal("se esperaba error con clave de API vacía")
	}
}
