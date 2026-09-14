package main

import (
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
