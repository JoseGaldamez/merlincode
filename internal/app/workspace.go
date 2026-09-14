package app

import (
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"merlincode/internal/domain"
)

// SelectProjectFolder abre el diálogo nativo para seleccionar una carpeta y la establece como activa
func (a *App) SelectProjectFolder() (*domain.ProjectInfo, error) {
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
func (a *App) SetActiveProject(dirPath string) (*domain.ProjectInfo, error) {
	return a.workspaceService.SetActive(dirPath)
}

// GetActiveProject retorna la información del proyecto activo o nil
func (a *App) GetActiveProject() *domain.ProjectInfo {
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
func (a *App) ListProjectFiles() ([]domain.FileItem, error) {
	return a.workspaceService.ListFiles()
}

// GetProjectTree retorna el árbol jerárquico de archivos del proyecto activo
func (a *App) GetProjectTree() ([]domain.FileNode, error) {
	return a.workspaceService.GetTree()
}

// OpenDirectoryInExplorer abre la carpeta indicada o el proyecto activo en el explorador del sistema
func (a *App) OpenDirectoryInExplorer(targetPath string) error {
	return a.workspaceService.OpenExplorer(targetPath)
}
