package repomap

import (
	"context"
	"testing"
)

const sampleGoSource = `package sample

import "fmt"

// Greet dice hola.
func Greet(name string) string {
	return "hello " + name
}

func internalHelper() {}

type Config struct {
	Name string
}

type privateHelper struct{}

const MaxRetries = 3

const (
	first = 1
	Second = 2
)

var GlobalCounter int

var A, B = 1, 2

func (c *Config) Describe() string {
	return c.Name
}

func main() {
	fmt.Println(Greet("world"))
}
`

// TestExtractGoSymbolsParseaASTReal es el test exigido por la tarea: no usa mocks ni heurísticas
// de texto. Instancia go.wasm (motor núcleo de tree-sitter + gramática de tree-sitter-go)
// mediante el runtime wazero puro-Go, le hace parsear sampleGoSource de verdad y verifica los
// símbolos exportados que produce caminando el AST real — exactamente el mismo camino de código
// que usa Service.Snapshot en producción.
func TestExtractGoSymbolsParseaASTReal(t *testing.T) {
	ctx := context.Background()
	rt, err := newRuntime(ctx)
	if err != nil {
		t.Fatalf("newRuntime falló: %v", err)
	}
	defer rt.Close(ctx)

	p, err := newParser(ctx, rt, goWasm)
	if err != nil {
		t.Fatalf("newParser falló: %v", err)
	}

	symbols, err := extractGoSymbols(ctx, p, []byte(sampleGoSource))
	if err != nil {
		t.Fatalf("extractGoSymbols falló: %v", err)
	}

	got := map[string]string{}
	lines := map[string]int{}
	for _, sym := range symbols {
		got[sym.Name] = sym.Kind
		lines[sym.Name] = sym.Line
	}

	want := map[string]string{
		"Greet":           "func",
		"Config":          "type",
		"MaxRetries":      "const",
		"Second":          "const",
		"GlobalCounter":   "var",
		"A":               "var",
		"B":               "var",
		"Config.Describe": "method",
	}
	for name, kind := range want {
		gotKind, ok := got[name]
		if !ok {
			t.Fatalf("esperaba el símbolo %q (kind=%s) en %v", name, kind, got)
		}
		if gotKind != kind {
			t.Fatalf("símbolo %q: esperaba kind=%s, obtenido=%s", name, kind, gotKind)
		}
	}

	// internalHelper, privateHelper y first no son exportados (minúscula inicial): no deben
	// aparecer. main tampoco se filtra aquí porque no es exportado (empieza en minúscula).
	for _, unexported := range []string{"internalHelper", "privateHelper", "first", "main"} {
		if _, ok := got[unexported]; ok {
			t.Fatalf("no debió incluirse el símbolo no exportado %q", unexported)
		}
	}

	// Greet está declarado en la línea 6 de sampleGoSource (1-based); confirma que startRow/+1
	// se está leyendo del nodo real y no de un valor arbitrario.
	if lines["Greet"] != 6 {
		t.Fatalf("esperaba Greet en la línea 6, obtenido %d", lines["Greet"])
	}
}

// TestExtractGoSymbolsReusaElParser confirma que el mismo *parser (y su TSParser* interno,
// reutilizado vía ensureTSParser) puede parsear múltiples archivos distintos en secuencia sin
// corromper memoria ni devolver resultados de un parseo anterior — el patrón real que usa
// Service.build al recorrer varios archivos del proyecto con la misma instancia de lenguaje.
func TestExtractGoSymbolsReusaElParser(t *testing.T) {
	ctx := context.Background()
	rt, err := newRuntime(ctx)
	if err != nil {
		t.Fatalf("newRuntime falló: %v", err)
	}
	defer rt.Close(ctx)

	p, err := newParser(ctx, rt, goWasm)
	if err != nil {
		t.Fatalf("newParser falló: %v", err)
	}

	first, err := extractGoSymbols(ctx, p, []byte("package a\n\nfunc Foo() {}\n"))
	if err != nil {
		t.Fatalf("primer parseo falló: %v", err)
	}
	if len(first) != 1 || first[0].Name != "Foo" {
		t.Fatalf("esperaba [Foo], obtenido %v", first)
	}

	second, err := extractGoSymbols(ctx, p, []byte("package b\n\nfunc Bar() {}\nfunc Baz() {}\n"))
	if err != nil {
		t.Fatalf("segundo parseo falló: %v", err)
	}
	names := map[string]bool{}
	for _, s := range second {
		names[s.Name] = true
	}
	if !names["Bar"] || !names["Baz"] || names["Foo"] {
		t.Fatalf("esperaba [Bar Baz] sin arrastrar Foo del parseo anterior, obtenido %v", second)
	}
}
