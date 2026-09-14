package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"merlincode/internal/domain"
)

func TestWorkspaceServiceBasicsAndSandboxing(t *testing.T) {
	svc := NewService()
	defer svc.Close()

	// 1. Operar sin proyecto activo debe retornar ErrNoActiveProject
	if err := svc.WriteFile("test.txt", "contenido"); !errors.Is(err, domain.ErrNoActiveProject) {
		t.Fatalf("Esperado ErrNoActiveProject, obtenido: %v", err)
	}
	if _, err := svc.ReadFile("test.txt"); !errors.Is(err, domain.ErrNoActiveProject) {
		t.Fatalf("Esperado ErrNoActiveProject al leer, obtenido: %v", err)
	}

	tempDir, err := os.MkdirTemp("", "merlin_ws_test_*")
	if err != nil {
		t.Fatalf("Error creando tempDir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("no se pudo limpiar el directorio temporal: %v", err)
		}
	})

	// 2. Definir proyecto activo
	proj, err := svc.SetActive(tempDir)
	if err != nil {
		t.Fatalf("SetActive falló: %v", err)
	}
	if proj == nil {
		t.Fatal("Proyecto inesperadamente nil")
	}

	expectedInfo, err := os.Stat(tempDir)
	if err != nil {
		t.Fatalf("No se pudo inspeccionar tempDir: %v", err)
	}

	actualInfo, err := os.Stat(proj.Path)
	if err != nil {
		t.Fatalf("No se pudo inspeccionar proj.Path: %v", err)
	}

	if !os.SameFile(expectedInfo, actualInfo) {
		t.Fatalf("Ruta de proyecto inesperada: esperado=%q obtenido=%q", tempDir, proj.Path)
	}

	// 3. Escritura y lectura válida dentro del proyecto
	err = svc.WriteFile("sub/doc.txt", "Texto de prueba seguro")
	if err != nil {
		t.Fatalf("WriteFile falló: %v", err)
	}

	content, err := svc.ReadFile("sub/doc.txt")
	if err != nil {
		t.Fatalf("ReadFile falló: %v", err)
	}
	if content != "Texto de prueba seguro" {
		t.Fatalf("Esperado 'Texto de prueba seguro', obtenido '%s'", content)
	}

	// 4. Intento de Path Traversal debe ser rechazado
	err = svc.WriteFile("../escape.txt", "Hack")
	if !errors.Is(err, domain.ErrAccessDenied) {
		t.Fatalf("Esperado ErrAccessDenied al intentar escribir fuera, obtenido: %v", err)
	}

	_, err = svc.ReadFile("../escape.txt")
	if !errors.Is(err, domain.ErrAccessDenied) {
		t.Fatalf("Esperado ErrAccessDenied al intentar leer fuera, obtenido: %v", err)
	}

	// 5. Listar y obtener árbol
	tree, err := svc.GetTree()
	if err != nil {
		t.Fatalf("GetTree falló: %v", err)
	}
	if len(tree) == 0 {
		t.Fatal("Esperado al menos 1 nodo en el árbol del proyecto")
	}
}

func TestWorkspaceConcurrency(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "merlin_ws_conc_*")
	if err != nil {
		t.Fatalf("Error creando tempDir: %v", err)
	}
	t.Cleanup(func() {
		for i := 0; i < 5; i++ {
			if err := os.RemoveAll(tempDir); err == nil {
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
	})

	svc := NewService()
	defer svc.Close()
	_, _ = svc.SetActive(tempDir)

	var wg sync.WaitGroup
	workers := 25

	for i := 0; i < workers; i++ {
		wg.Add(2)
		go func(idx int) {
			defer wg.Done()
			fileName := filepath.Join("dir", "file.txt")
			_ = svc.WriteFile(fileName, "datos concurrentes")
		}(i)

		go func() {
			defer wg.Done()
			_ = svc.GetActive()
			_, _ = svc.GetTree()
		}()
	}

	wg.Wait()
}
