package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"merlincode/internal/domain"
)

type turnProviderFunc func(context.Context, string, string, []domain.ChatMessage, []domain.ToolDefinition, func(domain.StreamChunk) error) (*domain.ChatCompletionResult, error)

func (f turnProviderFunc) StreamChatWithTools(ctx context.Context, provider, model string, messages []domain.ChatMessage, tools []domain.ToolDefinition, emit func(domain.StreamChunk) error) (*domain.ChatCompletionResult, error) {
	return f(ctx, provider, model, messages, tools, emit)
}

type turnMessageStore struct{ records []domain.ChatMessageRecord }

func (s *turnMessageStore) SaveMessage(_ context.Context, record domain.ChatMessageRecord) error {
	s.records = append(s.records, record)
	return nil
}

func TestRunExecutesToolsAndPersistsCompletion(t *testing.T) {
	ws := newTestWorkspace(t)
	if err := ws.WriteFile("nota.txt", "contenido del proyecto"); err != nil {
		t.Fatal(err)
	}
	var calls int
	provider := turnProviderFunc(func(_ context.Context, provider, model string, messages []domain.ChatMessage, tools []domain.ToolDefinition, emit func(domain.StreamChunk) error) (*domain.ChatCompletionResult, error) {
		calls++
		if provider != "openai" || model != "gpt-6.2-sol" || len(tools) != 5 {
			t.Fatalf("provider configuration lost: %s %s %#v", provider, model, tools)
		}
		if calls == 1 {
			return &domain.ChatCompletionResult{
				TokensPrompt: 8, TokensCompletion: 6, Usage: &domain.ModelUsage{},
				ToolCalls: []domain.ToolCall{{ID: "read-1", Name: "read_file", Arguments: `{"path":"nota.txt"}`}},
			}, nil
		}
		if len(messages) != 3 || len(messages[1].ToolCalls) != 1 || len(messages[2].ToolResults) != 1 || messages[2].ToolResults[0].Content != "contenido del proyecto" {
			t.Fatalf("tool result was not returned to the provider: %#v", messages)
		}
		if err := emit(domain.StreamChunk{Type: domain.ChunkTypeContent, Text: "Respuesta"}); err != nil {
			t.Fatal(err)
		}
		return &domain.ChatCompletionResult{Content: "Respuesta", Model: model, TokensPrompt: 12, TokensCompletion: 4, Usage: &domain.ModelUsage{}}, nil
	})
	store := &turnMessageStore{}
	var events []domain.ChatStreamEvent
	svc := NewService(provider, ws, store, Callbacks{Emit: func(event domain.ChatStreamEvent) { events = append(events, event) }})
	svc.Run(context.Background(), domain.ChatStreamRequest{SessionID: "s1", MessageID: "m1", ProviderID: "openai", ModelID: "gpt-6.2-sol"}, []domain.ChatMessage{{Role: domain.ChatRoleUser, Content: "Lee la nota"}})
	if calls != 2 || len(store.records) != 1 {
		t.Fatalf("unexpected turn lifecycle: calls=%d records=%#v", calls, store.records)
	}
	record := store.records[0]
	if record.ID != "m1" || record.SessionID != "s1" || record.Status != "done" || record.Content != "Respuesta" || record.TokensPrompt != 20 || record.TokensCompletion != 10 || len(record.ToolTrace) != 1 || record.ToolTrace[0].Result != "contenido del proyecto" {
		t.Fatalf("completion or trace lost: %#v", record)
	}
	if record.Cost == nil || record.Cost.Partial || record.Cost.Requests != 2 ||
		record.Cost.NanosUSD == nil || *record.Cost.NanosUSD != 140_000 {
		t.Fatalf("cost did not include both requests: %+v", record.Cost)
	}
	if done := events[2]; done.TokensPrompt != 20 || done.TokensCompletion != 10 ||
		done.Cost == nil || done.Cost.NanosUSD == nil || *done.Cost.NanosUSD != 140_000 {
		t.Fatalf("terminal event lost turn totals: %+v", done)
	}
	if len(events) != 3 || events[0].Type != domain.ChunkTypeToolResult || events[1].Type != domain.ChunkTypeContent || events[2].Type != domain.ChunkTypeDone {
		t.Fatalf("missing tool, stream or completion event: %#v", events)
	}
	for _, event := range events {
		if event.MessageID != "m1" || event.SessionID != "s1" {
			t.Fatalf("event lost its chat identity: %#v", event)
		}
	}
}

func TestRunPersistsErrorsAndCancellation(t *testing.T) {
	for _, canceled := range []bool{false, true} {
		name := "provider error"
		if canceled {
			name = "cancellation"
		}
		t.Run(name, func(t *testing.T) {
			provider := turnProviderFunc(func(_ context.Context, _, _ string, _ []domain.ChatMessage, _ []domain.ToolDefinition, emit func(domain.StreamChunk) error) (*domain.ChatCompletionResult, error) {
				if err := emit(domain.StreamChunk{Type: domain.ChunkTypeContent, Text: "Parcial"}); err != nil {
					t.Fatal(err)
				}
				if canceled {
					return nil, context.Canceled
				}
				return nil, errors.New("PRIVATE_PROVIDER_DETAILS")
			})
			store := &turnMessageStore{}
			var events []domain.ChatStreamEvent
			svc := NewService(provider, newTestWorkspace(t), store, Callbacks{Emit: func(event domain.ChatStreamEvent) { events = append(events, event) }})
			svc.Run(context.Background(), domain.ChatStreamRequest{SessionID: "s1", MessageID: "m1"}, nil)
			if len(store.records) != 1 || len(events) != 2 || !strings.HasPrefix(store.records[0].Content, "Parcial") {
				t.Fatalf("interrupted turn not preserved: %#v %#v", store.records, events)
			}
			if strings.Contains(store.records[0].Content+events[1].Error, "PRIVATE_PROVIDER_DETAILS") {
				t.Fatal("internal provider details escaped the adapter")
			}
			if cost := store.records[0].Cost; cost == nil || !cost.Partial || cost.NanosUSD != nil {
				t.Fatalf("interrupted request without usage must be unknown: %+v", cost)
			}
			if canceled {
				if events[1].Type != domain.ChunkTypeDone || store.records[0].Status != "done" {
					t.Fatal("cancellation did not complete the turn")
				}
			} else if events[1].Type != domain.ChunkTypeError || store.records[0].Status != "error" || events[1].Error != domain.ErrAIProviderOperationFailed.Error() {
				t.Fatal("provider failure was not sanitized")
			}
		})
	}
}

func TestRunBoundsToolIterations(t *testing.T) {
	var calls int
	provider := turnProviderFunc(func(context.Context, string, string, []domain.ChatMessage, []domain.ToolDefinition, func(domain.StreamChunk) error) (*domain.ChatCompletionResult, error) {
		calls++
		return &domain.ChatCompletionResult{ToolCalls: []domain.ToolCall{{ID: "tree", Name: "get_tree", Arguments: "{}"}}}, nil
	})
	store := &turnMessageStore{}
	svc := NewService(provider, newTestWorkspace(t), store, Callbacks{})
	svc.Run(context.Background(), domain.ChatStreamRequest{SessionID: "s1"}, nil)
	if calls != maxToolIterations || len(store.records) != 1 || store.records[0].Status != "error" || len(store.records[0].ToolTrace) != maxToolIterations {
		t.Fatalf("tool loop did not terminate with its trace: calls=%d records=%#v", calls, store.records)
	}
}
