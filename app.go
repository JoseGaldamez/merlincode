package main

import (
	"context"
	"fmt"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"merlincode/internal/domain"
	"merlincode/internal/window"
	"merlincode/internal/workspace"
)

// Type aliases para mantener compatibilidad total con los bindings y modelos TypeScript de Wails
type ProjectInfo = domain.ProjectInfo
type FileItem = domain.FileItem
type FileNode = domain.FileNode
type PanelsState = domain.PanelsState
type WindowState = domain.WindowState

// App actúa como la fachada (Facade) que Wails expone al frontend de React
type App struct {
	ctx              context.Context
	windowService    *window.Service
	workspaceService *workspace.Service
}

// NewApp inicializa la estructura de la aplicación y sus servicios de dominio
func NewApp() *App {
	return &App{
		windowService:    window.NewService(),
		workspaceService: workspace.NewService(),
	}
}

// startup se ejecuta al iniciar la aplicación Wails y guarda el contexto de runtime
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if a.windowService.GetState().Maximised {
		runtime.WindowMaximise(ctx)
	}
}

// beforeClose se invoca antes de cerrar para guardar el tamaño final de la ventana
func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	if a.ctx != nil {
		isMax := runtime.WindowIsMaximised(a.ctx)
		w, h := runtime.WindowGetSize(a.ctx)
		a.windowService.SaveSize(w, h, isMax)
	}
	return false
}

// Greet retorna un mensaje de saludo para pruebas
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// SaveWindowSize persiste las dimensiones de la ventana desde el frontend
func (a *App) SaveWindowSize(width int, height int, isMaximised bool) {
	a.windowService.SaveSize(width, height, isMaximised)
}

// SavePanelsState persiste el estado de apertura de los paneles laterales
func (a *App) SavePanelsState(leftOpen bool, rightOpen bool) {
	a.windowService.SavePanelsState(leftOpen, rightOpen)
}

// GetPanelsState retorna el estado actual de los paneles
func (a *App) GetPanelsState() PanelsState {
	return a.windowService.GetPanelsState()
}

// SelectProjectFolder abre el diálogo nativo para seleccionar una carpeta y la establece como activa
func (a *App) SelectProjectFolder() (*ProjectInfo, error) {
	if a.ctx == nil {
		return nil, domain.ErrRuntimeNotInitialized
	}

	selectedDir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Seleccionar carpeta del proyecto",
	})
	if err != nil {
		return nil, err
	}
	if selectedDir == "" {
		return nil, nil // Cancelado por el usuario
	}

	return a.workspaceService.SetActive(selectedDir)
}

// SetActiveProject valida permisos y define la carpeta activa del proyecto
func (a *App) SetActiveProject(dirPath string) (*ProjectInfo, error) {
	return a.workspaceService.SetActive(dirPath)
}

// GetActiveProject retorna la información del proyecto activo o nil
func (a *App) GetActiveProject() *ProjectInfo {
	return a.workspaceService.GetActive()
}

// WriteProjectFile escribe contenido en un archivo dentro del proyecto activo asegurando sandboxing
func (a *App) WriteProjectFile(relativePath string, content string) error {
	return a.workspaceService.WriteFile(relativePath, content)
}

// ReadProjectFile lee el contenido de un archivo dentro del proyecto activo
func (a *App) ReadProjectFile(relativePath string) (string, error) {
	return a.workspaceService.ReadFile(relativePath)
}

// ListProjectFiles retorna los archivos y carpetas directamente en la raíz del proyecto activo
func (a *App) ListProjectFiles() ([]FileItem, error) {
	return a.workspaceService.ListFiles()
}

// GetProjectTree retorna el árbol jerárquico de archivos del proyecto activo
func (a *App) GetProjectTree() ([]FileNode, error) {
	return a.workspaceService.GetTree()
}

// OpenDirectoryInExplorer abre la carpeta indicada o el proyecto activo en el explorador del sistema
func (a *App) OpenDirectoryInExplorer(targetPath string) error {
	return a.workspaceService.OpenExplorer(targetPath)
}

// OpenURLInDefaultBrowser abre una URL en el navegador predeterminado del sistema operativo
func (a *App) OpenURLInDefaultBrowser(url string) {
	if a.ctx != nil {
		runtime.BrowserOpenURL(a.ctx, url)
	}
}

