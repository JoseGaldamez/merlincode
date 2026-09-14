package app

import (
	"merlincode/internal/domain"
)

// SaveWindowSize persiste las dimensiones de la ventana desde el frontend
func (a *App) SaveWindowSize(width int, height int, isMaximised bool) {
	a.windowService.SaveSize(width, height, isMaximised)
}

// SavePanelsState persiste el estado de apertura de los paneles laterales
func (a *App) SavePanelsState(leftOpen bool, rightOpen bool) {
	a.windowService.SavePanelsState(leftOpen, rightOpen)
}

// GetPanelsState retorna el estado actual de los paneles
func (a *App) GetPanelsState() domain.PanelsState {
	return a.windowService.GetPanelsState()
}
