package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"merlincode/internal/ai"
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
type ProviderStatus = domain.ProviderStatus
type ProviderValidationResult = domain.ProviderValidationResult
type ProviderUsageResult = domain.ProviderUsageResult
type TestMessageResult = domain.TestMessageResult

// App actúa como la fachada (Facade) que Wails expone al frontend de React
type App struct {
	ctx               context.Context
	cancelFunc        context.CancelFunc
	windowService     *window.Service
	workspaceService  *workspace.Service
	aiProviderService *ai.Service
}

// NewApp inicializa la estructura de la aplicación y sus servicios de dominio
func NewApp() *App {
	return &App{
		windowService:     window.NewService(),
		workspaceService:  workspace.NewService(),
		aiProviderService: ai.NewService(),
	}
}

// startup se ejecuta al iniciar la aplicación Wails y guarda el contexto de runtime
func (a *App) startup(ctx context.Context) {
	appCtx, cancel := context.WithCancel(ctx)
	a.ctx = appCtx
	a.cancelFunc = cancel
	if a.windowService.GetState().Maximised {
		runtime.WindowMaximise(ctx)
	}
}

// beforeClose se invoca antes de cerrar para cancelar peticiones pendientes y guardar el tamaño final de la ventana
func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	if a.cancelFunc != nil {
		a.cancelFunc()
	}
	if a.ctx != nil {
		isMax := runtime.WindowIsMaximised(a.ctx)
		w, h := runtime.WindowGetSize(a.ctx)
		a.windowService.SaveSize(w, h, isMax)
	}
	return false
}

// getAppContext retorna un contexto derivado del ciclo de vida de la aplicación con un timeout específico
func (a *App) getAppContext(timeout time.Duration) (context.Context, context.CancelFunc, error) {
	if a.ctx == nil {
		return nil, nil, domain.ErrRuntimeNotInitialized
	}
	ctx, cancel := context.WithTimeout(a.ctx, timeout)
	return ctx, cancel, nil
}

func sanitizeAIProviderError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return errors.New("la operación con el proveedor fue cancelada o excedió el tiempo de espera")
	case errors.Is(err, domain.ErrRuntimeNotInitialized):
		return domain.ErrRuntimeNotInitialized
	case errors.Is(err, domain.ErrUnknownAIProvider):
		return domain.ErrUnknownAIProvider
	case errors.Is(err, domain.ErrAdminKeyNotSupported):
		return domain.ErrAdminKeyNotSupported
	case errors.Is(err, domain.ErrInvalidModel):
		return domain.ErrInvalidModel
	case errors.Is(err, domain.ErrMessageTooLarge):
		return domain.ErrMessageTooLarge
	case errors.Is(err, domain.ErrRateLimited):
		return domain.ErrRateLimited
	case errors.Is(err, domain.ErrConcurrentRequestBlocked):
		return domain.ErrConcurrentRequestBlocked
	case errors.Is(err, domain.ErrCredentialMigrationFailed):
		return domain.ErrCredentialMigrationFailed
	case errors.Is(err, domain.ErrCredentialStoreUnavailable):
		return domain.ErrCredentialStoreUnavailable
	case errors.Is(err, domain.ErrProviderMetadataUnavailable):
		return domain.ErrProviderMetadataUnavailable
	default:
		return domain.ErrAIProviderOperationFailed
	}
}

func (a *App) ensureAIProviderReady() error {
	if a.aiProviderService == nil {
		return domain.ErrAIProviderOperationFailed
	}
	return sanitizeAIProviderError(a.aiProviderService.InitializationError())
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

// ListAIProviderStatuses retorna el estado (configurado/verificado/masked key) de cada proveedor de IA soportado
func (a *App) ListAIProviderStatuses() ([]ProviderStatus, error) {
	if err := a.ensureAIProviderReady(); err != nil {
		return nil, err
	}
	return a.aiProviderService.ListStatuses(), nil
}

// ValidateAndSaveAIProviderKey conecta en vivo contra el proveedor indicado para confirmar que la
// clave de API es correcta y obtener datos básicos de la cuenta; solo persiste si la validación es exitosa
func (a *App) ValidateAndSaveAIProviderKey(providerID string, apiKey string) (ProviderValidationResult, error) {
	if err := a.ensureAIProviderReady(); err != nil {
		return ProviderValidationResult{}, err
	}
	ctx, cancel, err := a.getAppContext(20 * time.Second)
	if err != nil {
		return ProviderValidationResult{}, sanitizeAIProviderError(err)
	}
	defer cancel()
	result, err := a.aiProviderService.ValidateAndSaveKey(ctx, providerID, apiKey)
	return result, sanitizeAIProviderError(err)
}

// ClearAIProviderKey elimina la credencial guardada de un proveedor de IA tanto del llavero como de metadata
func (a *App) ClearAIProviderKey(providerID string) error {
	if err := a.ensureAIProviderReady(); err != nil {
		return err
	}
	return sanitizeAIProviderError(a.aiProviderService.ClearKey(providerID))
}

// SaveAIProviderAdminKey guarda la Admin API Key de un proveedor que la requiera (p. ej. Anthropic)
// para poder consultar su consumo/costo real de organización
func (a *App) SaveAIProviderAdminKey(providerID string, adminKey string) error {
	if err := a.ensureAIProviderReady(); err != nil {
		return err
	}
	return sanitizeAIProviderError(a.aiProviderService.SaveAdminKey(providerID, adminKey))
}

// ClearAIProviderAdminKey elimina la Admin API Key guardada de un proveedor del llavero y de metadata
func (a *App) ClearAIProviderAdminKey(providerID string) error {
	if err := a.ensureAIProviderReady(); err != nil {
		return err
	}
	return sanitizeAIProviderError(a.aiProviderService.ClearAdminKey(providerID))
}

// GetAIProviderUsage consulta en vivo el consumo/costo o saldo real de la cuenta de un proveedor
func (a *App) GetAIProviderUsage(providerID string) (ProviderUsageResult, error) {
	if err := a.ensureAIProviderReady(); err != nil {
		return ProviderUsageResult{}, err
	}
	ctx, cancel, err := a.getAppContext(20 * time.Second)
	if err != nil {
		return ProviderUsageResult{}, sanitizeAIProviderError(err)
	}
	defer cancel()
	result, err := a.aiProviderService.GetProviderUsage(ctx, providerID)
	return result, sanitizeAIProviderError(err)
}

// SendAIProviderTestMessage envía un mensaje real de prueba al proveedor indicado usando su clave
// de API estándar y el modelo seleccionado, para confirmar en la interfaz que la conexión funciona
// de punta a punta con ese modelo específico
func (a *App) SendAIProviderTestMessage(providerID string, message string, model string) (TestMessageResult, error) {
	if err := a.ensureAIProviderReady(); err != nil {
		return TestMessageResult{}, err
	}
	ctx, cancel, err := a.getAppContext(30 * time.Second)
	if err != nil {
		return TestMessageResult{}, sanitizeAIProviderError(err)
	}
	defer cancel()
	result, err := a.aiProviderService.SendTestMessage(ctx, providerID, message, model)
	return result, sanitizeAIProviderError(err)
}
