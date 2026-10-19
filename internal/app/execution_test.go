package app

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"merlincode/internal/agent"
	"merlincode/internal/domain"
	"merlincode/internal/runner"
)

func TestProjectSwitchKeepsServersRunningAndCloseStopsAll(t *testing.T) {
	a := newTestApp(t)
	a.runnerService = runner.NewService(a.workspaceService)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := a.runnerService.Close(ctx); err != nil {
			t.Error(err)
		}
	})
	projectA := a.workspaceService.GetActiveDir()
	var runs []domain.Execution
	for _, project := range []string{projectA, t.TempDir()} {
		info, err := a.SetActiveProject(project)
		if err != nil {
			t.Fatal(err)
		}
		if err := a.workspaceService.WriteFile("index.html", "served by "+info.Path); err != nil {
			t.Fatal(err)
		}
		proposals, err := a.PrepareAppRun(domain.RunRequest{Directory: "."})
		if err != nil || len(proposals) != 1 {
			t.Fatalf("missing proposal: %v", err)
		}
		run, err := a.StartAppRun(proposals[0].ID)
		if err != nil {
			t.Fatal(err)
		}
		runs = append(runs, run)
	}
	if _, err := a.StartAppRun(runs[0].Proposal.ID); err == nil {
		t.Fatal("a proposal from another project must still require activating its project")
	}
	// Clearing the selected project also preserves processes and their owned roots.
	if _, err := a.SetActiveProject(""); err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Timeout: time.Second}
	for _, run := range runs {
		response, err := client.Get(run.URL)
		if err != nil {
			t.Fatalf("server stopped on project change: %v", err)
		}
		body, err := io.ReadAll(response.Body)
		response.Body.Close()
		if err != nil || string(body) != "served by "+run.Proposal.Project {
			t.Fatalf("server no longer serves its own project: got %q, want %q (err=%v)", string(body), "served by "+run.Proposal.Project, err)
		}
	}
	for _, run := range a.ListAppRuns() {
		if run.Status != "running" {
			t.Fatalf("execution did not survive switching: %s", run.Status)
		}
	}
	// No native window runtime is attached in this test; exercise the real close path.
	a.ctx = nil
	if a.BeforeClose(context.Background()) {
		t.Fatal("closing was unexpectedly prevented")
	}
	for _, run := range runs {
		if _, err := client.Get(run.URL); err == nil {
			t.Fatal("server survived closing Merlin Code")
		}
	}
}

func TestRunToolOnlyProposesAndEmitsCard(t *testing.T) {
	silenceAgentEvents(t)
	a := newTestApp(t)
	a.runnerService = runner.NewService(a.workspaceService)
	if err := a.workspaceService.WriteFile("server.go", "package main\nfunc main(){}"); err != nil {
		t.Fatal(err)
	}
	tool := agent.NewRunTool(a.PrepareAppRun)
	if tool.RequiresApproval() {
		t.Fatal("preparation must not wait for launch approval")
	}
	content, err := tool.Execute(context.Background(), `{"directory":".","entry":"server.go"}`)
	if err != nil {
		t.Fatal(err)
	}
	var proposals []domain.RunProposal
	if err := json.Unmarshal([]byte(content), &proposals); err != nil || len(proposals) != 1 {
		t.Fatalf("missing proposal: %s %v", content, err)
	}
	if len(a.ListAppRuns()) != 0 {
		t.Fatal("tool started application without Play")
	}
	if _, err := a.StartAppRun("arbitrary command"); err == nil {
		t.Fatal("accepted a command string")
	}
	var emitted []domain.RunProposal
	emitChatStreamEvent = func(_ context.Context, name string, data ...interface{}) {
		if event, ok := data[0].(domain.ChatStreamEvent); ok && event.Type == "run_proposed" {
			emitted = event.RunProposals
		}
	}
	var trace []domain.ToolTraceEntry
	results, canceled := a.agentService.ExecuteTools(context.Background(), domain.ChatStreamRequest{MessageID: "m1"}, agent.NewToolRegistry(a.workspaceService, tool), []domain.ToolCall{{ID: "run1", Name: tool.Name(), Arguments: `{"directory":".","entry":"server.go"}`}}, &trace)
	if canceled || len(results) != 1 || results[0].IsError || len(emitted) != 1 || len(trace) != 1 {
		t.Fatalf("proposal event/trace missing: %#v %#v %#v", results, emitted, trace)
	}
	if trace[0].Result != content {
		t.Fatal("history cannot rehydrate the same card")
	}
	if emitted[0].Command != "go run server.go" || !strings.Contains(results[0].Content, "pulse Play") {
		t.Fatal("agent did not receive runnable proposal and Play instructions")
	}
}

func TestRunToolErrorIsReportedWithoutAPlayCard(t *testing.T) {
	silenceAgentEvents(t)
	a := newTestApp(t)
	a.runnerService = runner.NewService(a.workspaceService)
	var outcome domain.ChatStreamEvent
	cards := 0
	emitChatStreamEvent = func(_ context.Context, _ string, data ...interface{}) {
		event := data[0].(domain.ChatStreamEvent)
		if event.Type == domain.ChunkTypeToolResult {
			outcome = event
		}
		if event.Type == "run_proposed" {
			cards++
		}
	}
	var trace []domain.ToolTraceEntry
	results, canceled := a.agentService.ExecuteTools(context.Background(), domain.ChatStreamRequest{MessageID: "m1"}, agent.NewToolRegistry(a.workspaceService, agent.NewRunTool(a.PrepareAppRun)), []domain.ToolCall{{ID: "run1", Name: "propose_app_run", Arguments: `{"directory":".","entry":"missing.go"}`}}, &trace)
	if canceled || len(results) != 1 || !results[0].IsError || outcome.Error == "" || outcome.ToolCallID != "run1" || cards != 0 {
		t.Fatalf("tool failure hidden: %#v %#v", results, outcome)
	}
}

func TestRunToolProposesStandalonePython(t *testing.T) {
	silenceAgentEvents(t)
	a := newTestApp(t)
	a.runnerService = runner.NewService(a.workspaceService)
	if err := a.workspaceService.WriteFile("hola_mundo.py", "print('Hola mundo')"); err != nil {
		t.Fatal(err)
	}
	tool := agent.NewRunTool(a.PrepareAppRun)
	if !strings.Contains(tool.Definition().Description, "scripts Python") {
		t.Fatal("agent is not told scripts are supported")
	}
	var cards []domain.RunProposal
	emitChatStreamEvent = func(_ context.Context, _ string, data ...interface{}) {
		event := data[0].(domain.ChatStreamEvent)
		if event.Type == "run_proposed" {
			cards = event.RunProposals
		}
	}
	var trace []domain.ToolTraceEntry
	results, canceled := a.agentService.ExecuteTools(context.Background(), domain.ChatStreamRequest{MessageID: "m1"}, agent.NewToolRegistry(a.workspaceService, tool), []domain.ToolCall{{ID: "run1", Name: tool.Name(), Arguments: `{"directory":".","entry":"hola_mundo.py"}`}}, &trace)
	if canceled || len(results) != 1 || results[0].IsError || len(cards) != 1 || cards[0].Command != "python -u hola_mundo.py" {
		t.Fatalf("standalone script did not produce a Play card: %#v %#v", results, cards)
	}
	if len(a.ListAppRuns()) != 0 {
		t.Fatal("script started without Play")
	}
}

func TestRunToolProposesAstroFromManifest(t *testing.T) {
	a := newTestApp(t)
	a.runnerService = runner.NewService(a.workspaceService)
	if err := a.workspaceService.WriteFile("package.json", `{"scripts":{"site:dev":"astro dev","build":"astro build"},"dependencies":{"astro":"*"}}`); err != nil {
		t.Fatal(err)
	}
	if err := a.workspaceService.WriteFile("pnpm-lock.yaml", "lockfileVersion: '9.0'"); err != nil {
		t.Fatal(err)
	}
	content, err := (agent.NewRunTool(a.PrepareAppRun)).Execute(context.Background(), `{"directory":".","entry":"package.json"}`)
	if err != nil {
		t.Fatal(err)
	}
	var proposals []domain.RunProposal
	if err := json.Unmarshal([]byte(content), &proposals); err != nil || len(proposals) != 1 || proposals[0].Command != "pnpm run site:dev" {
		t.Fatalf("tool did not inspect the manifest and lockfile: %s %v", content, err)
	}
	if proposals[0].Request.Entry != "" || len(a.ListAppRuns()) != 0 {
		t.Fatal("manifest was treated as an executable or started without Play")
	}
}

func TestMultipleRunProposalsRequestComparison(t *testing.T) {
	silenceAgentEvents(t)
	a := newTestApp(t)
	a.runnerService = runner.NewService(a.workspaceService)
	if err := a.workspaceService.WriteFile("package.json", `{"packageManager":"pnpm@10.0.0","scripts":{"dev":"astro dev","preview":"astro preview"}}`); err != nil {
		t.Fatal(err)
	}
	var cards []domain.RunProposal
	emitChatStreamEvent = func(_ context.Context, _ string, data ...interface{}) {
		if event, ok := data[0].(domain.ChatStreamEvent); ok && event.Type == "run_proposed" {
			cards = event.RunProposals
		}
	}
	var trace []domain.ToolTraceEntry
	results, canceled := a.agentService.ExecuteTools(context.Background(), domain.ChatStreamRequest{}, agent.NewToolRegistry(a.workspaceService, agent.NewRunTool(a.PrepareAppRun)), []domain.ToolCall{{ID: "alternatives", Name: "propose_app_run", Arguments: `{"directory":"."}`}}, &trace)
	if canceled || len(results) != 1 || results[0].IsError || len(cards) != 2 {
		t.Fatalf("missing alternative cards: %#v %#v", results, cards)
	}
	if !strings.HasPrefix(cards[0].Reason, "Desarrollo:") || !strings.HasPrefix(cards[1].Reason, "Vista previa:") {
		t.Fatal("cards do not explain their different purposes")
	}
	for _, instruction := range []string{"Hay varias alternativas", "cuándo elegirlo", "Recomienda una opción", "Son alternativas, no pasos", "compilación previa"} {
		if !strings.Contains(results[0].Content, instruction) {
			t.Fatalf("missing model instruction %q", instruction)
		}
	}
	var saved []domain.RunProposal
	if len(trace) != 1 || json.Unmarshal([]byte(trace[0].Result), &saved) != nil || len(saved) != 2 || saved[1].Reason != cards[1].Reason {
		t.Fatal("history does not preserve the card explanations")
	}
	if len(a.ListAppRuns()) != 0 {
		t.Fatal("alternatives started without Play")
	}
}
