// Package lint runs best-effort, read-only static analysis (go vet, tsc --noEmit) right after
// the agent applies a write_file/edit_file change, so the model learns about broken syntax or
// types in the same turn. This is diagnostic only: it never mutates the project and never starts
// or supervises the user's own program, so unlike propose_app_run it requires no approval or
// explicit Play action.
package lint

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"merlincode/internal/workspace"
)

const (
	timeout   = 20 * time.Second
	maxOutput = 8 * 1024
)

// Check inspects editedPath's extension and runs the matching diagnostic, if its toolchain is
// available. ran reports whether a diagnostic actually executed; output is empty when it ran
// clean. Errors from missing/unavailable toolchains are intentionally swallowed: linting is a
// bonus signal for the model, not a required capability.
func Check(ctx context.Context, ws *workspace.Service, editedPath string) (output string, ran bool) {
	switch strings.ToLower(filepath.Ext(editedPath)) {
	case ".go":
		return checkGo(ctx, ws)
	case ".ts", ".tsx", ".js", ".jsx":
		return checkTypeScript(ctx, ws, editedPath)
	default:
		return "", false
	}
}

func checkGo(ctx context.Context, ws *workspace.Service) (string, bool) {
	if _, err := ws.ReadFile("go.mod"); err != nil {
		return "", false
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		return "", false
	}
	return run(ctx, ws.GetActiveDir(), goBin, []string{"vet", "./..."})
}

func checkTypeScript(ctx context.Context, ws *workspace.Service, editedPath string) (string, bool) {
	dir := nearestTSConfigDir(ws, editedPath)
	if dir == "" {
		return "", false
	}
	absDir := filepath.Join(ws.GetActiveDir(), filepath.FromSlash(dir))
	tsc := filepath.Join(absDir, "node_modules", "typescript", "bin", "tsc")
	if _, err := os.Stat(tsc); err != nil {
		return "", false
	}
	nodeBin, err := exec.LookPath("node")
	if err != nil {
		return "", false
	}
	return run(ctx, absDir, nodeBin, []string{tsc, "--noEmit"})
}

// nearestTSConfigDir walks up from editedPath's directory to the project root looking for a
// tsconfig.json, reusing ws.ReadFile so the lookup stays inside the project sandbox.
func nearestTSConfigDir(ws *workspace.Service, editedPath string) string {
	dir := filepath.ToSlash(filepath.Dir(editedPath))
	for {
		candidate := "tsconfig.json"
		if dir != "." && dir != "" {
			candidate = dir + "/tsconfig.json"
		}
		if _, err := ws.ReadFile(candidate); err == nil {
			if dir == "" {
				dir = "."
			}
			return dir
		}
		if dir == "." || dir == "" {
			return ""
		}
		dir = filepath.ToSlash(filepath.Dir(dir))
	}
}

func run(ctx context.Context, dir, exe string, args []string) (string, bool) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.Dir = dir
	cmd.Env = sanitizedEnv()
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err == nil {
		return "", true
	}
	out := buf.String()
	if len(out) > maxOutput {
		out = out[:maxOutput] + "\n... (salida truncada)"
	}
	if strings.TrimSpace(out) == "" {
		out = "El comando de diagnóstico falló sin salida."
	}
	return out, true
}

// sanitizedEnv strips provider credentials and tokens before spawning go/node, independent of
// internal/runner's own filtering, since this package must not depend on its unexported helpers.
func sanitizedEnv() []string {
	var env []string
	for _, item := range os.Environ() {
		key, _, _ := strings.Cut(item, "=")
		upper := strings.ToUpper(key)
		if strings.Contains(upper, "API_KEY") || strings.Contains(upper, "TOKEN") || strings.Contains(upper, "SECRET") || strings.Contains(upper, "PASSWORD") {
			continue
		}
		env = append(env, item)
	}
	env = append(env, "NO_COLOR=1", "GOFLAGS=-mod=readonly", "GOPROXY=off", "GOSUMDB=off", "GOTOOLCHAIN=local")
	return env
}
