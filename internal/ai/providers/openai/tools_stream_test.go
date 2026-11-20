package openai

import (
	"context"
	"encoding/json"
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

func mockResponse(t *testing.T, client *http.Client, handle func(*http.Request) string) {
	t.Helper()
	original := client.Transport
	client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(handle(req))),
			Header:     make(http.Header),
		}, nil
	})
	t.Cleanup(func() { client.Transport = original })
}

func decodeRequest(t *testing.T, req *http.Request) map[string]any {
	t.Helper()
	if req.Method != http.MethodPost || req.URL.String() != responsesEndpoint ||
		req.Header.Get("Authorization") != "Bearer test-key" {
		t.Fatalf("se esperaba POST a Responses con autenticación por encabezado")
	}
	var body map[string]any
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["store"] != false || body["model"] != "gpt-6.2-sol" {
		t.Fatalf("solicitud de OpenAI inesperada: %+v", body)
	}
	return body
}

func TestStreamChatWithTools_ContinuesStatelessTurn(t *testing.T) {
	const first = `data: {"type":"response.output_text.delta","delta":"Voy a leer "}
data: {"type":"response.output_text.delta","delta":"los archivos."}
data: {"type":"response.function_call_arguments.delta","item_id":"fc_a","delta":"{\"path\":"}
data: {"type":"response.function_call_arguments.delta","item_id":"fc_a","delta":"\"a.go\"}"}
data: {"type":"response.completed","response":{"status":"completed","model":"gpt-6.2-sol","usage":{"input_tokens":30,"output_tokens":12,"input_tokens_details":{"cached_tokens":4,"cache_write_tokens":5,"cache_write_tokens_details":{"ttl_30m_tokens":3,"ttl_12h_tokens":2}}},"output":[
`
	const firstOutput = `{"type":"reasoning","id":"rs_a","encrypted_content":"opaque-reasoning","summary":[]},
{"type":"message","id":"msg_a","role":"assistant","phase":"commentary","status":"completed","content":[{"type":"output_text","text":"Voy a leer los archivos.","annotations":[]}]},
{"type":"function_call","id":"fc_a","call_id":"call_a","status":"completed","name":"read_file","arguments":"{\"path\":\"a.go\"}"},
{"type":"function_call","id":"fc_b","call_id":"call_b","status":"completed","name":"read_file","arguments":"{\"path\":\"b.go\"}"}`
	const second = `data: {"type":"response.output_text.delta","delta":"Listo."}
data: {"type":"response.completed","response":{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"Listo."}]}]}}
`
	var calls int
	mockResponse(t, transport.StreamClient, func(req *http.Request) string {
		calls++
		payload := decodeRequest(t, req)
		if payload["stream"] != true || payload["max_output_tokens"] != float64(4096) ||
			payload["reasoning"].(map[string]any)["effort"] != "low" || req.Header.Get("Accept") != "text/event-stream" {
			t.Fatalf("configuración inesperada para streaming: %+v", payload)
		}
		tools := payload["tools"].([]any)
		if len(tools) != 1 {
			t.Fatalf("se esperaba una herramienta: %+v", tools)
		}
		tool := tools[0].(map[string]any)
		if tool["type"] != "function" || tool["name"] != "read_file" || tool["strict"] != false {
			t.Fatalf("definición de herramienta inesperada: %+v", tools)
		}
		input := payload["input"].([]any)
		if calls == 1 {
			if len(input) != 1 {
				t.Fatalf("historial inicial inesperado: %+v", input)
			}
			return strings.TrimSuffix(first, "\n") + strings.ReplaceAll(firstOutput, "\n", "") + "]}}\n"
		}
		if len(input) != 7 {
			t.Fatalf("historial de continuación inesperado: %+v", input)
		}
		reasoning := input[1].(map[string]any)
		commentary := input[2].(map[string]any)
		call := input[3].(map[string]any)
		output := input[5].(map[string]any)
		secondOutput := input[6].(map[string]any)
		if reasoning["encrypted_content"] != "opaque-reasoning" || commentary["phase"] != "commentary" ||
			call["id"] != "fc_a" || output["type"] != "function_call_output" ||
			output["call_id"] != "call_a" || output["output"] != "a" || secondOutput["call_id"] != "call_b" {
			t.Fatalf("la continuación perdió los elementos originales o los resultados enlazados: %+v", input)
		}
		return second
	})
	client := Client{}
	messages := []domain.ChatMessage{{Role: domain.ChatRoleUser, Content: "Lee ambos archivos"}}
	tools := []domain.ToolDefinition{{Name: "read_file", InputSchema: map[string]any{"type": "object"}}}
	var text strings.Builder
	var toolCalls []domain.ToolCall
	result, err := client.StreamChatWithTools(context.Background(), "test-key", "", messages, tools, func(chunk StreamChunk) error {
		text.WriteString(chunk.Text)
		if chunk.ToolCall != nil {
			toolCalls = append(toolCalls, *chunk.ToolCall)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if text.String() != "Voy a leer los archivos." || result.Content != text.String() ||
		result.TokensPrompt != 30 || result.TokensCompletion != 12 || result.Usage == nil ||
		result.Usage.CacheRead != 4 || result.Usage.CacheWriteShort != 3 || result.Usage.CacheWriteLong != 2 || len(toolCalls) != 2 ||
		toolCalls[0].ID != "call_a" || toolCalls[0].Arguments != `{"path":"a.go"}` ||
		toolCalls[1].ID != "call_b" || toolCalls[1].Arguments != `{"path":"b.go"}` {
		t.Fatalf("respuesta de streaming inesperada: %+v, texto %q, llamadas %+v", result, text.String(), toolCalls)
	}

	messages = append(messages,
		domain.ChatMessage{Role: domain.ChatRoleAssistant, Content: result.Content, ToolCalls: result.ToolCalls, ProviderContent: result.ProviderContent},
		domain.ChatMessage{Role: domain.ChatRoleUser, ToolResults: []domain.ToolResult{
			{ToolCallID: "call_a", Content: "a"}, {ToolCallID: "call_b", Content: "b"},
		}},
	)
	result, err = client.StreamChatWithTools(context.Background(), "test-key", "", messages, tools, nil)
	if err != nil || result.Content != "Listo." || result.Model != "gpt-6.2-sol" {
		t.Fatalf("continuación inesperada: %+v, %v", result, err)
	}
}

func TestStreamChatWithTools_RejectsInvalidOrUnfinishedCalls(t *testing.T) {
	for _, test := range []struct {
		name  string
		event string
	}{
		{"truncated", `{"type":"response.output_item.done","item":{"type":"function_call","name":"read_file","call_id":"call_x","arguments":"{}"}}`},
		{"incomplete", `{"type":"response.incomplete","response":{"status":"incomplete","incomplete_details":{"reason":"max_output_tokens"}}}`},
		{"invalid arguments", `{"type":"response.completed","response":{"status":"completed","output":[{"type":"function_call","name":"read_file","call_id":"call_x","arguments":"{secret"}]}}`},
		{"provider error", `{"type":"error","code":"server_error","message":"secret https://private.invalid"}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			mockResponse(t, transport.StreamClient, func(*http.Request) string { return "data: " + test.event + "\n" })
			toolEmitted := false
			result, err := (Client{}).StreamChatWithTools(context.Background(), "test-key", "",
				[]domain.ChatMessage{{Role: domain.ChatRoleUser, Content: "hola"}},
				[]domain.ToolDefinition{{Name: "read_file"}}, func(chunk StreamChunk) error {
					toolEmitted = toolEmitted || chunk.ToolCall != nil
					return nil
				})
			if result != nil || err == nil || strings.Contains(err.Error(), "secret") || toolEmitted {
				t.Fatalf("respuesta insegura: %+v, %v, llamada emitida %t", result, err, toolEmitted)
			}
		})
	}
}

func TestSendTestMessage_UsesStatelessResponses(t *testing.T) {
	mockResponse(t, transport.Client, func(req *http.Request) string {
		payload := decodeRequest(t, req)
		if payload["stream"] != nil || payload["max_output_tokens"] != float64(256) || payload["tools"] != nil {
			t.Fatalf("configuración inesperada para prueba de conexión: %+v", payload)
		}
		return `{"status":"completed","model":"gpt-6.2-sol","output":[{"type":"message","content":[{"type":"output_text","text":"Funciona."}]}]}`
	})
	result, err := (Client{}).SendTestMessage(context.Background(), "test-key", "Hola", "")
	if err != nil || !result.Success || result.Model != "gpt-6.2-sol" || result.ResponseText != "Funciona." {
		t.Fatalf("prueba de conexión inesperada: %+v, %v", result, err)
	}
}

func TestStreamChatWithTools_RejectsEmptyAPIKey(t *testing.T) {
	_, err := (Client{}).StreamChatWithTools(context.Background(), "", "",
		[]domain.ChatMessage{{Role: domain.ChatRoleUser, Content: "hola"}}, nil, nil)
	if err == nil {
		t.Fatal("se esperaba error con clave de API vacía")
	}
}
