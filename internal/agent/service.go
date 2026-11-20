package agent

import (
	"context"
	"strings"
	"sync"
	"time"

	"merlincode/internal/ai/pricing"
	"merlincode/internal/domain"
	"merlincode/internal/workspace"
)

// maxToolIterations limita cuántas rondas de tool_call/tool_result puede tener un solo turno de agente.
const maxToolIterations = 8

// MaxTurnTimeout cubre el turno de agente completo, incluyendo esperas de aprobación humana —
// distinto de ai.ChatTimeout, que cubre solo el round-trip individual con el proveedor.
const MaxTurnTimeout = 15 * time.Minute

// ChatProvider is the provider capability consumed by the agent loop.
type ChatProvider interface {
	StreamChatWithTools(ctx context.Context, providerID, model string, messages []domain.ChatMessage,
		tools []domain.ToolDefinition, onChunk func(domain.StreamChunk) error) (*domain.ChatCompletionResult, error)
}

// MessageStore persists turns without coupling the agent to SQLite.
type MessageStore interface {
	SaveMessage(context.Context, domain.ChatMessageRecord) error
}

// Callbacks connect agent events to the application adapter without depending on Wails.
type Callbacks struct {
	Emit             func(domain.ChatStreamEvent)
	WorkspaceChanged func()
	LogError         func(domain.ChatStreamRequest, error)
	FormatError      func(error) error
}

// Service coordinates model turns, tool execution and human approvals.
// The caller owns cancellation and the stream slot for the lifetime of Run.
type Service struct {
	aiProviderService  ChatProvider
	workspaceService   *workspace.Service
	sessionService     MessageStore
	callbacks          Callbacks
	extraTools         []ToolExecutor
	pendingApprovalsMu sync.Mutex
	pendingApprovals   map[string]*pendingApproval
}

func NewService(provider ChatProvider, ws *workspace.Service, messages MessageStore, callbacks Callbacks, extra ...ToolExecutor) *Service {
	if callbacks.Emit == nil {
		callbacks.Emit = func(domain.ChatStreamEvent) {}
	}
	if callbacks.WorkspaceChanged == nil {
		callbacks.WorkspaceChanged = func() {}
	}
	if callbacks.LogError == nil {
		callbacks.LogError = func(domain.ChatStreamRequest, error) {}
	}
	if callbacks.FormatError == nil {
		callbacks.FormatError = func(error) error { return domain.ErrAIProviderOperationFailed }
	}
	return &Service{
		aiProviderService: provider,
		workspaceService:  ws,
		sessionService:    messages,
		callbacks:         callbacks,
		extraTools:        append([]ToolExecutor(nil), extra...),
		pendingApprovals:  make(map[string]*pendingApproval),
	}
}

// Run ejecuta el bucle completo de un turno de agente (asistente -> tool_call -> tool_result
// -> asistente...) dentro del slot reservado por el llamador, sin abrir un segundo stream concurrente.
// El proveedor inyectado resuelve el soporte de tool-calling y su fallback.
func (a *Service) Run(ctx context.Context, req domain.ChatStreamRequest, messages []domain.ChatMessage) {
	streamStartTime := time.Now()
	registry := NewToolRegistry(a.workspaceService, a.extraTools...)
	tools := registry.Definitions()

	var fullContent strings.Builder
	var fullThinking strings.Builder
	var toolTrace []domain.ToolTraceEntry
	var usage turnUsage

	for iteration := 0; iteration < maxToolIterations; iteration++ {
		requestedAt := time.Now()
		result, err := a.aiProviderService.StreamChatWithTools(
			ctx,
			req.ProviderID,
			req.ModelID,
			messages,
			tools,
			func(chunk domain.StreamChunk) error {
				event := domain.ChatStreamEvent{
					SessionID:  req.SessionID,
					MessageID:  req.MessageID,
					Type:       chunk.Type,
					Content:    chunk.Text,
					Thinking:   chunk.Thinking,
					ProviderID: req.ProviderID,
					ModelID:    req.ModelID,
				}
				switch chunk.Type {
				case domain.ChunkTypeContent:
					fullContent.WriteString(chunk.Text)
				case domain.ChunkTypeThinking:
					fullThinking.WriteString(chunk.Thinking)
				case domain.ChunkTypeToolCall:
					if chunk.ToolCall != nil {
						event.ToolName = chunk.ToolCall.Name
						event.ToolCallID = chunk.ToolCall.ID
						event.ToolArgsSummary = summarizeToolArgs(chunk.ToolCall.Arguments)
					}
				}
				a.callbacks.Emit(event)
				return nil
			},
		)

		usage.record(req, result, requestedAt)
		if err != nil {
			a.handleAgentTurnError(req, err, streamStartTime, fullContent.String(), fullThinking.String(), toolTrace, usage)
			return
		}

		if len(result.ToolCalls) == 0 {
			a.finishAgentTurn(req, result, streamStartTime, toolTrace, usage)
			return
		}

		messages = append(messages, domain.ChatMessage{
			Role:            domain.ChatRoleAssistant,
			Content:         result.Content,
			ToolCalls:       result.ToolCalls,
			ProviderContent: result.ProviderContent,
		})

		toolResults, canceled := a.ExecuteTools(ctx, req, registry, result.ToolCalls, &toolTrace)
		if canceled {
			a.handleAgentTurnCanceled(req, streamStartTime, fullContent.String(), fullThinking.String(), toolTrace, usage)
			return
		}

		messages = append(messages, domain.ChatMessage{
			Role:        domain.ChatRoleUser,
			ToolResults: toolResults,
		})
	}

	a.handleAgentTurnError(req, domain.ErrMaxToolIterationsExceeded, streamStartTime, fullContent.String(), fullThinking.String(), toolTrace, usage)
}

type turnUsage struct {
	prompt, completion int64
	cost               domain.TurnCost
}

func (u *turnUsage) record(req domain.ChatStreamRequest, result *domain.ChatCompletionResult, requestedAt time.Time) {
	u.cost.Requests++
	if result == nil {
		u.cost.Partial = true
		return
	}
	u.prompt += max(0, min(result.TokensPrompt, 2_000_000))
	u.completion += max(0, min(result.TokensCompletion, 2_000_000))
	model := req.ModelID
	if model == "" {
		model = result.Model
	}
	amount := pricing.Estimate(req.ProviderID, model, result, requestedAt)
	if amount == nil {
		u.cost.Partial = true
		return
	}
	if u.cost.NanosUSD == nil {
		u.cost.NanosUSD = amount
	} else {
		*u.cost.NanosUSD += *amount
	}
}
