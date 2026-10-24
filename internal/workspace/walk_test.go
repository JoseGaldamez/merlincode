package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"merlincode/internal/domain"
)

func newWalkTestProject(t *testing.T) (*Service, string) {
	t.Helper()
	svc := NewService()

	tempDir, err := os.MkdirTemp("", "merlin_walk_test_*")
	if err != nil {
		t.Fatalf("error creando tempDir: %v", err)
	}
	// Orden de registro importante en Windows: t.Cleanup se ejecuta en orden LIFO, así que
	// Close() (registrado después) debe liberar el handle del os.Root antes de que RemoveAll
	// intente borrar el directorio temporal.
	t.Cleanup(func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("no se pudo limpiar el directorio temporal: %v", err)
		}
	})
	t.Cleanup(func() { _ = svc.Close() })

	files := map[string]string{
		"main.go":                 "package main\n",
		"helper.go":               "package main\n",
		"README.md":               "# demo\n",
		"nested/inner.go":         "package nested\n",
		"nested/deep/deepest.go":  "package deep\n",
		"node_modules/pkg/dep.go": "package dep\n",
		".git/config":             "[core]\n",
	}
	for rel, content := range files {
		full := filepath.Join(tempDir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("error creando directorio para %q: %v", rel, err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("error escribiendo %q: %v", rel, err)
		}
	}

	if _, err := svc.SetActive(tempDir); err != nil {
		t.Fatalf("SetActive falló: %v", err)
	}
	return svc, tempDir
}

func TestWalkFilesSinProyectoActivo(t *testing.T) {
	svc := NewService()
	defer svc.Close()

	if _, err := svc.WalkFiles(nil, 0); !errors.Is(err, domain.ErrNoActiveProject) {
		t.Fatalf("esperado ErrNoActiveProject, obtenido: %v", err)
	}
}

func TestWalkFilesFiltraExtensionesYDirectoriosIgnorados(t *testing.T) {
	svc, _ := newWalkTestProject(t)

	got, err := svc.WalkFiles(map[string]bool{".go": true}, 0)
	if err != nil {
		t.Fatalf("WalkFiles falló: %v", err)
	}
	sort.Strings(got)

	want := []string{"helper.go", "main.go", "nested/deep/deepest.go", "nested/inner.go"}
	if len(got) != len(want) {
		t.Fatalf("esperado %d archivos .go, obtenido %d: %v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("esperado %v, obtenido %v", want, got)
		}
	}

	for _, p := range got {
		if p == "node_modules/pkg/dep.go" {
			t.Fatalf("WalkFiles no debió incluir archivos dentro de node_modules: %v", got)
		}
	}
}

func TestWalkFilesSinFiltroDevuelveTodasLasExtensiones(t *testing.T) {
	svc, _ := newWalkTestProject(t)

	got, err := svc.WalkFiles(nil, 0)
	if err != nil {
		t.Fatalf("WalkFiles falló: %v", err)
	}

	foundReadme := false
	for _, p := range got {
		if p == "README.md" {
			foundReadme = true
		}
		if p == ".git/config" {
			t.Fatalf("WalkFiles no debió descender dentro de .git: %v", got)
		}
	}
	if !foundReadme {
		t.Fatalf("esperado encontrar README.md con exts=nil, obtenido: %v", got)
	}
}

func TestWalkFilesRespetaLimit(t *testing.T) {
	svc, _ := newWalkTestProject(t)

	got, err := svc.WalkFiles(map[string]bool{".go": true}, 2)
	if err != nil {
		t.Fatalf("WalkFiles falló: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("esperado exactamente 2 resultados por el límite, obtenido %d: %v", len(got), got)
	}
}

func TestWalkFilesUsaSeparadorForwardSlash(t *testing.T) {
	svc, _ := newWalkTestProject(t)

	got, err := svc.WalkFiles(map[string]bool{".go": true}, 0)
	if err != nil {
		t.Fatalf("WalkFiles falló: %v", err)
	}
	foundNested := false
	for _, p := range got {
		if p == "nested/deep/deepest.go" {
			foundNested = true
		}
		if filepath.Separator != '/' && filepath.ToSlash(p) != p {
			t.Fatalf("ruta %q no está normalizada con '/'", p)
		}
	}
	if !foundNested {
		t.Fatalf("esperado encontrar nested/deep/deepest.go, obtenido: %v", got)
	}
}
