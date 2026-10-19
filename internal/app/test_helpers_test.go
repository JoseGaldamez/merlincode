package app

import (
	"context"
	"os"
	"testing"

	"merlincode/internal/workspace"
)

// silenceAgentEvents reemplaza los hooks de runtime de Wails por dobles no-op durante la prueba.
// runtime.EventsEmit/LogErrorf exigen un context.Context con valores inyectados por el runtime real
// de la app y abortan el proceso (log.Fatalf) si reciben un context.Context de prueba corriente.
func silenceAgentEvents(t *testing.T) {
	t.Helper()
	origEmit := emitChatStreamEvent
	origLog := logAgentTurnError
	emitChatStreamEvent = func(ctx context.Context, eventName string, data ...interface{}) {}
	logAgentTurnError = func(ctx context.Context, format string, args ...interface{}) {}
	t.Cleanup(func() {
		emitChatStreamEvent = origEmit
		logAgentTurnError = origLog
	})
}

func newTestWorkspace(t *testing.T) *workspace.Service {
	t.Helper()
	ws := workspace.NewService()
	tempDir, err := os.MkdirTemp("", "merlin_agent_test_*")
	if err != nil {
		t.Fatalf("error creando tempDir: %v", err)
	}
	t.Cleanup(func() {
		_ = ws.Close()
		_ = os.RemoveAll(tempDir)
	})
	if _, err := ws.SetActive(tempDir); err != nil {
		t.Fatalf("SetActive falló: %v", err)
	}
	return ws
}

func newTestApp(t *testing.T) *App {
	t.Helper()
	a := &App{
		ctx:              context.Background(),
		workspaceService: newTestWorkspace(t),
	}
	a.agentService = a.newAgentService()
	return a
}
