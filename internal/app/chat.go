package app

import (
	"context"
	"errors"
	"fmt"
	"log"
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
	req.ProviderID = strings.TrimSpace(req.ProviderID)
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
		if err := a.sessionService.SaveMessage(context.Background(), userRecord); err != nil {
			log.Printf("[ChatStream] Error guardando mensaje de usuario en SQLite: %v", err)
		}
	}

	streamCtx, cancel := context.WithCancel(a.ctx)
	a.activeStreamsMu.Lock()
	a.activeStreams[req.MessageID] = cancel
	a.activeStreamsMu.Unlock()

	log.Printf("[ChatStream] Iniciando petición (messageID=%s, provider=%s, model=%s)", req.MessageID, req.ProviderID, req.ModelID)

	// 1. Emitir evento de estado inicial
	runtime.EventsEmit(a.ctx, "chat:stream", domain.ChatStreamEvent{
		SessionID:  req.SessionID,
		MessageID:  req.MessageID,
		Type:       domain.ChunkTypeStatus,
		StatusText: fmt.Sprintf("Conectando con %s...", req.ProviderID),
		ProviderID: req.ProviderID,
		ModelID:    req.ModelID,
	})

	// 2. Ejecutar streaming en una goroutine independiente para no bloquear el WebView / IPC
	go func() {
		defer func() {
			a.activeStreamsMu.Lock()
			delete(a.activeStreams, req.MessageID)
			a.activeStreamsMu.Unlock()
		}()

		streamStartTime := time.Now()
		var fullContent strings.Builder
		var fullThinking strings.Builder

		result, err := a.aiProviderService.StreamChat(
			streamCtx,
			req.ProviderID,
			req.ModelID,
			messages,
			func(chunk domain.StreamChunk) error {
				if chunk.Type == domain.ChunkTypeContent {
					fullContent.WriteString(chunk.Text)
				} else if chunk.Type == domain.ChunkTypeThinking {
					fullThinking.WriteString(chunk.Thinking)
				}

				event := domain.ChatStreamEvent{
					SessionID:  req.SessionID,
					MessageID:  req.MessageID,
					Type:       chunk.Type,
					Content:    chunk.Text,
					Thinking:   chunk.Thinking,
					ProviderID: req.ProviderID,
					ModelID:    req.ModelID,
				}
				runtime.EventsEmit(a.ctx, "chat:stream", event)
				return nil
			},
		)

		if err != nil {
			elapsedSec := max(1, int(time.Since(streamStartTime).Seconds()))
			if errors.Is(err, context.Canceled) || streamCtx.Err() != nil {
				log.Printf("[ChatStream] Stream cancelado por usuario (messageID=%s)", req.MessageID)
				runtime.EventsEmit(a.ctx, "chat:stream", domain.ChatStreamEvent{
					SessionID:  req.SessionID,
					MessageID:  req.MessageID,
					Type:       domain.ChunkTypeDone,
					StatusText: "Generación detenida.",
					ProviderID: req.ProviderID,
				})

				if a.sessionService != nil && req.SessionID != "" {
					stoppedContent := fullContent.String()
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
						ThoughtChain:    fullThinking.String(),
						ProviderID:      req.ProviderID,
						ModelID:         req.ModelID,
						DurationSeconds: elapsedSec,
						Status:          "done",
						CreatedAt:       time.Now().UTC(),
					}
					_ = a.sessionService.SaveMessage(context.Background(), asstRecord)
				}
				return
			}

			log.Printf("[ChatStream] Error durante stream (messageID=%s): %v", req.MessageID, err)
			runtime.EventsEmit(a.ctx, "chat:stream", domain.ChatStreamEvent{
				SessionID:  req.SessionID,
				MessageID:  req.MessageID,
				Type:       domain.ChunkTypeError,
				Error:      err.Error(),
				ProviderID: req.ProviderID,
			})

			if a.sessionService != nil && req.SessionID != "" {
				errContent := fullContent.String()
				if errContent != "" {
					errContent += fmt.Sprintf("\n\n[Error: %s]", err.Error())
				} else {
					errContent = fmt.Sprintf("Error: %s", err.Error())
				}
				asstRecord := domain.ChatMessageRecord{
					ID:              req.MessageID,
					SessionID:       req.SessionID,
					Role:            domain.ChatRoleAssistant,
					Content:         errContent,
					ThoughtChain:    fullThinking.String(),
					ProviderID:      req.ProviderID,
					ModelID:         req.ModelID,
					DurationSeconds: elapsedSec,
					Status:          "error",
					CreatedAt:       time.Now().UTC(),
				}
				_ = a.sessionService.SaveMessage(context.Background(), asstRecord)
			}
			return
		}

		elapsedSec := max(1, int(time.Since(streamStartTime).Seconds()))
		log.Printf("[ChatStream] Stream completado con éxito (messageID=%s, model=%s, promptTokens=%d, compTokens=%d, duration=%ds)",
			req.MessageID, result.Model, result.TokensPrompt, result.TokensCompletion, elapsedSec)

		// Guardar respuesta del asistente en SQLite
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
			}
			if err := a.sessionService.SaveMessage(context.Background(), asstRecord); err != nil {
				log.Printf("[ChatStream] Error guardando mensaje de asistente en SQLite: %v", err)
			}
		}

		// 3. Emitir evento de finalización exitosa
		runtime.EventsEmit(a.ctx, "chat:stream", domain.ChatStreamEvent{
			SessionID:        req.SessionID,
			MessageID:        req.MessageID,
			Type:             domain.ChunkTypeDone,
			ProviderID:       req.ProviderID,
			ModelID:          result.Model,
			TokensPrompt:     result.TokensPrompt,
			TokensCompletion: result.TokensCompletion,
		})
	}()

	return nil
}

// CancelChatStream cancela una respuesta en streaming activa por su ID de mensaje.
func (a *App) CancelChatStream(messageID string) bool {
	messageID = strings.TrimSpace(messageID)
	a.activeStreamsMu.Lock()
	cancel, found := a.activeStreams[messageID]
	if found {
		delete(a.activeStreams, messageID)
	}
	a.activeStreamsMu.Unlock()

	if found && cancel != nil {
		log.Printf("[ChatStream] Cancelando stream activo (messageID=%s)...", messageID)
		cancel()
		return true
	}

	log.Printf("[ChatStream] CancelChatStream: messageID=%s no estaba activo o ya finalizó", messageID)
	return false
}
