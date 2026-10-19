package agent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"merlincode/internal/domain"
)

// handleAgentTurnError maneja el error final de un turno de agente (tras agotar reintentos de la capa
// de streaming), persistiendo lo generado hasta el momento y notificando a la UI.
func (a *Service) handleAgentTurnError(req domain.ChatStreamRequest, err error, streamStartTime time.Time, content string, thinking string, toolTrace []domain.ToolTraceEntry) {
	elapsedSec := max(1, int(time.Since(streamStartTime).Seconds()))

	if errors.Is(err, context.Canceled) {
		a.handleAgentTurnCanceled(req, streamStartTime, content, thinking, toolTrace)
		return
	}

	a.callbacks.LogError(req, err)

	sanitized := a.callbacks.FormatError(err)
	a.callbacks.Emit(domain.ChatStreamEvent{
		SessionID:  req.SessionID,
		MessageID:  req.MessageID,
		Type:       domain.ChunkTypeError,
		Error:      sanitized.Error(),
		ProviderID: req.ProviderID,
	})

	if a.sessionService != nil && req.SessionID != "" {
		errContent := content
		if errContent != "" {
			errContent += fmt.Sprintf("\n\n[Error: %s]", sanitized.Error())
		} else {
			errContent = fmt.Sprintf("Error: %s", sanitized.Error())
		}
		asstRecord := domain.ChatMessageRecord{
			ID:              req.MessageID,
			SessionID:       req.SessionID,
			Role:            domain.ChatRoleAssistant,
			Content:         errContent,
			ThoughtChain:    thinking,
			ProviderID:      req.ProviderID,
			ModelID:         req.ModelID,
			DurationSeconds: elapsedSec,
			Status:          "error",
			CreatedAt:       time.Now().UTC(),
			ToolTrace:       toolTrace,
		}
		_ = a.sessionService.SaveMessage(context.Background(), asstRecord)
	}
}

// handleAgentTurnCanceled maneja la cancelación explícita del turno (vía CancelChatStream).
func (a *Service) handleAgentTurnCanceled(req domain.ChatStreamRequest, streamStartTime time.Time, content string, thinking string, toolTrace []domain.ToolTraceEntry) {
	elapsedSec := max(1, int(time.Since(streamStartTime).Seconds()))

	a.callbacks.Emit(domain.ChatStreamEvent{
		SessionID:  req.SessionID,
		MessageID:  req.MessageID,
		Type:       domain.ChunkTypeDone,
		StatusText: "Generación detenida.",
		ProviderID: req.ProviderID,
	})

	if a.sessionService != nil && req.SessionID != "" {
		stoppedContent := content
		if stoppedContent != "" {
			stoppedContent += "\n\n*(Generación detenida)*"
		} else {
			stoppedContent = "*(Generación detenida)*"
		}
		asstRecord := domain.ChatMessageRecord{
			ID:              req.MessageID,
			SessionID:       req.SessionID,
			Role:            domain.ChatRoleAssistant,
			Content:         stoppedContent,
			ThoughtChain:    thinking,
			ProviderID:      req.ProviderID,
			ModelID:         req.ModelID,
			DurationSeconds: elapsedSec,
			Status:          "done",
			CreatedAt:       time.Now().UTC(),
			ToolTrace:       toolTrace,
		}
		_ = a.sessionService.SaveMessage(context.Background(), asstRecord)
	}
}

// finishAgentTurn persiste y notifica la finalización exitosa de un turno de agente (sin más tool calls pendientes).
func (a *Service) finishAgentTurn(req domain.ChatStreamRequest, result *domain.ChatCompletionResult, streamStartTime time.Time, toolTrace []domain.ToolTraceEntry) {
	elapsedSec := max(1, int(time.Since(streamStartTime).Seconds()))

	if a.sessionService != nil && req.SessionID != "" {
		asstRecord := domain.ChatMessageRecord{
			ID:               req.MessageID,
			SessionID:        req.SessionID,
			Role:             domain.ChatRoleAssistant,
			Content:          result.Content,
			ThoughtChain:     result.Thinking,
			ProviderID:       req.ProviderID,
			ModelID:          result.Model,
			TokensPrompt:     result.TokensPrompt,
			TokensCompletion: result.TokensCompletion,
			DurationSeconds:  elapsedSec,
			Status:           "done",
			CreatedAt:        time.Now().UTC(),
			ToolTrace:        toolTrace,
		}
		_ = a.sessionService.SaveMessage(context.Background(), asstRecord)
	}

	a.callbacks.Emit(domain.ChatStreamEvent{
		SessionID:        req.SessionID,
		MessageID:        req.MessageID,
		Type:             domain.ChunkTypeDone,
		ProviderID:       req.ProviderID,
		ModelID:          result.Model,
		TokensPrompt:     result.TokensPrompt,
		TokensCompletion: result.TokensCompletion,
	})
}
