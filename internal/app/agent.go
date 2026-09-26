package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"merlincode/internal/ai"
	"merlincode/internal/domain"
)

// maxToolIterations limita cuántas rondas de tool_call/tool_result puede tener un solo turno de agente.
const maxToolIterations = 8

// MaxTurnTimeout cubre el turno de agente completo, incluyendo esperas de aprobación humana —
// distinto de ai.ChatTimeout, que cubre solo el round-trip individual con el proveedor.
const MaxTurnTimeout = 15 * time.Minute

// toolApprovalTimeout es cuánto se espera una respuesta humana antes de considerar la escritura rechazada.
const toolApprovalTimeout = 10 * time.Minute

// approvalResponse es lo que RespondToToolApproval envía de vuelta a la goroutine que espera en
// awaitToolApproval — approved indica si el usuario aprobó, y editedContent (cuando approved es true)
// es el contenido final a escribir, que puede diferir del que propuso el modelo si el usuario lo editó
// en la tarjeta de aprobación antes de confirmar.
type approvalResponse struct {
	approved       bool
	editedContent  string
	hasEditedValue bool
}

// pendingApproval representa una escritura de archivo esperando aprobación explícita del usuario en la UI.
type pendingApproval struct {
	resultCh chan approvalResponse
}

// runAgentTurn ejecuta el bucle completo de un turno de agente (asistente -> tool_call -> tool_result
// -> asistente...) dentro del mismo slot de stream reservado por beginChatStream, sin abrir un segundo
// stream concurrente. Los proveedores que no implementan ai.ToolCaller (Google/DeepSeek en el Incremento 1)
// siguen funcionando sin cambios vía el fallback automático de ai.Service.StreamChatWithTools.
func (a *App) runAgentTurn(ctx context.Context, finish func(), req domain.ChatStreamRequest, messages []domain.ChatMessage) {
	defer finish()

	streamStartTime := time.Now()
	registry := ai.NewToolRegistry(a.workspaceService)
	tools := registry.Definitions()

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
				runtime.EventsEmit(a.ctx, "chat:stream", event)
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

		toolResults, canceled := a.executeToolCalls(ctx, req, registry, result.ToolCalls, &toolTrace)
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

// executeToolCalls ejecuta secuencialmente las herramientas solicitadas por el modelo en una iteración,
// bloqueando en awaitToolApproval para las que requieren aprobación humana (una tarjeta a la vez).
func (a *App) executeToolCalls(ctx context.Context, req domain.ChatStreamRequest, registry *ai.ToolRegistry, calls []domain.ToolCall, trace *[]domain.ToolTraceEntry) (results []domain.ToolResult, canceled bool) {
	for _, tc := range calls {
		executor, ok := registry.Get(tc.Name)
		if !ok {
			msg := fmt.Sprintf("herramienta desconocida: %s", tc.Name)
			results = append(results, domain.ToolResult{ToolCallID: tc.ID, Name: tc.Name, Content: msg, IsError: true})
			*trace = append(*trace, domain.ToolTraceEntry{ToolName: tc.Name, Arguments: tc.Arguments, Result: msg, IsError: true})
			continue
		}

		var approvedPtr *bool
		if executor.RequiresApproval() {
			resp, err := a.awaitToolApproval(ctx, req, tc)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return results, true
				}
				resp.approved = false
			}
			approvedPtr = &resp.approved
			if !resp.approved {
				msg := "el usuario rechazó esta escritura de archivo"
				results = append(results, domain.ToolResult{ToolCallID: tc.ID, Name: tc.Name, Content: msg, IsError: true})
				*trace = append(*trace, domain.ToolTraceEntry{ToolName: tc.Name, Arguments: tc.Arguments, Result: msg, IsError: true, Approved: approvedPtr})
				continue
			}
			if resp.hasEditedValue {
				tc.Arguments = withEditedContent(tc.Arguments, resp.editedContent)
			}
		}

		content, err := executor.Execute(ctx, tc.Arguments)
		isError := false
		if err != nil {
			content = sanitizeToolErrorForModel(err)
			isError = true
		}
		results = append(results, domain.ToolResult{ToolCallID: tc.ID, Name: tc.Name, Content: content, IsError: isError})
		*trace = append(*trace, domain.ToolTraceEntry{ToolName: tc.Name, Arguments: tc.Arguments, Result: content, IsError: isError, Approved: approvedPtr})
	}
	return results, false
}

// awaitToolApproval registra una solicitud de aprobación pendiente, la emite hacia la UI, y bloquea
// esta goroutine hasta que RespondToToolApproval la resuelva, se agote el timeout, o el contexto del
// turno se cancele (p. ej. por CancelChatStream).
func (a *App) awaitToolApproval(ctx context.Context, req domain.ChatStreamRequest, tc domain.ToolCall) (approvalResponse, error) {
	requestID := fmt.Sprintf("approval-%s-%d", tc.ID, time.Now().UnixNano())

	var args struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	_ = json.Unmarshal([]byte(tc.Arguments), &args)

	oldContent, _ := a.workspaceService.ReadFile(args.Path)

	approvalReq := domain.ToolApprovalRequest{
		RequestID:  requestID,
		ToolName:   tc.Name,
		Path:       args.Path,
		NewContent: args.Content,
		OldContent: oldContent,
		ExpiresAt:  time.Now().Add(toolApprovalTimeout).UnixMilli(),
	}

	pending := &pendingApproval{resultCh: make(chan approvalResponse, 1)}
	a.pendingApprovalsMu.Lock()
	a.pendingApprovals[requestID] = pending
	a.pendingApprovalsMu.Unlock()

	defer func() {
		a.pendingApprovalsMu.Lock()
		delete(a.pendingApprovals, requestID)
		a.pendingApprovalsMu.Unlock()
	}()

	runtime.EventsEmit(a.ctx, "chat:stream", domain.ChatStreamEvent{
		SessionID:       req.SessionID,
		MessageID:       req.MessageID,
		Type:            domain.ChunkTypeToolApprovalRequired,
		ProviderID:      req.ProviderID,
		ModelID:         req.ModelID,
		ToolName:        tc.Name,
		ToolCallID:      tc.ID,
		ApprovalRequest: &approvalReq,
	})

	timer := time.NewTimer(toolApprovalTimeout)
	defer timer.Stop()

	select {
	case resp := <-pending.resultCh:
		runtime.EventsEmit(a.ctx, "chat:stream", domain.ChatStreamEvent{
			SessionID:  req.SessionID,
			MessageID:  req.MessageID,
			Type:       domain.ChunkTypeToolApprovalResolved,
			ProviderID: req.ProviderID,
			ModelID:    req.ModelID,
			ToolName:   tc.Name,
			ToolCallID: tc.ID,
		})
		return resp, nil
	case <-timer.C:
		return approvalResponse{approved: false}, domain.ErrToolApprovalTimedOut
	case <-ctx.Done():
		return approvalResponse{approved: false}, ctx.Err()
	}
}

// RespondToToolApproval resuelve una solicitud de aprobación pendiente emitida por awaitToolApproval.
// editedContent es el contenido final a escribir cuando approved es true; si el usuario no editó nada
// en la tarjeta de aprobación, el frontend envía el mismo newContent que propuso el modelo.
func (a *App) RespondToToolApproval(requestID string, approved bool, editedContent string) error {
	requestID = strings.TrimSpace(requestID)
	a.pendingApprovalsMu.Lock()
	pending, ok := a.pendingApprovals[requestID]
	a.pendingApprovalsMu.Unlock()
	if !ok {
		return domain.ErrApprovalRequestNotFound
	}
	resp := approvalResponse{approved: approved}
	if approved {
		resp.hasEditedValue = true
		resp.editedContent = editedContent
	}
	select {
	case pending.resultCh <- resp:
	default:
	}
	return nil
}

// withEditedContent reemplaza el campo "content" de los argumentos JSON crudos de una tool call de
// escritura por el contenido final que el usuario confirmó en la tarjeta de aprobación.
func withEditedContent(argsJSON string, editedContent string) string {
	var args map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil || args == nil {
		args = map[string]any{}
	}
	args["content"] = editedContent
	out, err := json.Marshal(args)
	if err != nil {
		return argsJSON
	}
	return string(out)
}

// sanitizeToolErrorForModel traduce errores internos de una herramienta a mensajes accionables para
// el modelo, sin filtrar rutas absolutas ni detalles internos — audiencia distinta de sanitizeAIProviderError.
func sanitizeToolErrorForModel(err error) string {
	switch {
	case errors.Is(err, domain.ErrAccessDenied):
		return "Error: la ruta especificada está fuera del proyecto activo. Usa una ruta relativa dentro del proyecto."
	case errors.Is(err, domain.ErrFileTooLarge):
		return "Error: el archivo excede el límite de tamaño permitido (4 MiB)."
	case errors.Is(err, domain.ErrNoActiveProject):
		return "Error: no hay ningún proyecto activo abierto para acceder a archivos."
	case errors.Is(err, domain.ErrFolderNotExist):
		return "Error: la ruta especificada no existe."
	default:
		return "Error: no se pudo ejecutar la herramienta solicitada."
	}
}

// summarizeToolArgs produce un resumen corto y seguro de los argumentos de una tool call para mostrar
// en la UI mientras se ejecuta, sin volcar contenidos completos de archivos.
func summarizeToolArgs(argsJSON string) string {
	var probe struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &probe); err == nil && probe.Path != "" {
		return probe.Path
	}
	if len(argsJSON) > 80 {
		return argsJSON[:80] + "..."
	}
	return argsJSON
}

// handleAgentTurnError maneja el error final de un turno de agente (tras agotar reintentos de la capa
// de streaming), persistiendo lo generado hasta el momento y notificando a la UI.
func (a *App) handleAgentTurnError(req domain.ChatStreamRequest, err error, streamStartTime time.Time, content string, thinking string, toolTrace []domain.ToolTraceEntry) {
	elapsedSec := max(1, int(time.Since(streamStartTime).Seconds()))

	if errors.Is(err, context.Canceled) {
		a.handleAgentTurnCanceled(req, streamStartTime, content, thinking, toolTrace)
		return
	}

	runtime.LogErrorf(a.ctx, "turno de agente falló (provider=%s, model=%s): %v", req.ProviderID, req.ModelID, err)

	sanitized := sanitizeAIProviderError(err)
	runtime.EventsEmit(a.ctx, "chat:stream", domain.ChatStreamEvent{
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
func (a *App) handleAgentTurnCanceled(req domain.ChatStreamRequest, streamStartTime time.Time, content string, thinking string, toolTrace []domain.ToolTraceEntry) {
	elapsedSec := max(1, int(time.Since(streamStartTime).Seconds()))

	runtime.EventsEmit(a.ctx, "chat:stream", domain.ChatStreamEvent{
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
func (a *App) finishAgentTurn(req domain.ChatStreamRequest, result *domain.ChatCompletionResult, streamStartTime time.Time, toolTrace []domain.ToolTraceEntry) {
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

	runtime.EventsEmit(a.ctx, "chat:stream", domain.ChatStreamEvent{
		SessionID:        req.SessionID,
		MessageID:        req.MessageID,
		Type:             domain.ChunkTypeDone,
		ProviderID:       req.ProviderID,
		ModelID:          result.Model,
		TokensPrompt:     result.TokensPrompt,
		TokensCompletion: result.TokensCompletion,
	})
}
