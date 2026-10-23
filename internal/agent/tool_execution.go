package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"merlincode/internal/domain"
)

// ExecuteTools ejecuta secuencialmente las herramientas solicitadas por el modelo en una iteración,
// bloqueando en awaitToolApproval para las que requieren aprobación humana (una tarjeta a la vez).
func (a *Service) ExecuteTools(ctx context.Context, req domain.ChatStreamRequest, registry *ToolRegistry, calls []domain.ToolCall, trace *[]domain.ToolTraceEntry) (results []domain.ToolResult, canceled bool) {
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
				if errors.Is(err, domain.ErrEditOldStringNotFound) || errors.Is(err, domain.ErrEditOldStringNotUnique) || errors.Is(err, domain.ErrEditOldStringEmpty) {
					msg := sanitizeToolErrorForModel(err)
					results = append(results, domain.ToolResult{ToolCallID: tc.ID, Name: tc.Name, Content: msg, IsError: true})
					*trace = append(*trace, domain.ToolTraceEntry{ToolName: tc.Name, Arguments: tc.Arguments, Result: msg, IsError: true})
					continue
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
		if (tc.Name == "write_file" || tc.Name == "edit_file") && !isError {
			a.callbacks.WorkspaceChanged()
		}
		resultEvent := domain.ChatStreamEvent{SessionID: req.SessionID, MessageID: req.MessageID, Type: domain.ChunkTypeToolResult, ToolCallID: tc.ID, ToolName: tc.Name}
		if isError {
			resultEvent.Error = content
		}
		a.callbacks.Emit(resultEvent)
		modelContent := content
		if tc.Name == "propose_app_run" && !isError {
			var proposals []domain.RunProposal
			if json.Unmarshal([]byte(content), &proposals) == nil {
				a.callbacks.Emit(domain.ChatStreamEvent{SessionID: req.SessionID, MessageID: req.MessageID, Type: "run_proposed", RunProposals: proposals})
				guidance := "Merlin Code preparó estas propuestas y mostró sus tarjetas con botón Play. Confirma el comando y pide al usuario que pulse Play para iniciar la aplicación desde Merlin Code. No afirmes que no puedes ejecutarla ni lo envíes a una terminal externa. Todavía no se inició ningún proceso; el clic del usuario lo inicia."
				if len(proposals) > 1 {
					guidance += " Hay varias alternativas: explica brevemente cada comando, para qué sirve y cuándo elegirlo, usando el campo reason y los archivos inspeccionados. Compara desarrollo, vista previa o inicio de producción cuando corresponda; menciona los requisitos de compilación previa indicados en reason sin afirmar que la compilación ya existe si no lo has comprobado. Recomienda una opción acorde a lo pedido; para trabajar y ver cambios suele corresponder desarrollo. Si las opciones ejecutan distintas aplicaciones, entradas o puertos, explica esas diferencias verificables. Son alternativas, no pasos que deban ejecutarse todos. No deduzcas el modo únicamente por el nombre del script: start puede iniciar desarrollo."
				}
				modelContent = guidance + "\n\n" + content
			}
		}
		results = append(results, domain.ToolResult{ToolCallID: tc.ID, Name: tc.Name, Content: modelContent, IsError: isError})
		*trace = append(*trace, domain.ToolTraceEntry{ToolName: tc.Name, Arguments: tc.Arguments, Result: content, IsError: isError, Approved: approvedPtr})
	}
	return results, false
}

// sanitizeToolErrorForModel traduce errores internos de una herramienta a mensajes accionables para
// el modelo, sin filtrar rutas absolutas ni detalles internos — audiencia distinta de sanitizeAIProviderError.
func sanitizeToolErrorForModel(err error) string {
	var runError *domain.RunPolicyError
	if errors.As(err, &runError) {
		return runError.Message
	}
	switch {
	case errors.Is(err, domain.ErrAccessDenied):
		return "Error: la ruta especificada está fuera del proyecto activo. Usa una ruta relativa dentro del proyecto."
	case errors.Is(err, domain.ErrFileTooLarge):
		return "Error: el archivo excede el límite de tamaño permitido (4 MiB)."
	case errors.Is(err, domain.ErrNoActiveProject):
		return "Error: no hay ningún proyecto activo abierto para acceder a archivos."
	case errors.Is(err, domain.ErrFolderNotExist):
		return "Error: la ruta especificada no existe."
	case errors.Is(err, domain.ErrEditOldStringNotFound):
		return "Error: old_string no se encontró en el archivo. Usa read_file para revisar el contenido exacto (indentación y saltos de línea incluidos) y vuelve a intentar edit_file."
	case errors.Is(err, domain.ErrEditOldStringNotUnique):
		return "Error: old_string aparece más de una vez en el archivo. Agrega más contexto (líneas antes o después) para que identifique un único lugar."
	case errors.Is(err, domain.ErrEditOldStringEmpty):
		return "Error: old_string no puede estar vacío."
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
