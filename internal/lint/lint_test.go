package lint

import (
	"context"
	"os"
	"testing"

	"merlincode/internal/workspace"
)

func newProject(t *testing.T) *workspace.Service {
	t.Helper()
	dir, err := os.MkdirTemp("", "merlin_lint_test_*")
	if err != nil {
		t.Fatalf("MkdirTemp falló: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	svc := workspace.NewService()
	t.Cleanup(func() { _ = svc.Close() })
	if _, err := svc.SetActive(dir); err != nil {
		t.Fatalf("SetActive falló: %v", err)
	}
	return svc
}

func TestCheck_UnsupportedExtension_DoesNotRun(t *testing.T) {
	ws := newProject(t)
	output, ran := Check(context.Background(), ws, "README.md")
	if ran || output != "" {
		t.Fatalf("esperado ran=false sin salida para extensión no soportada, obtenido ran=%v output=%q", ran, output)
	}
}

func TestCheck_Go_WithoutGoMod_DoesNotRun(t *testing.T) {
	ws := newProject(t)
	output, ran := Check(context.Background(), ws, "main.go")
	if ran || output != "" {
		t.Fatalf("esperado ran=false sin go.mod, obtenido ran=%v output=%q", ran, output)
	}
}

func TestCheck_Go_WithGoMod_RunsVet(t *testing.T) {
	ws := newProject(t)
	if err := ws.WriteFile("go.mod", "module merlinlinttest\n\ngo 1.22\n"); err != nil {
		t.Fatalf("WriteFile(go.mod) falló: %v", err)
	}
	if err := ws.WriteFile("main.go", "package main\n\nfunc main() {}\n"); err != nil {
		t.Fatalf("WriteFile(main.go) falló: %v", err)
	}
	output, ran := Check(context.Background(), ws, "main.go")
	if !ran {
		t.Fatalf("esperado ran=true con go.mod presente")
	}
	if output != "" {
		t.Fatalf("esperado archivo Go válido sin diagnósticos, obtenido: %q", output)
	}
}

func TestCheck_Go_VetReportsBrokenCode(t *testing.T) {
	ws := newProject(t)
	if err := ws.WriteFile("go.mod", "module merlinlinttest\n\ngo 1.22\n"); err != nil {
		t.Fatalf("WriteFile(go.mod) falló: %v", err)
	}
	// Printf con verbo y argumento que no coinciden: vet lo detecta de forma confiable.
	broken := "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Printf(\"%d\\n\", \"no es un entero\")\n}\n"
	if err := ws.WriteFile("main.go", broken); err != nil {
		t.Fatalf("WriteFile(main.go) falló: %v", err)
	}
	output, ran := Check(context.Background(), ws, "main.go")
	if !ran {
		t.Fatalf("esperado ran=true con go.mod presente")
	}
	if output == "" {
		t.Fatalf("esperado diagnóstico de go vet para código roto, obtenido salida vacía")
	}
}

func TestNearestTSConfigDir_WalksUpToProjectRoot(t *testing.T) {
	ws := newProject(t)
	if err := ws.WriteFile("frontend/tsconfig.json", "{}"); err != nil {
		t.Fatalf("WriteFile(tsconfig.json) falló: %v", err)
	}
	if got := nearestTSConfigDir(ws, "frontend/src/features/chat/Chat.tsx"); got != "frontend" {
		t.Fatalf("esperado 'frontend', obtenido %q", got)
	}
}

func TestNearestTSConfigDir_NoneFound(t *testing.T) {
	ws := newProject(t)
	if got := nearestTSConfigDir(ws, "frontend/src/features/chat/Chat.tsx"); got != "" {
		t.Fatalf("esperado cadena vacía sin tsconfig.json, obtenido %q", got)
	}
}

func TestCheck_TypeScript_WithoutNodeModules_DoesNotRun(t *testing.T) {
	ws := newProject(t)
	if err := ws.WriteFile("frontend/tsconfig.json", "{}"); err != nil {
		t.Fatalf("WriteFile(tsconfig.json) falló: %v", err)
	}
	output, ran := Check(context.Background(), ws, "frontend/src/App.tsx")
	if ran || output != "" {
		t.Fatalf("esperado ran=false sin typescript instalado en node_modules, obtenido ran=%v output=%q", ran, output)
	}
}

func TestSanitizedEnv_StripsSecrets(t *testing.T) {
	t.Setenv("MERLIN_TEST_API_KEY", "super-secreto")
	t.Setenv("MERLIN_TEST_TOKEN", "otro-secreto")
	t.Setenv("MERLIN_TEST_HARMLESS", "visible")
	env := sanitizedEnv()
	for _, item := range env {
		if containsPrefix(item, "MERLIN_TEST_API_KEY=") || containsPrefix(item, "MERLIN_TEST_TOKEN=") {
			t.Fatalf("variable secreta no debió propagarse: %q", item)
		}
	}
	found := false
	for _, item := range env {
		if containsPrefix(item, "MERLIN_TEST_HARMLESS=") {
			found = true
		}
	}
	if !found {
		t.Fatalf("variable inofensiva debió propagarse intacta")
	}
}

func containsPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
