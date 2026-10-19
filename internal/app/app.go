package app

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"merlincode/internal/agent"
	"merlincode/internal/ai"
	"merlincode/internal/domain"
	"merlincode/internal/platform/config"
	"merlincode/internal/runner"
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
	runnerService     *runner.Service
	executionMu       sync.Mutex
	streamsWG         sync.WaitGroup
	activeStreamsMu   sync.Mutex
	activeStreams     map[string]context.CancelFunc

	agentService *agent.Service
}

// NewApp inicializa la estructura de la aplicación y sus servicios de dominio
func NewApp() *App {
	dbPath, err := config.GetConfigFilePath("merlin.db")
	var sessionSvc *session.Service
	if err == nil {
		var errSvc error
		sessionSvc, errSvc = session.NewService(dbPath)
		if errSvc != nil {
			log.Print("[App] No se pudo inicializar la base de datos local")
		}
	} else {
		log.Print("[App] No se pudo obtener el directorio de datos local")
	}

	ws := workspace.NewService()
	a := &App{
		windowService:     window.NewService(),
		workspaceService:  ws,
		runnerService:     runner.NewService(ws),
		aiProviderService: ai.NewService(),
		sessionService:    sessionSvc,
		activeStreams:     make(map[string]context.CancelFunc),
	}
	a.agentService = a.newAgentService()
	return a
}

// Startup se ejecuta al iniciar la aplicación Wails y guarda el contexto de runtime
func (a *App) Startup(ctx context.Context) {
	appCtx, cancel := context.WithCancel(ctx)
	a.ctx = appCtx
	a.cancelFunc = cancel
	if a.runnerService != nil {
		a.runnerService.StartEvents(appCtx, func(event domain.Execution) {
			runtime.EventsEmit(appCtx, "execution:update", event)
		})
	}
	if a.windowService.GetState().Maximised {
		runtime.WindowMaximise(ctx)
	}
}

// BeforeClose se invoca antes de cerrar para cancelar peticiones pendientes y guardar el tamaño final de la ventana
func (a *App) BeforeClose(ctx context.Context) (prevent bool) {
	a.executionMu.Lock()
	if a.runnerService != nil {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := a.runnerService.Close(stopCtx)
		cancel()
		if err != nil {
			a.executionMu.Unlock()
			runtime.MessageDialog(ctx, runtime.MessageDialogOptions{Type: runtime.ErrorDialog, Title: "Aplicaciones en ejecución", Message: "No se pudo confirmar la detención de todas las aplicaciones. Intenta cerrar Merlin Code nuevamente."})
			return true
		}
	}
	a.executionMu.Unlock()
	a.activeStreamsMu.Lock()
	if a.cancelFunc != nil {
		a.cancelFunc()
	}
	a.activeStreamsMu.Unlock()
	a.streamsWG.Wait()
	if a.workspaceService != nil {
		_ = a.workspaceService.Close()
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
