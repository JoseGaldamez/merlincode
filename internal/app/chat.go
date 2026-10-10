package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"merlincode/internal/ai"
	"merlincode/internal/domain"
)

// StartChatStream inicia la generación de respuesta en streaming para el chat de forma asíncrona,
// emitiendo eventos en tiempo real vía Wails sin bloquear el bucle de eventos ni el IPC del WebView.
func (a *App) StartChatStream(req domain.ChatStreamRequest) error {
	if a.ctx == nil {
		return domain.ErrRuntimeNotInitialized
	}
	if err := a.ensureAIProviderReady(); err != nil {
		return err
	}

	req.MessageID = strings.TrimSpace(req.MessageID)
	req.SessionID = strings.TrimSpace(req.SessionID)
	req.ProviderID = ai.NormalizeProviderID(req.ProviderID)
	req.Prompt = strings.TrimSpace(req.Prompt)
	req.ModelID = strings.TrimSpace(req.ModelID)

	if req.ModelID == "" {
		req.ModelID = ai.GetOrchestratorModel(req.ProviderID)
	}

	if req.MessageID == "" {
		req.MessageID = fmt.Sprintf("asst-%d", time.Now().UnixNano())
	}
	if req.Prompt == "" && len(req.History) == 0 {
		return errors.New("el mensaje no puede estar vacío")
	}

	messages := req.History
	if req.Prompt != "" {
		messages = append(messages, domain.ChatMessage{
			Role:    domain.ChatRoleUser,
			Content: req.Prompt,
		})
	}

	if len(req.MessageID) > 128 || len(req.SessionID) > 128 || len(req.UserMessageID) > 128 {
		return domain.ErrInvalidChat
	}
	if err := ai.ValidateChatRequest(req.ProviderID, req.ModelID, messages); err != nil {
		return err
	}
	streamCtx, finish, err := a.beginChatStream(req.MessageID, MaxTurnTimeout)
	if err != nil {
		return err
	}

	// Persistir mensaje del usuario inmediatamente si hay sesión activa y servicio disponible
	if a.sessionService != nil && req.SessionID != "" && req.Prompt != "" {
		userMsgID := strings.TrimSpace(req.UserMessageID)
		if userMsgID == "" {
			userMsgID = fmt.Sprintf("usr-%d", time.Now().UnixMilli())
		}
		userRecord := domain.ChatMessageRecord{
			ID:        userMsgID,
			SessionID: req.SessionID,
			Role:      domain.ChatRoleUser,
			Content:   req.Prompt,
			Status:    "done",
			CreatedAt: time.Now().UTC(),
		}
		if err := a.sessionService.SaveMessage(streamCtx, userRecord); err != nil {
			finish()
			return errors.New("no se pudo guardar el mensaje en el historial local")
		}
	}

	// 1. Emitir evento de estado inicial
	runtime.EventsEmit(a.ctx, "chat:stream", domain.ChatStreamEvent{
		SessionID:  req.SessionID,
		MessageID:  req.MessageID,
		Type:       domain.ChunkTypeStatus,
		StatusText: fmt.Sprintf("Conectando con %s...", req.ProviderID),
		ProviderID: req.ProviderID,
		ModelID:    req.ModelID,
	})

	// 2. Ejecutar el turno de agente (streaming + posibles iteraciones de tool-calling) en una
	// goroutine independiente para no bloquear el WebView / IPC.
	go a.runAgentTurn(streamCtx, finish, req, messages)

	return nil
}

// CancelChatStream cancela una respuesta en streaming activa por su ID de mensaje.
func (a *App) CancelChatStream(messageID string) bool {
	messageID = strings.TrimSpace(messageID)
	a.activeStreamsMu.Lock()
	cancel, found := a.activeStreams[messageID]

	a.activeStreamsMu.Unlock()

	if found && cancel != nil {
		cancel()
		return true
	}

	return false
}

// beginChatStream reserves a single slot until completion, including cancellation cleanup.
func (a *App) beginChatStream(messageID string, timeout time.Duration) (context.Context, func(), error) {
	a.activeStreamsMu.Lock()
	defer a.activeStreamsMu.Unlock()
	if a.ctx == nil {
		return nil, nil, domain.ErrRuntimeNotInitialized
	}
	if a.ctx.Err() != nil {
		return nil, nil, context.Canceled
	}
	if len(a.activeStreams) != 0 {
		return nil, nil, domain.ErrChatBusy
	}
	ctx, cancel := context.WithTimeout(a.ctx, timeout)
	if a.activeStreams == nil {
		a.activeStreams = make(map[string]context.CancelFunc)
	}
	a.activeStreams[messageID] = cancel
	a.streamsWG.Add(1)
	return ctx, func() {
		cancel()
		a.activeStreamsMu.Lock()
		delete(a.activeStreams, messageID)
		a.activeStreamsMu.Unlock()
		a.streamsWG.Done()
	}, nil
}
