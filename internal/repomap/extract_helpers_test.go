package repomap

import (
	"context"
	"testing"
)

// extractWithRealWasm instancia wasmBytes (uno de los módulos embebidos reales, nunca un mock)
// en un runtime wazero nuevo y ejecuta extract sobre src: el mismo camino de código que usa
// Service.build en producción.
func extractWithRealWasm(t *testing.T, wasmBytes []byte, extract func(context.Context, *parser, []byte) ([]Symbol, error), src string) []Symbol {
	t.Helper()
	ctx := context.Background()
	rt, err := newRuntime(ctx)
	if err != nil {
		t.Fatalf("newRuntime falló: %v", err)
	}
	t.Cleanup(func() { rt.Close(ctx) })

	p, err := newParser(ctx, rt, wasmBytes)
	if err != nil {
		t.Fatalf("newParser falló: %v", err)
	}
	symbols, err := extract(ctx, p, []byte(src))
	if err != nil {
		t.Fatalf("la extracción falló: %v", err)
	}
	return symbols
}

// assertExactSymbols exige que symbols contenga exactamente los nombres de want con su kind: ni
// faltantes ni sobrantes (un símbolo privado que se cuele también hace fallar el test).
func assertExactSymbols(t *testing.T, symbols []Symbol, want map[string]string) {
	t.Helper()
	got := map[string]string{}
	for _, s := range symbols {
		if prev, dup := got[s.Name]; dup {
			t.Errorf("símbolo duplicado %q (kinds %s y %s)", s.Name, prev, s.Kind)
		}
		got[s.Name] = s.Kind
	}
	for name, kind := range want {
		gotKind, ok := got[name]
		if !ok {
			t.Errorf("falta el símbolo %q (kind=%s)", name, kind)
			continue
		}
		if gotKind != kind {
			t.Errorf("símbolo %q: esperaba kind=%s, obtenido=%s", name, kind, gotKind)
		}
	}
	for name, kind := range got {
		if _, ok := want[name]; !ok {
			t.Errorf("símbolo inesperado %q (kind=%s)", name, kind)
		}
	}
	if t.Failed() {
		t.Logf("símbolos obtenidos: %+v", symbols)
	}
}

// symbolLine devuelve la línea (1-based) del primer símbolo con ese nombre, o 0.
func symbolLine(symbols []Symbol, name string) int {
	for _, s := range symbols {
		if s.Name == name {
			return s.Line
		}
	}
	return 0
}
