package window

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestWindowServiceLifecycle(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "merlin_window_test_*")
	if err != nil {
		t.Fatalf("Error creando tempDir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFilePath := filepath.Join(tempDir, "window.json")

	// 1. Instancia inicial con defaults
	svc := NewServiceWithCustomPath(testFilePath)
	initialState := svc.GetState()

	if initialState.Width != 1920 || initialState.Height != 1080 {
		t.Fatalf("Esperado 1920x1080, obtenido %dx%d", initialState.Width, initialState.Height)
	}
	if !initialState.LeftPanelOpen || initialState.RightPanelOpen {
		t.Fatalf("Esperado Left: true, Right: false; obtenido Left: %v, Right: %v",
			initialState.LeftPanelOpen, initialState.RightPanelOpen)
	}

	// 2. Modificación y persistencia
	svc.SaveSize(1440, 900, false)
	svc.SavePanelsState(false, true)

	panels := svc.GetPanelsState()
	if panels.LeftOpen != false || panels.RightOpen != true {
		t.Fatalf("Esperado paneles (false, true), obtenido (%v, %v)", panels.LeftOpen, panels.RightOpen)
	}

	// 3. Recarga desde disco en nueva instancia
	svcReloaded := NewServiceWithCustomPath(testFilePath)
	reloadedState := svcReloaded.GetState()

	if reloadedState.Width != 1440 || reloadedState.Height != 900 {
		t.Fatalf("Esperado recargado 1440x900, obtenido %dx%d", reloadedState.Width, reloadedState.Height)
	}
	if reloadedState.LeftPanelOpen != false || reloadedState.RightPanelOpen != true {
		t.Fatalf("Esperado estado recargado (false, true), obtenido (%v, %v)",
			reloadedState.LeftPanelOpen, reloadedState.RightPanelOpen)
	}
}

func TestWindowServiceConcurrency(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "merlin_window_conc_*")
	if err != nil {
		t.Fatalf("Error creando tempDir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFilePath := filepath.Join(tempDir, "window.json")
	svc := NewServiceWithCustomPath(testFilePath)

	var wg sync.WaitGroup
	workers := 20

	// Lanzar lecturas y escrituras concurrentes
	for i := 0; i < workers; i++ {
		wg.Add(2)
		go func(idx int) {
			defer wg.Done()
			svc.SaveSize(1000+idx, 800+idx, false)
			svc.SavePanelsState(idx%2 == 0, idx%2 != 0)
		}(i)

		go func() {
			defer wg.Done()
			_ = svc.GetState()
			_ = svc.GetPanelsState()
		}()
	}

	wg.Wait()

	finalState := svc.GetState()
	if finalState.Width < 1000 || finalState.Height < 800 {
		t.Fatalf("Estado final inesperado tras concurrencia: %+v", finalState)
	}
}
