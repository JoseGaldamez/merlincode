package main

import (
	"os"
	"testing"
)

func TestAppIntegrationWindowState(t *testing.T) {
	app := NewApp()

	// Probar persistencia de tamaño y paneles a través de la fachada App
	app.SaveWindowSize(1440, 900, false)
	app.SavePanelsState(false, true)

	panels := app.GetPanelsState()
	if panels.LeftOpen != false || panels.RightOpen != true {
		t.Fatalf("Esperado GetPanelsState() (false, true), obtenido (%v, %v)",
			panels.LeftOpen, panels.RightOpen)
	}

	// Restaurar valores estándar
	app.SaveWindowSize(1920, 1080, false)
	app.SavePanelsState(true, false)
}

func TestAppIntegrationWorkspaceSandboxing(t *testing.T) {
	app := NewApp()

	// Operación sin proyecto activo debe fallar
	if err := app.WriteProjectFile("test.txt", "hello"); err == nil {
		t.Fatal("Esperado error al escribir sin proyecto activo")
	}

	tempDir, err := os.MkdirTemp("", "merlin_app_integration_*")
	if err != nil {
		t.Fatalf("Error creando tempDir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	proj, err := app.SetActiveProject(tempDir)
	if err != nil {
		t.Fatalf("SetActiveProject falló: %v", err)
	}
	if proj == nil || proj.Path != tempDir {
		t.Fatalf("Información de proyecto inesperada: %+v", proj)
	}

	// Escritura permitida dentro del proyecto
	err = app.WriteProjectFile("nested/example.txt", "Contenido seguro")
	if err != nil {
		t.Fatalf("WriteProjectFile falló: %v", err)
	}

	content, err := app.ReadProjectFile("nested/example.txt")
	if err != nil {
		t.Fatalf("ReadProjectFile falló: %v", err)
	}
	if content != "Contenido seguro" {
		t.Fatalf("Esperado 'Contenido seguro', obtenido '%s'", content)
	}

	// Intento de escape hacia afuera del proyecto (Path Traversal)
	err = app.WriteProjectFile("../outside.txt", "Hack")
	if err == nil {
		t.Fatal("Esperado que el intento de path traversal sea rechazado, pero fue aceptado")
	}

	_, err = app.ReadProjectFile("../outside.txt")
	if err == nil {
		t.Fatal("Esperado que la lectura con path traversal sea rechazada, pero fue aceptada")
	}

	// Listar archivos y árbol
	items, err := app.ListProjectFiles()
	if err != nil {
		t.Fatalf("ListProjectFiles falló: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("Esperado al menos 1 elemento en el proyecto")
	}

	tree, err := app.GetProjectTree()
	if err != nil {
		t.Fatalf("GetProjectTree falló: %v", err)
	}
	if len(tree) == 0 {
		t.Fatal("Esperado al menos 1 nodo en el árbol")
	}
}
