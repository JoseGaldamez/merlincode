package agent

import (
	"context"
	"strings"
	"sync"
	"time"

	"merlincode/internal/domain"
	"merlincode/internal/repomap"
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
	repoMapService     *repomap.Service
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
		repoMapService:    repomap.NewService(ws),
	}
}

// Run ejecuta el bucle completo de un turno de agente (asistente -> tool_call -> tool_result
// -> asistente...) dentro del slot reservado por el llamador, sin abrir un segundo stream concurrente.
// El proveedor inyectado resuelve el soporte de tool-calling y su fallback.
func (a *Service) Run(ctx context.Context, req domain.ChatStreamRequest, messages []domain.ChatMessage) {
	streamStartTime := time.Now()
	registry := NewToolRegistry(a.workspaceService, a.extraTools...)
	tools := registry.Definitions()

	if repoMap := a.repoMapService.Snapshot(); repoMap != "" {
		messages = append([]domain.ChatMessage{{Role: domain.ChatRoleSystem, Content: repoMap}}, messages...)
	}

	var fullContent strings.Builder
	var fullThinking strings.Builder
	var toolTrace []domain.ToolTraceEntry

	for iteration := 0; iteration < maxToolIterations; iteration++ {
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

		if err != nil {
			a.handleAgentTurnError(req, err, streamStartTime, fullContent.String(), fullThinking.String(), toolTrace)
			return
		}

		if len(result.ToolCalls) == 0 {
			a.finishAgentTurn(req, result, streamStartTime, toolTrace)
			return
		}

		messages = append(messages, domain.ChatMessage{
			Role:      domain.ChatRoleAssistant,
			Content:   result.Content,
			ToolCalls: result.ToolCalls,
		})

		toolResults, canceled := a.ExecuteTools(ctx, req, registry, result.ToolCalls, &toolTrace)
		if canceled {
			a.handleAgentTurnCanceled(req, streamStartTime, fullContent.String(), fullThinking.String(), toolTrace)
			return
		}

		messages = append(messages, domain.ChatMessage{
			Role:        domain.ChatRoleUser,
			ToolResults: toolResults,
		})
	}

	a.handleAgentTurnError(req, domain.ErrMaxToolIterationsExceeded, streamStartTime, fullContent.String(), fullThinking.String(), toolTrace)
}
