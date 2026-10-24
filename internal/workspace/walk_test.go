package workspace

import (
	"errors"
	"os"
	"testing"

	"merlincode/internal/domain"
)

func TestWalkFiles_NoActiveProject(t *testing.T) {
	svc := NewService()
	defer svc.Close()
	if _, err := svc.WalkFiles(nil, 0); !errors.Is(err, domain.ErrNoActiveProject) {
		t.Fatalf("esperado ErrNoActiveProject, obtenido: %v", err)
	}
}

func TestWalkFiles_FiltersExtensionsAndIgnoredDirs(t *testing.T) {
	svc := NewService()
	defer svc.Close()

	tempDir, err := os.MkdirTemp("", "merlin_walk_test_*")
	if err != nil {
		t.Fatalf("MkdirTemp falló: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })

	if _, err := svc.SetActive(tempDir); err != nil {
		t.Fatalf("SetActive falló: %v", err)
	}

	if err := svc.WriteFile("a.go", "package pkg\n"); err != nil {
		t.Fatalf("WriteFile falló: %v", err)
	}
	if err := svc.WriteFile("README.md", "# hola"); err != nil {
		t.Fatalf("WriteFile falló: %v", err)
	}
	if err := svc.WriteFile("node_modules/pkg/index.go", "package pkg\n"); err != nil {
		t.Fatalf("WriteFile falló: %v", err)
	}
	if err := svc.WriteFile("src/b.ts", "export const x = 1\n"); err != nil {
		t.Fatalf("WriteFile falló: %v", err)
	}

	got, err := svc.WalkFiles(map[string]bool{".go": true, ".ts": true}, 0)
	if err != nil {
		t.Fatalf("WalkFiles falló: %v", err)
	}

	want := map[string]bool{"a.go": true, "src/b.ts": true}
	if len(got) != len(want) {
		t.Fatalf("esperado %d archivos, obtenido %d: %v", len(want), len(got), got)
	}
	for _, f := range got {
		if !want[f] {
			t.Fatalf("archivo inesperado en el resultado: %q (lista completa: %v)", f, got)
		}
	}
}

func TestWalkFiles_RespectsLimit(t *testing.T) {
	svc := NewService()
	defer svc.Close()

	tempDir, err := os.MkdirTemp("", "merlin_walk_test_limit_*")
	if err != nil {
		t.Fatalf("MkdirTemp falló: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })

	if _, err := svc.SetActive(tempDir); err != nil {
		t.Fatalf("SetActive falló: %v", err)
	}

	for _, name := range []string{"a.go", "b.go", "c.go"} {
		if err := svc.WriteFile(name, "package pkg\n"); err != nil {
			t.Fatalf("WriteFile falló: %v", err)
		}
	}

	got, err := svc.WalkFiles(map[string]bool{".go": true}, 2)
	if err != nil {
		t.Fatalf("WalkFiles falló: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("esperado 2 archivos por el límite, obtenido %d: %v", len(got), got)
	}
}

func TestWalkFiles_NilExtsMatchesEverything(t *testing.T) {
	svc := NewService()
	defer svc.Close()

	tempDir, err := os.MkdirTemp("", "merlin_walk_test_all_*")
	if err != nil {
		t.Fatalf("MkdirTemp falló: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })

	if _, err := svc.SetActive(tempDir); err != nil {
		t.Fatalf("SetActive falló: %v", err)
	}
	if err := svc.WriteFile("a.go", "package pkg\n"); err != nil {
		t.Fatalf("WriteFile falló: %v", err)
	}
	if err := svc.WriteFile("README.md", "# hola"); err != nil {
		t.Fatalf("WriteFile falló: %v", err)
	}

	got, err := svc.WalkFiles(nil, 0)
	if err != nil {
		t.Fatalf("WalkFiles falló: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("esperado 2 archivos sin filtro de extensión, obtenido %d: %v", len(got), got)
	}
}
