package ai

import (
	"context"
	"errors"
	"strings"
	"testing"

	"merlincode/internal/domain"
)

// fakeToolCaller es el mismo tipo de doble de prueba que fakeStreamer (declarado en service_test.go),
// pero implementa además ToolCaller, para probar el despacho por type assertion en StreamChatWithTools.
type fakeToolCaller struct {
	fakeValidator
	gotTools []domain.ToolDefinition
	chunks   []domain.StreamChunk
	result   *domain.ChatCompletionResult
	err      error
}

func (f *fakeToolCaller) StreamChatWithTools(
	ctx context.Context,
	apiKey string,
	model string,
	messages []domain.ChatMessage,
	tools []domain.ToolDefinition,
	onChunk func(chunk domain.StreamChunk) error,
) (*domain.ChatCompletionResult, error) {
	f.gotTools = tools
	if f.err != nil {
		return nil, f.err
	}
	for _, c := range f.chunks {
		if onChunk != nil {
			if err := onChunk(c); err != nil {
				return nil, err
			}
		}
	}
	if f.result != nil {
		return f.result, nil
	}
	return &domain.ChatCompletionResult{Content: "respuesta de prueba", Model: model}, nil
}

func TestStreamChatWithTools_DispatchesToToolCaller(t *testing.T) {
	toolCaller := &fakeToolCaller{
		fakeValidator: fakeValidator{result: domain.ProviderValidationResult{Valid: true}},
		result: &domain.ChatCompletionResult{
			Content: "listo",
			Model:   "claude-sonnet-5",
			ToolCalls: []domain.ToolCall{
				{ID: "call-1", Name: "read_file", Arguments: `{"path":"a.txt"}`},
			},
		},
	}
	svc, _, _ := newTestService(t, map[string]Provider{"anthropic": toolCaller})
	_ = svc.setSecretVerified("anthropic", "sk-ant-test-key")
	_ = svc.hydrateFromKeyring()

	tools := []domain.ToolDefinition{{Name: "read_file", Description: "lee un archivo"}}

	result, err := svc.StreamChatWithTools(
		context.Background(),
		"anthropic",
		"claude-sonnet-5",
		[]domain.ChatMessage{{Role: domain.ChatRoleUser, Content: "hola"}},
		tools,
		nil,
	)
	if err != nil {
		t.Fatalf("StreamChatWithTools falló: %v", err)
	}
	if len(result.ToolCalls) != 1 || result.ToolCalls[0].Name != "read_file" {
		t.Fatalf("tool calls no propagadas: %+v", result.ToolCalls)
	}
	if len(toolCaller.gotTools) != 1 || toolCaller.gotTools[0].Name != "read_file" {
		t.Fatalf("las definiciones de herramientas no se pasaron al proveedor: %+v", toolCaller.gotTools)
	}
}

func TestStreamChatWithTools_FallsBackToStreamerWhenNoToolCaller(t *testing.T) {
	streamer := &fakeStreamer{
		fakeValidator: fakeValidator{result: domain.ProviderValidationResult{Valid: true}},
		result:        &domain.ChatCompletionResult{Content: "sin tools", Model: "gemini-3.8-flash"},
	}
	svc, _, _ := newTestService(t, map[string]Provider{"google": streamer})
	_ = svc.setSecretVerified("google", "AIzaSyTestKey")
	_ = svc.hydrateFromKeyring()

	result, err := svc.StreamChatWithTools(
		context.Background(),
		"google",
		"gemini-3.8-flash",
		[]domain.ChatMessage{{Role: domain.ChatRoleUser, Content: "hola"}},
		[]domain.ToolDefinition{{Name: "read_file"}},
		nil,
	)
	if err != nil {
		t.Fatalf("StreamChatWithTools falló: %v", err)
	}
	if result.Content != "sin tools" {
		t.Fatalf("se esperaba el fallback a Streamer.StreamChat, obtenido: %+v", result)
	}
}

func TestStreamChatWithTools_UnconfiguredProviderFails(t *testing.T) {
	toolCaller := &fakeToolCaller{fakeValidator: fakeValidator{result: domain.ProviderValidationResult{Valid: true}}}
	svc, _, _ := newTestService(t, map[string]Provider{"openai": toolCaller})

	_, err := svc.StreamChatWithTools(
		context.Background(),
		"openai",
		"gpt-5.6-terra",
		[]domain.ChatMessage{{Role: domain.ChatRoleUser, Content: "hola"}},
		nil,
		nil,
	)
	if err == nil {
		t.Fatal("se esperaba error al intentar streaming con tools sin un proveedor configurado")
	}
}

func TestStreamChatWithTools_OutputTooLargeIsRejected(t *testing.T) {
	toolCaller := &fakeToolCaller{
		fakeValidator: fakeValidator{result: domain.ProviderValidationResult{Valid: true}},
		chunks: []domain.StreamChunk{
			{Type: domain.ChunkTypeContent, Text: strings.Repeat("x", MaxChatOutputBytes+1)},
		},
	}
	svc, _, _ := newTestService(t, map[string]Provider{"anthropic": toolCaller})
	_ = svc.setSecretVerified("anthropic", "sk-ant-test-key")
	_ = svc.hydrateFromKeyring()

	_, err := svc.StreamChatWithTools(
		context.Background(),
		"anthropic",
		"claude-sonnet-5",
		[]domain.ChatMessage{{Role: domain.ChatRoleUser, Content: "hola"}},
		nil,
		func(domain.StreamChunk) error { return nil },
	)
	if !errors.Is(err, domain.ErrChatOutputTooLarge) {
		t.Fatalf("esperado ErrChatOutputTooLarge, obtenido: %v", err)
	}
}

func TestStreamChatWithTools_PropagatesProviderError(t *testing.T) {
	toolCaller := &fakeToolCaller{
		fakeValidator: fakeValidator{result: domain.ProviderValidationResult{Valid: true}},
		err:           errors.New("fallo simulado del proveedor"),
	}
	svc, _, _ := newTestService(t, map[string]Provider{"anthropic": toolCaller})
	_ = svc.setSecretVerified("anthropic", "sk-ant-test-key")
	_ = svc.hydrateFromKeyring()

	_, err := svc.StreamChatWithTools(
		context.Background(),
		"anthropic",
		"claude-sonnet-5",
		[]domain.ChatMessage{{Role: domain.ChatRoleUser, Content: "hola"}},
		nil,
		nil,
	)
	if err == nil {
		t.Fatal("se esperaba propagar el error del proveedor")
	}
}
