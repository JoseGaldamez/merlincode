package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"merlincode/internal/domain"
)

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

// awaitToolApproval registra una solicitud de aprobación pendiente, la emite hacia la UI, y bloquea
// esta goroutine hasta que RespondToToolApproval la resuelva, se agote el timeout, o el contexto del
// turno se cancele (p. ej. por CancelChatStream).
func (a *Service) awaitToolApproval(ctx context.Context, req domain.ChatStreamRequest, tc domain.ToolCall) (approvalResponse, error) {
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

	a.callbacks.Emit(domain.ChatStreamEvent{
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
		a.callbacks.Emit(domain.ChatStreamEvent{
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
func (a *Service) RespondToToolApproval(requestID string, approved bool, editedContent string) error {
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
