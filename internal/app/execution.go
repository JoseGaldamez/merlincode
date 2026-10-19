package app

import (
	"context"
	"errors"
	"time"

	"merlincode/internal/domain"
)

// PrepareAppRun derives proposals from project files, never from a command string.
func (a *App) PrepareAppRun(request domain.RunRequest) ([]domain.RunProposal, error) {
	a.executionMu.Lock()
	defer a.executionMu.Unlock()
	if a.runnerService == nil {
		return nil, domain.ErrRuntimeNotInitialized
	}
	proposals, err := a.runnerService.Propose(request)
	return proposals, safeExecutionError(err)
}

// StartAppRun is invoked only by the user's Play action.
func (a *App) StartAppRun(proposalID string) (domain.Execution, error) {
	a.executionMu.Lock()
	defer a.executionMu.Unlock()
	if a.runnerService == nil {
		return domain.Execution{}, domain.ErrRuntimeNotInitialized
	}
	run, err := a.runnerService.Start(proposalID)
	return run, safeExecutionError(err)
}

func (a *App) StopAppRun(id string) error {
	if a.runnerService == nil {
		return domain.ErrRuntimeNotInitialized
	}
	return safeExecutionError(a.runnerService.Stop(id))
}

func (a *App) ClearAppRunLogs(id string) (domain.Execution, error) {
	if a.runnerService == nil {
		return domain.Execution{}, domain.ErrRuntimeNotInitialized
	}
	run, err := a.runnerService.ClearLogs(id)
	return run, safeExecutionError(err)
}

// RemoveAppRun stops and removes a managed execution without touching project files.
func (a *App) RemoveAppRun(id string) (domain.Execution, error) {
	a.executionMu.Lock()
	defer a.executionMu.Unlock()
	if a.runnerService == nil {
		return domain.Execution{}, domain.ErrRuntimeNotInitialized
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	run, err := a.runnerService.Remove(ctx, id)
	return run, safeExecutionError(err)
}

func (a *App) ListAppRuns() []domain.Execution {
	if a.runnerService == nil {
		return []domain.Execution{}
	}
	return a.runnerService.List()
}

func safeExecutionError(err error) error {
	if err == nil {
		return nil
	}
	var policy *domain.RunPolicyError
	if errors.As(err, &policy) {
		return policy
	}
	if errors.Is(err, domain.ErrNoActiveProject) {
		return domain.ErrNoActiveProject
	}
	if errors.Is(err, domain.ErrAccessDenied) {
		return &domain.RunPolicyError{Message: "La carpeta o entrada no es válida dentro del proyecto activo. No se permiten enlaces ni rutas externas."}
	}
	return &domain.RunPolicyError{Message: "No se pudo preparar la ejecución de la aplicación."}
}
