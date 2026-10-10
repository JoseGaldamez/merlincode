package app

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"merlincode/internal/ai"
	"merlincode/internal/domain"
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
	return &App{
		ctx:              context.Background(),
		workspaceService: newTestWorkspace(t),
		pendingApprovals: make(map[string]*pendingApproval),
	}
}

func TestWithEditedContent(t *testing.T) {
	out := withEditedContent(`{"path":"a.txt","content":"viejo"}`, "nuevo")
	if !strings.Contains(out, `"content":"nuevo"`) || !strings.Contains(out, `"path":"a.txt"`) {
		t.Fatalf("contenido editado no aplicado correctamente: %s", out)
	}

	// JSON malformado: no debe entrar en pánico, retorna algo usable (mapa vacío + content nuevo)
	out = withEditedContent(`{no es json`, "nuevo")
	if !strings.Contains(out, `"content":"nuevo"`) {
		t.Fatalf("fallback de JSON inválido no aplicó el contenido editado: %s", out)
	}
}

func TestSanitizeToolErrorForModel(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{domain.ErrAccessDenied, "fuera del proyecto activo"},
		{domain.ErrFileTooLarge, "excede el límite"},
		{domain.ErrNoActiveProject, "ningún proyecto activo"},
		{domain.ErrFolderNotExist, "no existe"},
		{errors.New("algo interno explotó"), "no se pudo ejecutar"},
	}
	for _, c := range cases {
		got := sanitizeToolErrorForModel(c.err)
		if !strings.Contains(got, c.want) {
			t.Errorf("para %v: esperado que contenga %q, obtenido %q", c.err, c.want, got)
		}
		if strings.Contains(got, "C:\\") || strings.Contains(got, "/home/") {
			t.Errorf("el mensaje filtró una ruta absoluta: %q", got)
		}
	}
}

func TestSummarizeToolArgs(t *testing.T) {
	if got := summarizeToolArgs(`{"path":"src/main.go","content":"..."}`); got != "src/main.go" {
		t.Fatalf("esperado 'src/main.go', obtenido %q", got)
	}

	long := strings.Repeat("x", 200)
	got := summarizeToolArgs(`{"other":"` + long + `"}`)
	if len(got) != 83 || !strings.HasSuffix(got, "...") {
		t.Fatalf("truncamiento inesperado: len=%d valor=%q", len(got), got)
	}

	short := `{"other":"abc"}`
	if got := summarizeToolArgs(short); got != short {
		t.Fatalf("argumentos cortos sin path no deberían truncarse: %q", got)
	}
}

func TestRespondToToolApproval_UnknownRequestID(t *testing.T) {
	a := newTestApp(t)
	err := a.RespondToToolApproval("no-existe", true, "")
	if !errors.Is(err, domain.ErrApprovalRequestNotFound) {
		t.Fatalf("esperado ErrApprovalRequestNotFound, obtenido: %v", err)
	}
}

// waitForPendingApprovalID espera (con timeout corto) a que awaitToolApproval registre su entrada,
// y retorna el requestID sintetizado internamente.
func waitForPendingApprovalID(t *testing.T, a *App) string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		a.pendingApprovalsMu.Lock()
		for id := range a.pendingApprovals {
			a.pendingApprovalsMu.Unlock()
			return id
		}
		a.pendingApprovalsMu.Unlock()
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("no se registró ninguna aprobación pendiente a tiempo")
	return ""
}

func TestAwaitToolApproval_ApprovedWithEditedContent(t *testing.T) {
	silenceAgentEvents(t)
	a := newTestApp(t)
	tc := domain.ToolCall{ID: "call-1", Name: "write_file", Arguments: `{"path":"a.txt","content":"propuesto"}`}

	type outcome struct {
		resp approvalResponse
		err  error
	}
	resCh := make(chan outcome, 1)
	go func() {
		resp, err := a.awaitToolApproval(context.Background(), domain.ChatStreamRequest{SessionID: "s1", MessageID: "m1"}, tc)
		resCh <- outcome{resp, err}
	}()

	requestID := waitForPendingApprovalID(t, a)
	if err := a.RespondToToolApproval(requestID, true, "editado por el usuario"); err != nil {
		t.Fatalf("RespondToToolApproval falló: %v", err)
	}

	select {
	case r := <-resCh:
		if r.err != nil {
			t.Fatalf("error inesperado: %v", r.err)
		}
		if !r.resp.approved || !r.resp.hasEditedValue || r.resp.editedContent != "editado por el usuario" {
			t.Fatalf("respuesta inesperada: %+v", r.resp)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("awaitToolApproval no retornó a tiempo")
	}

	// La entrada debe limpiarse del mapa tras resolverse
	a.pendingApprovalsMu.Lock()
	_, stillPending := a.pendingApprovals[requestID]
	a.pendingApprovalsMu.Unlock()
	if stillPending {
		t.Fatal("la aprobación resuelta no fue eliminada del mapa de pendientes")
	}
}

func TestAwaitToolApproval_Rejected(t *testing.T) {
	silenceAgentEvents(t)
	a := newTestApp(t)
	tc := domain.ToolCall{ID: "call-2", Name: "write_file", Arguments: `{"path":"b.txt","content":"x"}`}

	resCh := make(chan approvalResponse, 1)
	go func() {
		resp, _ := a.awaitToolApproval(context.Background(), domain.ChatStreamRequest{}, tc)
		resCh <- resp
	}()

	requestID := waitForPendingApprovalID(t, a)
	if err := a.RespondToToolApproval(requestID, false, ""); err != nil {
		t.Fatalf("RespondToToolApproval falló: %v", err)
	}

	select {
	case resp := <-resCh:
		if resp.approved {
			t.Fatalf("se esperaba rechazo, obtenido: %+v", resp)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("awaitToolApproval no retornó a tiempo")
	}
}

func TestAwaitToolApproval_ContextCanceled(t *testing.T) {
	silenceAgentEvents(t)
	a := newTestApp(t)
	tc := domain.ToolCall{ID: "call-3", Name: "write_file", Arguments: `{"path":"c.txt","content":"x"}`}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	resp, err := a.awaitToolApproval(ctx, domain.ChatStreamRequest{}, tc)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("esperado context.Canceled, obtenido: %v", err)
	}
	if resp.approved {
		t.Fatalf("una cancelación no debe reportarse como aprobada: %+v", resp)
	}

	a.pendingApprovalsMu.Lock()
	count := len(a.pendingApprovals)
	a.pendingApprovalsMu.Unlock()
	if count != 0 {
		t.Fatalf("la entrada pendiente no se limpió tras la cancelación, quedan: %d", count)
	}
}

func TestExecuteToolCalls_UnknownTool(t *testing.T) {
	a := newTestApp(t)
	registry := ai.NewToolRegistry(a.workspaceService)
	var trace []domain.ToolTraceEntry

	calls := []domain.ToolCall{{ID: "1", Name: "borrar_todo", Arguments: "{}"}}
	results, canceled := a.executeToolCalls(context.Background(), domain.ChatStreamRequest{}, registry, calls, &trace)

	if canceled {
		t.Fatal("no se esperaba cancelación")
	}
	if len(results) != 1 || !results[0].IsError || !strings.Contains(results[0].Content, "borrar_todo") {
		t.Fatalf("resultado inesperado para herramienta desconocida: %+v", results)
	}
	if len(trace) != 1 || !trace[0].IsError {
		t.Fatalf("traza inesperada: %+v", trace)
	}
}

func TestExecuteToolCalls_ReadFileSuccessAndError(t *testing.T) {
	a := newTestApp(t)
	if err := a.workspaceService.WriteFile("nota.txt", "contenido de prueba"); err != nil {
		t.Fatalf("preparación falló: %v", err)
	}
	registry := ai.NewToolRegistry(a.workspaceService)
	var trace []domain.ToolTraceEntry

	calls := []domain.ToolCall{
		{ID: "1", Name: "read_file", Arguments: `{"path":"nota.txt"}`},
		{ID: "2", Name: "read_file", Arguments: `{"path":"../fuera.txt"}`},
	}
	results, canceled := a.executeToolCalls(context.Background(), domain.ChatStreamRequest{}, registry, calls, &trace)
	if canceled {
		t.Fatal("no se esperaba cancelación")
	}
	if len(results) != 2 {
		t.Fatalf("se esperaban 2 resultados, obtenido: %d", len(results))
	}
	if results[0].IsError || results[0].Content != "contenido de prueba" {
		t.Fatalf("lectura válida falló: %+v", results[0])
	}
	if !results[1].IsError || !strings.Contains(results[1].Content, "fuera del proyecto activo") {
		t.Fatalf("path traversal debió mapearse a error accionable: %+v", results[1])
	}
	if len(trace) != 2 {
		t.Fatalf("traza incompleta: %+v", trace)
	}
}

func TestExecuteToolCalls_WriteRequiresApprovalThenApplied(t *testing.T) {
	silenceAgentEvents(t)
	a := newTestApp(t)
	registry := ai.NewToolRegistry(a.workspaceService)
	var trace []domain.ToolTraceEntry

	calls := []domain.ToolCall{
		{ID: "call-w1", Name: "write_file", Arguments: `{"path":"salida.txt","content":"propuesto por el modelo"}`},
	}

	type outcome struct {
		results  []domain.ToolResult
		canceled bool
	}
	resCh := make(chan outcome, 1)
	go func() {
		results, canceled := a.executeToolCalls(context.Background(), domain.ChatStreamRequest{}, registry, calls, &trace)
		resCh <- outcome{results, canceled}
	}()

	requestID := waitForPendingApprovalID(t, a)
	// El usuario edita el contenido antes de aprobar
	if err := a.RespondToToolApproval(requestID, true, "contenido editado por el usuario"); err != nil {
		t.Fatalf("RespondToToolApproval falló: %v", err)
	}

	select {
	case o := <-resCh:
		if o.canceled {
			t.Fatal("no se esperaba cancelación")
		}
		if len(o.results) != 1 || o.results[0].IsError {
			t.Fatalf("resultado inesperado: %+v", o.results)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("executeToolCalls no retornó a tiempo")
	}

	written, err := a.workspaceService.ReadFile("salida.txt")
	if err != nil {
		t.Fatalf("no se pudo leer el archivo escrito: %v", err)
	}
	if written != "contenido editado por el usuario" {
		t.Fatalf("se esperaba el contenido editado, obtenido: %q", written)
	}
	if len(trace) != 1 || trace[0].Approved == nil || !*trace[0].Approved {
		t.Fatalf("traza no refleja la aprobación: %+v", trace)
	}
}

func TestExecuteToolCalls_WriteRejected(t *testing.T) {
	silenceAgentEvents(t)
	a := newTestApp(t)
	registry := ai.NewToolRegistry(a.workspaceService)
	var trace []domain.ToolTraceEntry

	calls := []domain.ToolCall{
		{ID: "call-w2", Name: "write_file", Arguments: `{"path":"rechazado.txt","content":"no debería escribirse"}`},
	}

	type outcome struct {
		results  []domain.ToolResult
		canceled bool
	}
	resCh := make(chan outcome, 1)
	go func() {
		results, canceled := a.executeToolCalls(context.Background(), domain.ChatStreamRequest{}, registry, calls, &trace)
		resCh <- outcome{results, canceled}
	}()

	requestID := waitForPendingApprovalID(t, a)
	if err := a.RespondToToolApproval(requestID, false, ""); err != nil {
		t.Fatalf("RespondToToolApproval falló: %v", err)
	}

	select {
	case o := <-resCh:
		if o.canceled {
			t.Fatal("no se esperaba cancelación")
		}
		if len(o.results) != 1 || !o.results[0].IsError {
			t.Fatalf("se esperaba un resultado de rechazo: %+v", o.results)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("executeToolCalls no retornó a tiempo")
	}

	if _, err := os.Stat(a.workspaceService.GetActiveDir() + "/rechazado.txt"); err == nil {
		t.Fatal("el archivo no debió escribirse tras el rechazo")
	}
	if len(trace) != 1 || trace[0].Approved == nil || *trace[0].Approved {
		t.Fatalf("traza no refleja el rechazo: %+v", trace)
	}
}

func TestExecuteToolCalls_CanceledWhileAwaitingApproval(t *testing.T) {
	silenceAgentEvents(t)
	a := newTestApp(t)
	registry := ai.NewToolRegistry(a.workspaceService)
	var trace []domain.ToolTraceEntry

	ctx, cancel := context.WithCancel(context.Background())
	calls := []domain.ToolCall{
		{ID: "call-w3", Name: "write_file", Arguments: `{"path":"cancelado.txt","content":"x"}`},
	}

	type outcome struct {
		results  []domain.ToolResult
		canceled bool
	}
	resCh := make(chan outcome, 1)
	go func() {
		results, canceled := a.executeToolCalls(ctx, domain.ChatStreamRequest{}, registry, calls, &trace)
		resCh <- outcome{results, canceled}
	}()

	waitForPendingApprovalID(t, a)
	cancel()

	select {
	case o := <-resCh:
		if !o.canceled {
			t.Fatalf("se esperaba cancelación, obtenido: %+v", o)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("executeToolCalls no retornó tras cancelar el contexto")
	}
}
