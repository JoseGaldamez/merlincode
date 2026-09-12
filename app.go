package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// PanelsState represents the open/closed state of the sidebars
type PanelsState struct {
	LeftOpen  bool `json:"leftOpen"`
	RightOpen bool `json:"rightOpen"`
}

// WindowState stores dimensions and state of the desktop window and panels
type WindowState struct {
	Width          int  `json:"width"`
	Height         int  `json:"height"`
	Maximised      bool `json:"maximised"`
	LeftPanelOpen  bool `json:"left_panel_open"`
	RightPanelOpen bool `json:"right_panel_open"`
}

// App struct
type App struct {
	ctx         context.Context
	windowState WindowState
}

func getWindowStateFilePath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	appDir := filepath.Join(configDir, "merlincode")
	_ = os.MkdirAll(appDir, 0755)
	return filepath.Join(appDir, "window.json")
}

// LoadWindowState loads saved window size or defaults to Full HD (1920x1080) with right panel closed
func LoadWindowState() WindowState {
	state := WindowState{
		Width:          1920,
		Height:         1080,
		Maximised:      false,
		LeftPanelOpen:  true,  // Left open by default
		RightPanelOpen: false, // Right closed by default
	}

	filePath := getWindowStateFilePath()
	data, err := os.ReadFile(filePath)
	if err != nil {
		return state
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return state
	}

	if w, ok := raw["width"].(float64); ok && int(w) >= 600 {
		state.Width = int(w)
	}
	if h, ok := raw["height"].(float64); ok && int(h) >= 400 {
		state.Height = int(h)
	}
	if max, ok := raw["maximised"].(bool); ok {
		state.Maximised = max
	}
	if l, ok := raw["left_panel_open"].(bool); ok {
		state.LeftPanelOpen = l
	}
	if r, ok := raw["right_panel_open"].(bool); ok {
		state.RightPanelOpen = r
	}

	return state
}

// SaveWindowState writes window state to config
func (a *App) SaveWindowState(state WindowState) {
	if state.Width < 600 || state.Height < 400 {
		return
	}
	filePath := getWindowStateFilePath()
	data, err := json.MarshalIndent(state, "", "  ")
	if err == nil {
		_ = os.WriteFile(filePath, data, 0644)
	}
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		windowState: LoadWindowState(),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if a.windowState.Maximised {
		runtime.WindowMaximise(ctx)
	}
}

// beforeClose is called before the application closes to persist the current window size
func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	if a.ctx != nil {
		isMax := runtime.WindowIsMaximised(a.ctx)
		w, h := runtime.WindowGetSize(a.ctx)
		if !isMax && w >= 600 && h >= 400 {
			a.windowState.Width = w
			a.windowState.Height = h
		}
		a.windowState.Maximised = isMax
		a.SaveWindowState(a.windowState)
	}
	return false
}

// SaveWindowSize is callable from the frontend on resize to ensure persistence
func (a *App) SaveWindowSize(width int, height int, isMaximised bool) {
	if !isMaximised && width >= 600 && height >= 400 {
		a.windowState.Width = width
		a.windowState.Height = height
	}
	a.windowState.Maximised = isMaximised
	a.SaveWindowState(a.windowState)
}

// SavePanelsState persists the left and right panels open/close state
func (a *App) SavePanelsState(leftOpen bool, rightOpen bool) {
	a.windowState.LeftPanelOpen = leftOpen
	a.windowState.RightPanelOpen = rightOpen
	a.SaveWindowState(a.windowState)
}

// GetPanelsState returns the saved panel states
func (a *App) GetPanelsState() PanelsState {
	return PanelsState{
		LeftOpen:  a.windowState.LeftPanelOpen,
		RightOpen: a.windowState.RightPanelOpen,
	}
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}
