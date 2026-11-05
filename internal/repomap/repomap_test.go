package repomap

import (
	"os"
	"strings"
	"testing"

	"merlincode/internal/workspace"
)

func newProject(t *testing.T) *workspace.Service {
	t.Helper()
	dir, err := os.MkdirTemp("", "merlin_repomap_test_*")
	if err != nil {
		t.Fatalf("MkdirTemp falló: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	svc := workspace.NewService()
	t.Cleanup(func() { _ = svc.Close() })
	if _, err := svc.SetActive(dir); err != nil {
		t.Fatalf("SetActive falló: %v", err)
	}
	return svc
}

func TestSnapshot_NoActiveProject_ReturnsEmpty(t *testing.T) {
	svc := NewService(workspace.NewService())
	if got := svc.Snapshot(); got != "" {
		t.Fatalf("esperado vacío sin proyecto activo, obtenido: %q", got)
	}
}

func TestSnapshot_NoSupportedFiles_ReturnsEmpty(t *testing.T) {
	ws := newProject(t)
	if err := ws.WriteFile("README.md", "# hola"); err != nil {
		t.Fatalf("WriteFile falló: %v", err)
	}
	svc := NewService(ws)
	if got := svc.Snapshot(); got != "" {
		t.Fatalf("esperado vacío sin archivos .go/.ts, obtenido: %q", got)
	}
}

func TestSnapshot_GoFile_ListsExportedSymbolsOnly(t *testing.T) {
	ws := newProject(t)
	src := `package pkg

type Widget struct{}

type helper struct{}

func (w *Widget) Render() string { return "" }

func Public() {}

func private() {}

var MaxSize = 10
var minSize = 1
`
	if err := ws.WriteFile("widget.go", src); err != nil {
		t.Fatalf("WriteFile falló: %v", err)
	}
	svc := NewService(ws)
	got := svc.Snapshot()
	for _, want := range []string{"widget.go", "struct Widget", "func Widget.Render", "func Public", "MaxSize"} {
		if !strings.Contains(got, want) {
			t.Fatalf("esperado que el mapa contenga %q, obtenido: %q", want, got)
		}
	}
	for _, unwanted := range []string{"helper", "private", "minSize"} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("símbolo no exportado %q no debió aparecer en el mapa: %q", unwanted, got)
		}
	}
}

func TestSnapshot_TypeScriptFile_HeuristicExports(t *testing.T) {
	ws := newProject(t)
	src := "export function useChat() {}\n" +
		"export class ChatStore {}\n" +
		"export interface ChatMessage {}\n" +
		"function helper() {}\n"
	if err := ws.WriteFile("frontend/src/chat.ts", src); err != nil {
		t.Fatalf("WriteFile falló: %v", err)
	}
	svc := NewService(ws)
	got := svc.Snapshot()
	for _, want := range []string{"frontend/src/chat.ts", "function useChat", "class ChatStore", "interface ChatMessage"} {
		if !strings.Contains(got, want) {
			t.Fatalf("esperado que el mapa contenga %q, obtenido: %q", want, got)
		}
	}
	if strings.Contains(got, "helper") {
		t.Fatalf("una función no exportada no debió aparecer: %q", got)
	}
}

func TestRegexSymbols_MultiLanguage(t *testing.T) {
	cases := []struct {
		name   string
		ext    string
		src    string
		want   []string
		unwant []string
	}{
		{
			name:   "python",
			ext:    ".py",
			src:    "def public_fn():\n    pass\n\ndef _private_fn():\n    pass\n\nclass Widget:\n    def method(self):\n        pass\n",
			want:   []string{"def public_fn", "class Widget"},
			unwant: []string{"_private_fn"},
		},
		{
			name:   "rust",
			ext:    ".rs",
			src:    "pub fn public_fn() {}\nfn private_fn() {}\npub struct Widget {}\npub(crate) enum Mode {}\n",
			want:   []string{"fn public_fn", "struct Widget", "enum Mode"},
			unwant: []string{"private_fn"},
		},
		{
			name:   "java",
			ext:    ".java",
			src:    "public class Widget {\n    public void render() {}\n}\nclass Internal {}\n",
			want:   []string{"class Widget"},
			unwant: []string{"Internal"},
		},
		{
			name:   "kotlin",
			ext:    ".kt",
			src:    "fun publicFn() {}\nprivate fun privateFn() {}\nclass Widget\n",
			want:   []string{"fun publicFn", "class Widget"},
			unwant: []string{"privateFn"},
		},
		{
			name:   "csharp",
			ext:    ".cs",
			src:    "public class Widget {}\ninternal class Internal {}\npublic partial struct Point {}\n",
			want:   []string{"class Widget", "struct Point"},
			unwant: []string{"Internal"},
		},
		{
			name:   "php",
			ext:    ".php",
			src:    "<?php\nclass Widget {}\nfunction publicFn() {}\n",
			want:   []string{"class Widget", "function publicFn"},
			unwant: nil,
		},
		{
			name:   "ruby",
			ext:    ".rb",
			src:    "class Widget\n  def method_name\n  end\nend\nmodule Helpers\nend\n",
			want:   []string{"class Widget", "module Helpers", "def method_name"},
			unwant: nil,
		},
		{
			name:   "swift",
			ext:    ".swift",
			src:    "public class Widget {}\nprivate func helper() {}\npublic func publicFn() {}\n",
			want:   []string{"class Widget", "func publicFn"},
			unwant: []string{"helper"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			symbols := regexSymbols(tc.ext, tc.src)
			joined := strings.Join(symbols, ", ")
			for _, want := range tc.want {
				if !strings.Contains(joined, want) {
					t.Fatalf("esperado %q en los símbolos de %s, obtenido: %q", want, tc.ext, joined)
				}
			}
			for _, unwanted := range tc.unwant {
				if strings.Contains(joined, unwanted) {
					t.Fatalf("%q no debió aparecer en los símbolos de %s: %q", unwanted, tc.ext, joined)
				}
			}
		})
	}
}

func TestSnapshot_CachesUntilInvalidated(t *testing.T) {
	ws := newProject(t)
	if err := ws.WriteFile("a.go", "package pkg\n\nfunc A() {}\n"); err != nil {
		t.Fatalf("WriteFile falló: %v", err)
	}
	svc := NewService(ws)
	first := svc.Snapshot()
	if !strings.Contains(first, "func A") {
		t.Fatalf("esperado func A en el primer snapshot: %q", first)
	}

	if err := ws.WriteFile("b.go", "package pkg\n\nfunc B() {}\n"); err != nil {
		t.Fatalf("WriteFile falló: %v", err)
	}
	stale := svc.Snapshot()
	if strings.Contains(stale, "func B") {
		t.Fatalf("el snapshot no debió reflejar un archivo nuevo antes de invalidar: %q", stale)
	}

	svc.Invalidate()
	fresh := svc.Snapshot()
	if !strings.Contains(fresh, "func B") {
		t.Fatalf("esperado func B tras invalidar: %q", fresh)
	}
}

func TestSnapshot_SwitchingProjectRebuilds(t *testing.T) {
	ws := newProject(t)
	if err := ws.WriteFile("a.go", "package pkg\n\nfunc A() {}\n"); err != nil {
		t.Fatalf("WriteFile falló: %v", err)
	}
	svc := NewService(ws)
	if got := svc.Snapshot(); !strings.Contains(got, "func A") {
		t.Fatalf("esperado func A: %q", got)
	}

	otherDir, err := os.MkdirTemp("", "merlin_repomap_test_other_*")
	if err != nil {
		t.Fatalf("MkdirTemp falló: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(otherDir) })
	if _, err := ws.SetActive(otherDir); err != nil {
		t.Fatalf("SetActive falló: %v", err)
	}
	if got := svc.Snapshot(); got != "" {
		t.Fatalf("esperado vacío en el proyecto nuevo sin invalidar explícitamente, obtenido: %q", got)
	}
}
