package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"merlincode/internal/domain"
	"merlincode/internal/workspace"
)

// applyUniqueReplace reemplaza old_string por new_string dentro de content, exigiendo que old_string
// aparezca exactamente una vez. Esto evita ediciones ambiguas: si el modelo no incluyó suficiente
// contexto para identificar un único punto del archivo, falla de forma explícita en lugar de adivinar.
func applyUniqueReplace(content, oldString, newString string) (string, error) {
	if oldString == "" {
		return "", domain.ErrEditOldStringEmpty
	}
	switch strings.Count(content, oldString) {
	case 0:
		return "", domain.ErrEditOldStringNotFound
	case 1:
		return strings.Replace(content, oldString, newString, 1), nil
	default:
		return "", domain.ErrEditOldStringNotUnique
	}
}

type editFileArgs struct {
	Path      string `json:"path"`
	OldString string `json:"old_string"`
	NewString string `json:"new_string"`
	// Content, cuando está presente (incluso vacío), es el contenido completo que el usuario dejó
	// tras editar manualmente la propuesta en la tarjeta de aprobación, y sustituye a old_string/new_string.
	Content *string `json:"content,omitempty"`
}

type editFileTool struct{ ws *workspace.Service }

func (editFileTool) Name() string { return "edit_file" }

func (editFileTool) Definition() domain.ToolDefinition {
	return domain.ToolDefinition{
		Name: "edit_file",
		Description: "Edita quirúrgicamente un archivo existente del proyecto activo sin reescribirlo completo. " +
			"Da old_string con el fragmento EXACTO del archivo (misma indentación, saltos de línea y mayúsculas) que quieres reemplazar, y new_string con el texto que debe quedar en su lugar. " +
			"old_string debe aparecer exactamente una vez en el archivo: si no aparece o aparece varias veces, la herramienta falla y debes usar read_file para revisar el contenido exacto y volver a intentarlo con más contexto (por ejemplo, incluyendo líneas antes o después). " +
			"Al llamar a esta herramienta Merlin Code muestra automáticamente una tarjeta con la vista previa del cambio y los botones Aprobar/Rechazar, y espera esa aprobación antes de escribir; no pidas confirmación verbal previa. " +
			"Para crear un archivo nuevo o reemplazar todo su contenido usa write_file en su lugar.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Ruta relativa del archivo dentro del proyecto activo.",
				},
				"old_string": map[string]any{
					"type":        "string",
					"description": "Fragmento exacto de texto a reemplazar; debe aparecer exactamente una vez en el archivo.",
				},
				"new_string": map[string]any{
					"type":        "string",
					"description": "Texto que reemplazará a old_string.",
				},
			},
			"required": []string{"path", "old_string", "new_string"},
		},
	}
}

func (editFileTool) RequiresApproval() bool { return true }

func (t editFileTool) Execute(_ context.Context, argsJSON string) (string, error) {
	var args editFileArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("argumentos inválidos: %w", err)
	}

	if args.Content != nil {
		if err := t.ws.WriteFile(args.Path, *args.Content); err != nil {
			return "", err
		}
		return fmt.Sprintf("Archivo '%s' editado correctamente.", strings.TrimSpace(args.Path)), nil
	}

	current, err := t.ws.ReadFile(args.Path)
	if err != nil {
		return "", err
	}
	updated, err := applyUniqueReplace(current, args.OldString, args.NewString)
	if err != nil {
		return "", err
	}
	if err := t.ws.WriteFile(args.Path, updated); err != nil {
		return "", err
	}
	return fmt.Sprintf("Archivo '%s' editado correctamente.", strings.TrimSpace(args.Path)), nil
}
