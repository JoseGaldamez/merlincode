package main

import (
	"os"
	"testing"
)

func TestWindowStatePersistence(t *testing.T) {
	// Clean previous state file if any for testing default
	path := getWindowStateFilePath()
	_ = os.Remove(path)

	initial := LoadWindowState()
	if initial.Width != 1920 || initial.Height != 1080 {
		t.Fatalf("Expected default 1920x1080, got %dx%d", initial.Width, initial.Height)
	}
	if !initial.LeftPanelOpen {
		t.Fatalf("Expected LeftPanelOpen to be true by default, got %v", initial.LeftPanelOpen)
	}
	if initial.RightPanelOpen {
		t.Fatalf("Expected RightPanelOpen to be false by default, got %v", initial.RightPanelOpen)
	}

	app := NewApp()
	app.SaveWindowSize(1440, 900, false)
	app.SavePanelsState(false, true)

	reloaded := LoadWindowState()
	if reloaded.Width != 1440 || reloaded.Height != 900 {
		t.Fatalf("Expected reloaded 1440x900, got %dx%d", reloaded.Width, reloaded.Height)
	}
	if reloaded.LeftPanelOpen != false {
		t.Fatalf("Expected reloaded LeftPanelOpen to be false, got %v", reloaded.LeftPanelOpen)
	}
	if reloaded.RightPanelOpen != true {
		t.Fatalf("Expected reloaded RightPanelOpen to be true, got %v", reloaded.RightPanelOpen)
	}

	panels := app.GetPanelsState()
	if panels.LeftOpen != false || panels.RightOpen != true {
		t.Fatalf("Expected GetPanelsState to be false, true, got %v, %v", panels.LeftOpen, panels.RightOpen)
	}

	// Reset back to default 1920x1080 and panels (true, false) for user's launch
	app.SaveWindowSize(1920, 1080, false)
	app.SavePanelsState(true, false)
}
