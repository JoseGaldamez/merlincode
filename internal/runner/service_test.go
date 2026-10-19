package runner

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"merlincode/internal/domain"
	"merlincode/internal/workspace"
)

func testService(t *testing.T, files map[string]string) (*Service, string) {
	t.Helper()
	dir := t.TempDir()
	ws := workspace.NewService()
	if _, err := ws.SetActive(dir); err != nil {
		t.Fatal(err)
	}
	for path, content := range files {
		if err := ws.WriteFile(path, content); err != nil {
			t.Fatal(err)
		}
	}
	svc := NewService(ws)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := svc.Close(ctx); err != nil {
			t.Error(err)
		}
		if err := ws.Close(); err != nil {
			t.Error(err)
		}
	})
	return svc, dir
}

func TestDetectApplicationProfiles(t *testing.T) {
	cases := []struct {
		name     string
		files    map[string]string
		request  domain.RunRequest
		command  string
		rejected bool
	}{
		{"pnpm declared", map[string]string{"package.json": `{"packageManager":"pnpm@10.0.0","scripts":{"dev":"vite"}}`}, domain.RunRequest{}, "pnpm run dev", false},
		{"yarn lock", map[string]string{"package.json": `{"scripts":{"start":"next start"}}`, "yarn.lock": "lock"}, domain.RunRequest{}, "yarn run start", false},
		{"bun lock", map[string]string{"package.json": `{"scripts":{"dev":"astro dev"}}`, "bun.lock": "lock"}, domain.RunRequest{}, "bun run dev", false},
		{"npm lock", map[string]string{"package.json": `{"scripts":{"dev":"vite --host 127.0.0.1"}}`, "package-lock.json": "{}"}, domain.RunRequest{}, "npm run dev", false},
		{"monorepo", map[string]string{"package.json": `{"packageManager":"pnpm@10.0.0"}`, "apps/web/package.json": `{"scripts":{"dev":"vite"}}`}, domain.RunRequest{Directory: "apps/web"}, "pnpm run dev", false},
		{"independent nested package", map[string]string{"package.json": `{"packageManager":"pnpm@10"}`, "web/package.json": `{"scripts":{"dev":"vite"}}`, "web/bun.lock": "lock"}, domain.RunRequest{Directory: "web"}, "bun run dev", false},
		{"ambiguous", map[string]string{"package.json": `{"scripts":{"dev":"vite"}}`, "pnpm-lock.yaml": "lock", "yarn.lock": "lock"}, domain.RunRequest{}, "", true},
		{"no inferred manager", map[string]string{"package.json": `{"scripts":{"dev":"vite"}}`}, domain.RunRequest{}, "", true},
		{"compound command", map[string]string{"package.json": `{"packageManager":"npm@10","scripts":{"dev":"vite && echo unsafe"}}`}, domain.RunRequest{}, "", true},
		{"shell substitution", map[string]string{"package.json": `{"packageManager":"npm@10","scripts":{"dev":"vite $(whoami)"}}`}, domain.RunRequest{}, "", true},
		{"unsupported flag", map[string]string{"package.json": `{"packageManager":"npm@10","scripts":{"dev":"vite --eval=unsafe"}}`}, domain.RunRequest{}, "", true},
		{"external config", map[string]string{"package.json": `{"packageManager":"npm@10","scripts":{"dev":"vite --config=C:/outside.js"}}`}, domain.RunRequest{}, "", true},
		{"lan host", map[string]string{"package.json": `{"packageManager":"npm@10","scripts":{"dev":"vite --host 0.0.0.0"}}`}, domain.RunRequest{}, "", true},
		{"hook", map[string]string{"package.json": `{"packageManager":"npm@10","scripts":{"dev":"vite","predev":"echo unsafe"}}`}, domain.RunRequest{}, "", true},
		{"arbitrary script", map[string]string{"package.json": `{"packageManager":"npm@10","scripts":{"start":"node delete-files.js"}}`, "delete-files.js": "console.log('not a server')"}, domain.RunRequest{}, "", true},
		{"node backend", map[string]string{"package.json": `{"packageManager":"bun@1","scripts":{"start":"bun server.ts"}}`, "server.ts": "import express from 'express'; express().listen(3000)"}, domain.RunRequest{}, "bun run start", false},
		{"fastapi uv", map[string]string{"pyproject.toml": "[project]\nname='api'", "uv.lock": "version = 1", "main.py": "from fastapi import FastAPI\napi = FastAPI()"}, domain.RunRequest{}, "uv run --no-sync python -m uvicorn main:api --host 127.0.0.1 --port 8000", false},
		{"poetry flask", map[string]string{"poetry.lock": "lock", "app.py": "from flask import Flask\napp = Flask(__name__)"}, domain.RunRequest{}, "poetry run python -m flask --app app.py run --host 127.0.0.1 --port 5000", false},
		{"explicit python entry", map[string]string{"api/web.py": "from fastapi import FastAPI\napp = FastAPI()"}, domain.RunRequest{Entry: "api/web.py"}, "python -m uvicorn api.web:app --host 127.0.0.1 --port 8000", false},
		{"standalone python script", map[string]string{"hola_mundo.py": "print('Hola mundo')"}, domain.RunRequest{Entry: "hola_mundo.py"}, "python -u hola_mundo.py", false},
		{"python script with uv", map[string]string{"hola_mundo.py": "print('Hola mundo')", "uv.lock": "version = 1"}, domain.RunRequest{Entry: "hola_mundo.py"}, "uv run --no-sync python -u hola_mundo.py", false},
		{"python script with poetry", map[string]string{"hola_mundo.py": "print('Hola mundo')", "poetry.lock": "lock"}, domain.RunRequest{Entry: "hola_mundo.py"}, "poetry run python -u hola_mundo.py", false},
		{"python script with pipenv", map[string]string{"hola_mundo.py": "print('Hola mundo')", "Pipfile": "[packages]"}, domain.RunRequest{Entry: "hola_mundo.py"}, "pipenv run python -u hola_mundo.py", false},
		{"auto detect python main", map[string]string{"main.py": "print('Hola mundo')"}, domain.RunRequest{}, "python -u main.py", false},
		{"empty python script", map[string]string{"empty.py": ""}, domain.RunRequest{Entry: "empty.py"}, "python -u empty.py", false},
		{"python filename is not a flag", map[string]string{"-I.py": "print('Hola mundo')"}, domain.RunRequest{Entry: "-I.py"}, "python -u ./-I.py", false},
		{"python script in subfolder", map[string]string{"scripts/hola mundo.py": "print('Hola mundo')"}, domain.RunRequest{Entry: "scripts/hola mundo.py"}, `python -u "scripts/hola mundo.py"`, false},
		{"missing script is not a static site", map[string]string{"index.html": "hello"}, domain.RunRequest{Entry: "missing.py"}, "", true},
		{"python with js tooling", map[string]string{"package.json": `{"scripts":{"build":"tsc"}}`, "app.py": "from fastapi import FastAPI\napp = FastAPI()"}, domain.RunRequest{}, "python -m uvicorn app:app --host 127.0.0.1 --port 8000", false},
		{"django", map[string]string{"manage.py": "from django.core.management import execute_from_command_line\nexecute_from_command_line()"}, domain.RunRequest{}, "python manage.py runserver 127.0.0.1:8000 --noreload", false},
		{"go", map[string]string{"go.mod": "module app", "cmd/api/main.go": "package main\nfunc main(){}"}, domain.RunRequest{Entry: "cmd/api/main.go"}, "go run ./cmd/api", false},
		{"standalone go server", map[string]string{"server.go": "package main\nfunc main(){}"}, domain.RunRequest{Entry: "server.go"}, "go run server.go", false},
		{"auto detect standalone go server", map[string]string{"server.go": "package main\nfunc main(){}", "index.html": "hello"}, domain.RunRequest{}, "go run server.go", false},
		{"standalone main", map[string]string{"main.go": "package main\nfunc main(){}"}, domain.RunRequest{}, "go run main.go", false},
		{"go comment is not an entry", map[string]string{"server.go": "package helper\n// package main\nfunc main(){}"}, domain.RunRequest{Entry: "server.go"}, "", true},
		{"go library is not an entry", map[string]string{"server.go": "package main\nfunc helper(){}"}, domain.RunRequest{Entry: "server.go"}, "", true},
		{"static", map[string]string{"index.html": "<h1>Hi</h1>"}, domain.RunRequest{}, "merlin:serve . --host 127.0.0.1 --port auto", false},
		{"unknown", map[string]string{"README.md": "hello"}, domain.RunRequest{}, "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, _ := testService(t, tc.files)
			proposals, err := svc.Propose(tc.request)
			if tc.rejected {
				if err == nil {
					t.Fatal("unsafe or ambiguous proposal accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(proposals) != 1 || proposals[0].Command != tc.command {
				t.Fatalf("got %#v, want %q", proposals, tc.command)
			}
			if len(svc.List()) != 0 {
				t.Fatal("preparation started a process")
			}
		})
	}
}

func TestDetectAstroPackageScripts(t *testing.T) {
	cases := []struct {
		name, manager, lock, script, body, entry, directory, command string
	}{
		{name: "pnpm declaration", manager: "pnpm@10.0.0", script: "dev", body: "astro dev", command: "pnpm run dev"},
		{name: "pnpm lock", lock: "pnpm-lock.yaml", script: "dev", body: "astro dev", command: "pnpm run dev"},
		{name: "yarn lock custom script", lock: "yarn.lock", script: "site:dev", body: "astro dev", command: "yarn run site:dev"},
		{name: "npm lock custom script", lock: "package-lock.json", script: "local", body: "astro dev --port 4321", command: "npm run local"},
		{name: "npm shrinkwrap", lock: "npm-shrinkwrap.json", script: "dev", body: "astro dev", command: "npm run dev"},
		{name: "bun lock", lock: "bun.lock", script: "dev", body: "astro dev", command: "bun run dev"},
		{name: "bun binary lock", lock: "bun.lockb", script: "web", body: "astro dev", command: "bun run web"},
		{name: "declared manager takes precedence", manager: "yarn@4.0.0", lock: "pnpm-lock.yaml", script: "dev", body: "astro dev", command: "yarn run dev"},
		{name: "manifest as entry", lock: "pnpm-lock.yaml", script: "dev", body: "astro dev", entry: "package.json", command: "pnpm run dev"},
		{name: "relative manifest as entry", lock: "yarn.lock", script: "dev", body: "astro dev", entry: "./package.json", command: "yarn run dev"},
		{name: "monorepo package", manager: "pnpm@10.0.0", script: "site:dev", body: "astro dev", directory: "apps/site", command: "pnpm run site:dev"},
		{name: "build is not a launcher", lock: "pnpm-lock.yaml", script: "local", body: "astro build"},
		{name: "custom compound command rejected", lock: "pnpm-lock.yaml", script: "site:dev", body: "astro dev && echo unsafe"},
		{name: "option-like script rejected", lock: "pnpm-lock.yaml", script: "--dev", body: "astro dev"},
		{name: "unknown manager is not guessed", script: "dev", body: "astro dev"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			manifest, err := json.Marshal(map[string]any{
				"packageManager": tc.manager,
				"dependencies":   map[string]string{"astro": "*"},
				"scripts":        map[string]string{"build": "astro build", "check": "astro check", tc.script: tc.body},
			})
			if err != nil {
				t.Fatal(err)
			}
			files := map[string]string{filepath.Join(tc.directory, "package.json"): string(manifest)}
			if tc.directory != "" {
				// The app inherits the manager from the explicitly selected workspace root.
				files["package.json"] = `{"packageManager":"pnpm@10.0.0"}`
				files[filepath.Join(tc.directory, "package.json")] = `{"scripts":{"site:dev":"astro dev","build":"astro build"},"dependencies":{"astro":"*"}}`
			}
			if tc.lock != "" {
				files[tc.lock] = "lock"
			}
			svc, _ := testService(t, files)
			proposals, err := svc.Propose(domain.RunRequest{Directory: tc.directory, Entry: tc.entry})
			if tc.command == "" {
				if err == nil {
					t.Fatalf("unsupported launch was accepted: %#v", proposals)
				}
				return
			}
			if err != nil || len(proposals) != 1 || proposals[0].Command != tc.command {
				t.Fatalf("wrong Astro launch recipe: %#v %v", proposals, err)
			}
			if !strings.Contains(proposals[0].Reason, tc.body) || proposals[0].Request.Entry != "" || len(svc.List()) != 0 {
				t.Fatalf("proposal does not reflect inspected scripts or started without Play: %#v", proposals)
			}
		})
	}
}

func TestDetectJSFrameworkScripts(t *testing.T) {
	frameworks := []struct {
		name, dependency, script, body string
		vueCLI                         bool
		rejected                       bool
	}{
		{name: "React Vite", dependency: "react", script: "dev", body: "vite"},
		{name: "React Scripts", dependency: "react-scripts", script: "start", body: "react-scripts start"},
		{name: "Angular", dependency: "@angular/core", script: "start", body: "ng serve"},
		{name: "Angular configured app", dependency: "@angular/core", script: "web:dev", body: "ng serve web --configuration development --port 4201"},
		{name: "Angular short configuration", dependency: "@angular/core", script: "local", body: "ng serve -c development,staging"},
		{name: "Angular proxy", dependency: "@angular/core", script: "dev", body: "ng serve --proxy-config proxy.conf.json"},
		{name: "Vue Vite", dependency: "vue", script: "dev", body: "vite"},
		{name: "Vue CLI", dependency: "@vue/cli-service", script: "serve", body: "vue-cli-service serve", vueCLI: true},
		{name: "Vue CLI custom script", dependency: "@vue/cli-service", script: "local:web", body: "vue-cli-service serve --port 8081 --mode development", vueCLI: true},
		{name: "Next", dependency: "next", script: "dev", body: "next dev"},
		{name: "Nuxt", dependency: "nuxt", script: "dev", body: "nuxt dev"},
		{name: "Svelte Vite", dependency: "svelte", script: "dev", body: "vite"},
		{name: "Astro", dependency: "astro", script: "dev", body: "astro dev"},
		{name: "Webpack", dependency: "webpack", script: "local", body: "webpack serve"},
		{name: "Angular build rejected", dependency: "@angular/core", script: "dev", body: "ng build", rejected: true},
		{name: "Vue build rejected", dependency: "@vue/cli-service", script: "local", body: "vue-cli-service build", rejected: true},
		{name: "Angular generator rejected", dependency: "@angular/core", script: "dev", body: "ng generate component demo", rejected: true},
		{name: "Angular config escape rejected", dependency: "@angular/core", script: "local", body: "ng serve --proxy-config ../private.json", rejected: true},
	}
	for _, framework := range frameworks {
		for _, manager := range []string{"npm", "pnpm", "yarn", "bun"} {
			t.Run(framework.name+"/"+manager, func(t *testing.T) {
				manifest, err := json.Marshal(map[string]any{
					"packageManager": manager + "@1.0.0",
					"dependencies":   map[string]string{framework.dependency: "*"},
					"scripts":        map[string]string{framework.script: framework.body},
				})
				if err != nil {
					t.Fatal(err)
				}
				svc, _ := testService(t, map[string]string{"package.json": string(manifest), "proxy.conf.json": `{}`})
				proposals, err := svc.Propose(domain.RunRequest{})
				if framework.rejected {
					if err == nil {
						t.Fatalf("unsupported operation was offered as a launcher: %#v", proposals)
					}
					return
				}
				command := manager + " run " + framework.script
				if framework.vueCLI {
					if manager == "npm" {
						command += " --"
					}
					command += " --host 127.0.0.1"
				}
				if err != nil || len(proposals) != 1 || proposals[0].Command != command {
					t.Fatalf("wrong launch for framework and manager: %#v %v", proposals, err)
				}
				if !strings.Contains(proposals[0].Reason, framework.body) || len(svc.List()) != 0 {
					t.Fatal("script was not identified or started without Play")
				}
			})
		}
	}
}

func TestLaunchAlternativesHaveDistinctDescriptions(t *testing.T) {
	cases := []struct {
		name     string
		scripts  map[string]string
		prefixes map[string]string
	}{
		{"Astro", map[string]string{"dev": "astro dev", "preview": "astro preview"}, map[string]string{"dev": "Desarrollo:", "preview": "Vista previa:"}},
		{"Vite custom names", map[string]string{"start": "vite", "review": "vite preview"}, map[string]string{"start": "Desarrollo:", "review": "Vista previa:"}},
		{"Next", map[string]string{"dev": "next dev", "start": "next start"}, map[string]string{"dev": "Desarrollo:", "start": "Inicio de producción:"}},
		{"React start is development", map[string]string{"start": "react-scripts start"}, map[string]string{"start": "Desarrollo:"}},
		{"Node mode is not assumed", map[string]string{"start": "node server.js"}, map[string]string{"start": "Servidor de la aplicación:"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			manifest, err := json.Marshal(map[string]any{"packageManager": "pnpm@10.0.0", "scripts": tc.scripts})
			if err != nil {
				t.Fatal(err)
			}
			svc, _ := testService(t, map[string]string{"package.json": string(manifest), "server.js": "require('http').createServer().listen(3000)"})
			proposals, err := svc.Propose(domain.RunRequest{})
			if err != nil || len(proposals) != len(tc.scripts) {
				t.Fatalf("missing alternatives: %#v %v", proposals, err)
			}
			for _, proposal := range proposals {
				name := strings.TrimPrefix(proposal.Command, "pnpm run ")
				prefix := tc.prefixes[name]
				if prefix == "" || !strings.HasPrefix(proposal.Reason, prefix) || !strings.Contains(proposal.Reason, tc.scripts[name]) {
					t.Fatalf("incorrect purpose or missing source: %#v", proposal)
				}
				if prefix == "Vista previa:" || prefix == "Inicio de producción:" {
					if !strings.Contains(proposal.Reason, "Requiere una compilación previa") {
						t.Fatal("missing build prerequisite")
					}
				}
			}
		})
	}
}

func TestStandalonePythonScriptCompletion(t *testing.T) {
	if _, err := exec.LookPath("python"); err != nil {
		t.Skip("Python is not installed; no runtime is installed by this test")
	}
	for _, tc := range []struct {
		name, source, status, output string
		code                         int
	}{
		{"success", "print('Hola mundo desde Python')", "exited", "Hola mundo desde Python", 0},
		{"failure", "import sys\nprint('error del script',file=sys.stderr)\nsys.exit(7)", "error", "error del script", 7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, _ := testService(t, map[string]string{"hola_mundo.py": tc.source})
			proposals, err := svc.Propose(domain.RunRequest{Entry: "hola_mundo.py"})
			if err != nil {
				t.Fatal(err)
			}
			if len(proposals) != 1 || proposals[0].Command != "python -u hola_mundo.py" {
				t.Fatalf("missing script proposal: %#v", proposals)
			}
			if _, err := svc.Start(proposals[0].ID); err != nil {
				t.Fatal(err)
			}
			deadline := time.After(5 * time.Second)
			tick := time.NewTicker(10 * time.Millisecond)
			defer tick.Stop()
			for {
				run := svc.List()[0]
				if !isActive(run.Status) {
					if run.Status != tc.status || run.ExitCode == nil || *run.ExitCode != tc.code || !strings.Contains(run.Logs, tc.output) {
						t.Fatalf("unexpected script result: %#v", run)
					}
					return
				}
				select {
				case <-deadline:
					t.Fatal("short Python script did not finish")
				case <-tick.C:
				}
			}
		})
	}
}

func TestStandaloneGoServerLifecycle(t *testing.T) {
	svc, _ := testService(t, map[string]string{"server.go": `package main
import ("fmt";"net";"net/http")
func main() {
 listener,err:=net.Listen("tcp4","127.0.0.1:0");if err!=nil {panic(err)}
 fmt.Println("http://"+listener.Addr().String())
 http.Serve(listener,http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){fmt.Fprint(w,"standalone server")}))
}`})
	proposals, err := svc.Propose(domain.RunRequest{Entry: "server.go"})
	if err != nil {
		t.Fatal(err)
	}
	if len(proposals) != 1 || proposals[0].Command != "go run server.go" {
		t.Fatalf("missing runnable server proposal: %#v", proposals)
	}
	if _, err := svc.Start(proposals[0].ID); err != nil {
		t.Fatal(err)
	}
	deadline := time.After(30 * time.Second)
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	var address string
	for address == "" {
		run := svc.List()[0]
		if run.Status == "error" || run.Status == "exited" {
			t.Fatalf("server failed: %s\n%s", run.Error, run.Logs)
		}
		if start := strings.Index(run.Logs, "http://127.0.0.1:"); start >= 0 {
			if end := strings.IndexByte(run.Logs[start:], '\n'); end >= 0 {
				address = strings.TrimSpace(run.Logs[start : start+end])
			}
		}
		if address != "" {
			break
		}
		select {
		case <-deadline:
			t.Fatal("standalone server did not start")
		case <-tick.C:
		}
	}
	client := &http.Client{Timeout: time.Second}
	response, err := client.Get(address)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil || string(body) != "standalone server" {
		t.Fatalf("unexpected server response: %q %v", body, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := svc.StopAll(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Get(address); err == nil {
		t.Fatal("compiled Go child survived Stop")
	}
}

func TestProposalPathsAndStaleness(t *testing.T) {
	svc, dir := testService(t, map[string]string{"index.html": "hello"})
	for _, request := range []domain.RunRequest{{Directory: "../escape"}, {Directory: dir}, {Entry: "../main.py"}} {
		if _, err := svc.Propose(request); err == nil {
			t.Fatalf("accepted %#v", request)
		}
	}
	proposals, err := svc.Propose(domain.RunRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.ws.WriteFile("index.html", "changed"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Start(proposals[0].ID); err == nil {
		t.Fatal("stale proposal launched")
	}
	if _, err := svc.Start("invented"); err == nil {
		t.Fatal("unknown proposal launched")
	}
	other := t.TempDir()
	if _, err := svc.ws.SetActive(other); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Start(proposals[0].ID); err == nil {
		t.Fatal("proposal launched in wrong project")
	}
}

func TestStaticLifecycleAndConfinement(t *testing.T) {
	svc, dir := testService(t, map[string]string{"index.html": "<h1>Local app</h1>", ".env": "secret", "private.py": "secret"})
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.html"), []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	linked := os.Symlink(outside, filepath.Join(dir, "linked")) == nil
	proposals, err := svc.Propose(domain.RunRequest{})
	if err != nil {
		t.Fatal(err)
	}
	run, err := svc.Start(proposals[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	again, err := svc.Start(proposals[0].ID)
	if err != nil || again.URL != run.URL || len(svc.List()) != 1 {
		t.Fatal("duplicate launch")
	}
	client := &http.Client{Timeout: time.Second}
	for _, path := range []string{"/", "/.env", "/private.py", "/%2e%2e/secret.html", "/linked/secret.html"} {
		if strings.HasPrefix(path, "/linked") && !linked {
			continue
		}
		response, err := client.Get(run.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(response.Body)
		response.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if path == "/" {
			if response.StatusCode != 200 || !strings.Contains(string(body), "Local app") {
				t.Fatal("site not served")
			}
		} else if response.StatusCode != 404 {
			t.Fatalf("exposed %s", path)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := svc.StopAll(ctx); err != nil {
		t.Fatal(err)
	}
	if svc.List()[0].Status != "stopped" {
		t.Fatalf("status %#v", svc.List())
	}
	if _, err := client.Get(run.URL); err == nil {
		t.Fatal("server survived stop")
	}
	if _, err := svc.Start(proposals[0].ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.Close(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Start(proposals[0].ID); err == nil {
		t.Fatal("start after shutdown")
	}
}

func TestLogsAreBoundedAndClearIsOrdered(t *testing.T) {
	svc, _ := testService(t, map[string]string{"index.html": "hello"})
	proposal, err := svc.Propose(domain.RunRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Start(proposal[0].ID); err != nil {
		t.Fatal(err)
	}
	run := svc.runs[proposal[0].ID]
	writer := &logWriter{service: svc, run: run}
	large := []byte(strings.Repeat("line\n", maxLogBytes))
	if n, err := writer.Write(large); n != len(large) || err != nil {
		t.Fatalf("short write %d %v", n, err)
	}
	if len(svc.List()[0].Logs) > maxLogBytes {
		t.Fatal("unbounded logs")
	}
	before := svc.List()[0].Revision
	cleared, err := svc.ClearLogs(proposal[0].ID)
	if err != nil || cleared.Logs != "" || cleared.Revision <= before {
		t.Fatal("clear not ordered")
	}
	if _, err := writer.Write([]byte("new output")); err != nil {
		t.Fatal(err)
	}
	if svc.List()[0].Logs != "new output" {
		t.Fatal("old output returned")
	}
}

func TestStopFailureAllowsRetry(t *testing.T) {
	svc, _ := testService(t, nil)
	run := &execution{snapshot: domain.Execution{Status: "running"}, done: make(chan struct{})}
	svc.runs["test"] = run
	attempt := 0
	run.stop = func() error {
		attempt++
		if attempt == 1 {
			return errors.New("stop failed")
		}
		svc.finished(run, 0, nil)
		return nil
	}
	if err := svc.Stop("test"); err == nil {
		t.Fatal("missing stop failure")
	}
	if svc.List()[0].Status != "running" {
		t.Fatal("failed stop left controls stuck")
	}
	if err := svc.Stop("test"); err != nil {
		t.Fatal(err)
	}
	if svc.List()[0].Status != "stopped" {
		t.Fatal("retry did not stop")
	}
}

func TestRemoveStopsServerAndAllowsNewExplicitLaunch(t *testing.T) {
	svc, _ := testService(t, map[string]string{"index.html": "hello"})
	proposals, err := svc.Propose(domain.RunRequest{})
	if err != nil {
		t.Fatal(err)
	}
	run, err := svc.Start(proposals[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	var emitted domain.Execution
	svc.mu.Lock()
	svc.emit = func(snapshot domain.Execution) { emitted = snapshot }
	svc.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	removed, err := svc.Remove(ctx, run.Proposal.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(svc.List()) != 0 || removed.Status != "removed" || emitted.Status != "removed" || removed.Revision <= run.Revision {
		t.Fatalf("entry not removed: %#v", removed)
	}
	client := &http.Client{Timeout: time.Second}
	if response, err := client.Get(run.URL); err == nil {
		response.Body.Close()
		t.Fatal("removed server is still listening")
	}
	newRun, err := svc.Start(proposals[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if newRun.Revision <= removed.Revision || len(svc.List()) != 1 {
		t.Fatal("new launch cannot supersede deletion")
	}
}

func TestRemoveFailurePreservesEntry(t *testing.T) {
	svc, _ := testService(t, nil)
	run := &execution{snapshot: domain.Execution{Status: "running"}, done: make(chan struct{})}
	svc.runs["test"] = run
	run.stop = func() error { return errors.New("stop failed") }
	if _, err := svc.Remove(context.Background(), "test"); err == nil {
		t.Fatal("stop failure ignored")
	}
	if len(svc.List()) != 1 || run.removing {
		t.Fatal("entry removed despite failed cleanup")
	}
	run.stop = func() error { svc.finished(run, 0, nil); return nil }
	if _, err := svc.Remove(context.Background(), "test"); err != nil {
		t.Fatal(err)
	}
	if len(svc.List()) != 0 {
		t.Fatal("retry did not remove entry")
	}
}

func TestRelaunchEditedPythonFile(t *testing.T) {
	if _, err := exec.LookPath("python"); err != nil {
		t.Skip("Python is not installed")
	}
	svc, _ := testService(t, map[string]string{"hello.py": "print('first version')"})
	proposals, err := svc.Propose(domain.RunRequest{Entry: "hello.py"})
	if err != nil {
		t.Fatal(err)
	}
	first, err := svc.Start(proposals[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	svc.mu.Lock()
	done := svc.runs[first.Proposal.ID].done
	svc.mu.Unlock()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("first run did not finish")
	}
	if err := svc.ws.WriteFile("hello.py", "print('updated version')"); err != nil {
		t.Fatal(err)
	}
	fresh, err := svc.Propose(domain.RunRequest{Entry: "hello.py"})
	if err != nil {
		t.Fatal(err)
	}
	if fresh[0].ID == first.Proposal.ID || fresh[0].Command != first.Proposal.Command || fresh[0].Reason != first.Proposal.Reason {
		t.Fatal("edited source should refresh evidence for the same launch")
	}
	if _, err := svc.Remove(context.Background(), first.Proposal.ID); err != nil {
		t.Fatal(err)
	}
	second, err := svc.Start(fresh[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	svc.mu.Lock()
	done = svc.runs[second.Proposal.ID].done
	svc.mu.Unlock()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("edited run did not finish")
	}
	runs := svc.List()
	if len(runs) != 1 || !strings.Contains(runs[0].Logs, "updated version") || strings.Contains(runs[0].Logs, "first version") {
		t.Fatalf("stale or duplicate execution: %#v", runs)
	}
}

func TestExecutionEnvironmentDoesNotEnableDownloadsOrExposeKeys(t *testing.T) {
	t.Setenv("EXAMPLE_API_KEY", "never-forward-this")
	t.Setenv("NODE_OPTIONS", "--require arbitrary.js")
	t.Setenv("UV_OFFLINE", "0")
	env := strings.Join(executionEnvironment(), "\n")
	if strings.Contains(env, "never-forward-this") || strings.Contains(env, "arbitrary.js") || strings.Contains(env, "UV_OFFLINE=0") {
		t.Fatal("unsafe environment inherited")
	}
	for _, required := range []string{"UV_OFFLINE=1", "UV_PYTHON_DOWNLOADS=never", "COREPACK_ENABLE_NETWORK=0", "GOPROXY=off"} {
		if !strings.Contains(env, required) {
			t.Fatalf("missing %s", required)
		}
	}
}
