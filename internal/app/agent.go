package app

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"merlincode/internal/agent"
	"merlincode/internal/domain"
)

const MaxTurnTimeout = agent.MaxTurnTimeout

// Native event hooks are replaceable in adapter integration tests.
var emitChatStreamEvent = runtime.EventsEmit
var logAgentTurnError = runtime.LogErrorf

func (a *App) newAgentService() *agent.Service {
	var messages agent.MessageStore
	if a.sessionService != nil {
		messages = a.sessionService
	}
	return agent.NewService(a.aiProviderService, a.workspaceService, messages, agent.Callbacks{
		Emit:             func(event domain.ChatStreamEvent) { emitChatStreamEvent(a.ctx, "chat:stream", event) },
		WorkspaceChanged: a.notifyWorkspaceChanged,
		LogError: func(req domain.ChatStreamRequest, err error) {
			logAgentTurnError(a.ctx, "turno de agente falló (provider=%s, model=%s): %v", req.ProviderID, req.ModelID, err)
		},
		FormatError: sanitizeAIProviderError,
	}, agent.NewRunTool(a.PrepareAppRun))
}

func (a *App) runAgentTurn(ctx context.Context, finish func(), req domain.ChatStreamRequest, messages []domain.ChatMessage) {
	defer finish()
	a.agentService.Run(ctx, req, messages)
}

// RespondToToolApproval forwards the user's explicit approval to the agent service.
func (a *App) RespondToToolApproval(requestID string, approved bool, editedContent string) error {
	if a.agentService == nil {
		return domain.ErrRuntimeNotInitialized
	}
	return a.agentService.RespondToToolApproval(requestID, approved, editedContent)
}
