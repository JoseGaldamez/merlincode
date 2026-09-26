package ai

import (
	"context"
	"errors"
	"os"
	"testing"

	"merlincode/internal/domain"
	"merlincode/internal/workspace"
)

func newTestWorkspaceForTools(t *testing.T) *workspace.Service {
	t.Helper()
	ws := workspace.NewService()
	tempDir, err := os.MkdirTemp("", "merlin_ai_tools_test_*")
	if err != nil {
		t.Fatalf("error creando tempDir: %v", err)
	}
	t.Cleanup(func() {
		_ = ws.Close()
		_ = os.RemoveAll(tempDir)
	})
	if _, err := ws.SetActive(tempDir); err != nil {
		t.Fatalf("SetActive falló: %v", err)
	}
	return ws
}

func TestToolRegistry_DefinitionsAndGet(t *testing.T) {
	registry := NewToolRegistry(newTestWorkspaceForTools(t))

	defs := registry.Definitions()
	if len(defs) != 4 {
		t.Fatalf("se esperaban 4 definiciones, obtenidas: %d", len(defs))
	}

	wantApproval := map[string]bool{
		"read_file":  false,
		"list_files": false,
		"get_tree":   false,
		"write_file": true,
	}
	seen := map[string]bool{}
	for name, expectApproval := range wantApproval {
		tool, ok := registry.Get(name)
		if !ok {
			t.Fatalf("herramienta %q no encontrada en el registro", name)
		}
		if tool.Name() != name {
			t.Errorf("Name() inconsistente para %q: obtenido %q", name, tool.Name())
		}
		if tool.RequiresApproval() != expectApproval {
			t.Errorf("RequiresApproval() de %q: esperado %v, obtenido %v", name, expectApproval, tool.RequiresApproval())
		}
		seen[name] = true
	}
	if len(seen) != len(wantApproval) {
		t.Fatalf("no se verificaron todas las herramientas esperadas: %+v", seen)
	}

	if _, ok := registry.Get("no_existe"); ok {
		t.Fatal("se esperaba que una herramienta desconocida no se encontrara")
	}
}

func TestReadFileTool_Execute(t *testing.T) {
	ws := newTestWorkspaceForTools(t)
	if err := ws.WriteFile("doc.txt", "hola mundo"); err != nil {
		t.Fatalf("preparación falló: %v", err)
	}
	tool := readFileTool{ws: ws}

	content, err := tool.Execute(context.Background(), `{"path":"doc.txt"}`)
	if err != nil {
		t.Fatalf("Execute falló: %v", err)
	}
	if content != "hola mundo" {
		t.Fatalf("contenido inesperado: %q", content)
	}

	if _, err := tool.Execute(context.Background(), `{no es json`); err == nil {
		t.Fatal("se esperaba error con argumentos JSON inválidos")
	}

	if _, err := tool.Execute(context.Background(), `{"path":"../fuera.txt"}`); !errors.Is(err, domain.ErrAccessDenied) {
		t.Fatalf("esperado ErrAccessDenied por path traversal, obtenido: %v", err)
	}
}

func TestWriteFileTool_Execute(t *testing.T) {
	ws := newTestWorkspaceForTools(t)
	tool := writeFileTool{ws: ws}

	msg, err := tool.Execute(context.Background(), `{"path":"nuevo.txt","content":"contenido nuevo"}`)
	if err != nil {
		t.Fatalf("Execute falló: %v", err)
	}
	if msg == "" {
		t.Fatal("se esperaba un mensaje de confirmación")
	}

	got, err := ws.ReadFile("nuevo.txt")
	if err != nil || got != "contenido nuevo" {
		t.Fatalf("el archivo no se escribió correctamente: content=%q err=%v", got, err)
	}

	if _, err := tool.Execute(context.Background(), `{"path":"../fuera.txt","content":"x"}`); !errors.Is(err, domain.ErrAccessDenied) {
		t.Fatalf("esperado ErrAccessDenied por path traversal, obtenido: %v", err)
	}
}

func TestListFilesTool_Execute(t *testing.T) {
	ws := newTestWorkspaceForTools(t)
	if err := ws.WriteFile("a.txt", "x"); err != nil {
		t.Fatalf("preparación falló: %v", err)
	}
	tool := listFilesTool{ws: ws}

	out, err := tool.Execute(context.Background(), "")
	if err != nil {
		t.Fatalf("Execute falló: %v", err)
	}
	if out == "" || out == "null" {
		t.Fatalf("salida inesperada: %q", out)
	}
}

func TestGetTreeTool_Execute(t *testing.T) {
	ws := newTestWorkspaceForTools(t)
	if err := ws.WriteFile("dir/a.txt", "x"); err != nil {
		t.Fatalf("preparación falló: %v", err)
	}
	tool := getTreeTool{ws: ws}

	out, err := tool.Execute(context.Background(), "")
	if err != nil {
		t.Fatalf("Execute falló: %v", err)
	}
	if out == "" || out == "null" {
		t.Fatalf("salida inesperada: %q", out)
	}
}

func TestListFilesAndGetTreeTools_NoActiveProject(t *testing.T) {
	ws := workspace.NewService()
	defer ws.Close()

	if _, err := (listFilesTool{ws: ws}).Execute(context.Background(), ""); !errors.Is(err, domain.ErrNoActiveProject) {
		t.Fatalf("esperado ErrNoActiveProject, obtenido: %v", err)
	}
	if _, err := (getTreeTool{ws: ws}).Execute(context.Background(), ""); !errors.Is(err, domain.ErrNoActiveProject) {
		t.Fatalf("esperado ErrNoActiveProject, obtenido: %v", err)
	}
}
