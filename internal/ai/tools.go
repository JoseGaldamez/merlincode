package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"merlincode/internal/domain"
	"merlincode/internal/workspace"
)

// ToolExecutor es el contrato que implementa cada herramienta expuesta al modelo.
// Los adaptadores concretos son delgados: solo deserializan sus argumentos y llaman
// directamente al método correspondiente de workspace.Service, sin duplicar lógica
// de sandboxing ni de filesystem.
type ToolExecutor interface {
	Name() string
	Definition() domain.ToolDefinition
	RequiresApproval() bool
	Execute(ctx context.Context, argsJSON string) (string, error)
}

// ToolRegistry agrupa las herramientas disponibles para el modelo en un turno de agente.
type ToolRegistry struct {
	tools  []ToolExecutor
	byName map[string]ToolExecutor
}

// NewToolRegistry construye el registro de herramientas envolviendo workspace.Service tal cual,
// sin agregar código nuevo de filesystem ni bypass del sandboxing existente (os.Root + ValidatePathInProject).
func NewToolRegistry(ws *workspace.Service) *ToolRegistry {
	executors := []ToolExecutor{
		readFileTool{ws: ws},
		listFilesTool{ws: ws},
		getTreeTool{ws: ws},
		writeFileTool{ws: ws},
	}
	byName := make(map[string]ToolExecutor, len(executors))
	for _, e := range executors {
		byName[e.Name()] = e
	}
	return &ToolRegistry{tools: executors, byName: byName}
}

// Definitions retorna las definiciones de todas las herramientas registradas, listas para
// pasarse a StreamChatWithTools y traducirse al wire format de cada proveedor.
func (r *ToolRegistry) Definitions() []domain.ToolDefinition {
	defs := make([]domain.ToolDefinition, 0, len(r.tools))
	for _, t := range r.tools {
		defs = append(defs, t.Definition())
	}
	return defs
}

// Get busca una herramienta registrada por nombre.
func (r *ToolRegistry) Get(name string) (ToolExecutor, bool) {
	t, ok := r.byName[name]
	return t, ok
}

type readFileArgs struct {
	Path string `json:"path"`
}

type readFileTool struct{ ws *workspace.Service }

func (readFileTool) Name() string { return "read_file" }

func (readFileTool) Definition() domain.ToolDefinition {
	return domain.ToolDefinition{
		Name:        "read_file",
		Description: "Lee el contenido de texto de un archivo dentro del proyecto activo, dada su ruta relativa.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Ruta relativa del archivo dentro del proyecto activo.",
				},
			},
			"required": []string{"path"},
		},
	}
}

func (readFileTool) RequiresApproval() bool { return false }

func (t readFileTool) Execute(_ context.Context, argsJSON string) (string, error) {
	var args readFileArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("argumentos inválidos: %w", err)
	}
	return t.ws.ReadFile(args.Path)
}

type listFilesTool struct{ ws *workspace.Service }

func (listFilesTool) Name() string { return "list_files" }

func (listFilesTool) Definition() domain.ToolDefinition {
	return domain.ToolDefinition{
		Name:        "list_files",
		Description: "Lista los archivos y carpetas directamente en la raíz del proyecto activo.",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}
}

func (listFilesTool) RequiresApproval() bool { return false }

func (t listFilesTool) Execute(_ context.Context, _ string) (string, error) {
	items, err := t.ws.ListFiles()
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(items)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type getTreeTool struct{ ws *workspace.Service }

func (getTreeTool) Name() string { return "get_tree" }

func (getTreeTool) Definition() domain.ToolDefinition {
	return domain.ToolDefinition{
		Name:        "get_tree",
		Description: "Obtiene la estructura jerárquica (árbol) de archivos y carpetas del proyecto activo.",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
	}
}

func (getTreeTool) RequiresApproval() bool { return false }

func (t getTreeTool) Execute(_ context.Context, _ string) (string, error) {
	tree, err := t.ws.GetTree()
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(tree)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type writeFileArgs struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type writeFileTool struct{ ws *workspace.Service }

func (writeFileTool) Name() string { return "write_file" }

func (writeFileTool) Definition() domain.ToolDefinition {
	return domain.ToolDefinition{
		Name:        "write_file",
		Description: "Escribe (crea o sobrescribe) el contenido de un archivo dentro del proyecto activo. Requiere aprobación explícita del usuario antes de ejecutarse.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{
					"type":        "string",
					"description": "Ruta relativa del archivo dentro del proyecto activo.",
				},
				"content": map[string]any{
					"type":        "string",
					"description": "Contenido completo que tendrá el archivo tras la escritura.",
				},
			},
			"required": []string{"path", "content"},
		},
	}
}

func (writeFileTool) RequiresApproval() bool { return true }

func (t writeFileTool) Execute(_ context.Context, argsJSON string) (string, error) {
	var args writeFileArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("argumentos inválidos: %w", err)
	}
	if err := t.ws.WriteFile(args.Path, args.Content); err != nil {
		return "", err
	}
	return fmt.Sprintf("Archivo '%s' escrito correctamente.", strings.TrimSpace(args.Path)), nil
}
