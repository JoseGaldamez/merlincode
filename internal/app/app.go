package app

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"merlincode/internal/ai"
	"merlincode/internal/domain"
	"merlincode/internal/platform/config"
	"merlincode/internal/session"
	"merlincode/internal/window"
	"merlincode/internal/workspace"
)

// App actúa como la fachada (Facade) que Wails expone al frontend de React
type App struct {
	ctx               context.Context
	cancelFunc        context.CancelFunc
	windowService     *window.Service
	workspaceService  *workspace.Service
	aiProviderService *ai.Service
	sessionService    *session.Service
	activeStreamsMu   sync.Mutex
	activeStreams     map[string]context.CancelFunc
}

// NewApp inicializa la estructura de la aplicación y sus servicios de dominio
func NewApp() *App {
	dbPath, err := config.GetConfigFilePath("merlin.db")
	var sessionSvc *session.Service
	if err == nil {
		var errSvc error
		sessionSvc, errSvc = session.NewService(dbPath)
		if errSvc != nil {
			log.Printf("[App] Error inicializando session.Service en %s: %v", dbPath, errSvc)
		}
	} else {
		log.Printf("[App] Error obteniendo ruta de base de datos merlin.db: %v", err)
	}

	return &App{
		windowService:     window.NewService(),
		workspaceService:  workspace.NewService(),
		aiProviderService: ai.NewService(),
		sessionService:    sessionSvc,
		activeStreams:     make(map[string]context.CancelFunc),
	}
}

// Startup se ejecuta al iniciar la aplicación Wails y guarda el contexto de runtime
func (a *App) Startup(ctx context.Context) {
	appCtx, cancel := context.WithCancel(ctx)
	a.ctx = appCtx
	a.cancelFunc = cancel
	if a.windowService.GetState().Maximised {
		runtime.WindowMaximise(ctx)
	}
}

// BeforeClose se invoca antes de cerrar para cancelar peticiones pendientes y guardar el tamaño final de la ventana
func (a *App) BeforeClose(ctx context.Context) (prevent bool) {
	if a.cancelFunc != nil {
		a.cancelFunc()
	}
	if a.sessionService != nil {
		_ = a.sessionService.Close()
	}
	if a.ctx != nil {
		isMax := runtime.WindowIsMaximised(a.ctx)
		w, h := runtime.WindowGetSize(a.ctx)
		a.windowService.SaveSize(w, h, isMax)
	}
	return false
}

// InitialWindowState retorna el estado de la ventana para el arranque de Wails
func (a *App) InitialWindowState() domain.WindowState {
	return a.windowService.GetState()
}

// getAppContext retorna un contexto derivado del ciclo de vida de la aplicación con un timeout específico
func (a *App) getAppContext(timeout time.Duration) (context.Context, context.CancelFunc, error) {
	if a.ctx == nil {
		return nil, nil, domain.ErrRuntimeNotInitialized
	}
	ctx, cancel := context.WithTimeout(a.ctx, timeout)
	return ctx, cancel, nil
}

// Greet retorna un mensaje de saludo para pruebas
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// OpenURLInDefaultBrowser abre una URL en el navegador predeterminado del sistema operativo
func (a *App) OpenURLInDefaultBrowser(url string) {
	if a.ctx != nil {
		runtime.BrowserOpenURL(a.ctx, url)
	}
}
