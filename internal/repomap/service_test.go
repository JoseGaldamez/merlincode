package repomap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"merlincode/internal/workspace"
)

func newTestWorkspace(t *testing.T) *workspace.Service {
	t.Helper()
	ws := workspace.NewService()

	tempDir, err := os.MkdirTemp("", "merlin_repomap_test_*")
	if err != nil {
		t.Fatalf("error creando tempDir: %v", err)
	}
	// Orden de registro importante en Windows: Close() debe liberar el handle del os.Root antes
	// de que RemoveAll intente borrar el directorio temporal (t.Cleanup corre en orden LIFO).
	t.Cleanup(func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("no se pudo limpiar el directorio temporal: %v", err)
		}
	})
	t.Cleanup(func() { _ = ws.Close() })

	if _, err := ws.SetActive(tempDir); err != nil {
		t.Fatalf("SetActive falló: %v", err)
	}
	return ws
}

// TestSnapshotSinProyectoActivo confirma que Snapshot nunca intenta instanciar wasm ni devuelve
// contenido obsoleto si no hay ningún proyecto activo.
func TestSnapshotSinProyectoActivo(t *testing.T) {
	ws := workspace.NewService()
	defer ws.Close()

	s := NewService(ws)
	if got := s.Snapshot(); got != "" {
		t.Fatalf("esperaba \"\" sin proyecto activo, obtenido: %q", got)
	}
}

// TestSnapshotExtraeSimbolosRealesDeGo es un test de integración de punta a punta del contrato
// público del paquete (NewService/Snapshot) sobre un proyecto real de disco: valida que el mapa
// generado contiene los símbolos exportados de un archivo .go real, extraídos vía AST real de
// tree-sitter (no una heurística ni un mock).
func TestSnapshotExtraeSimbolosRealesDeGo(t *testing.T) {
	ws := newTestWorkspace(t)
	if err := ws.WriteFile("pkg/greeter.go", "package pkg\n\n// Greet saluda.\nfunc Greet(name string) string {\n\treturn \"hola \" + name\n}\n\nfunc internalOnly() {}\n"); err != nil {
		t.Fatalf("WriteFile falló: %v", err)
	}
	// Un archivo no-Go no debe aparecer en el mapa ni romper la extracción.
	if err := ws.WriteFile("README.md", "# demo\n"); err != nil {
		t.Fatalf("WriteFile falló: %v", err)
	}

	s := NewService(ws)
	snapshot := s.Snapshot()

	if !strings.Contains(snapshot, "pkg/greeter.go") {
		t.Fatalf("esperaba que el mapa mencionara pkg/greeter.go, obtenido:\n%s", snapshot)
	}
	if !strings.Contains(snapshot, "Greet") {
		t.Fatalf("esperaba que el mapa incluyera el símbolo exportado Greet, obtenido:\n%s", snapshot)
	}
	if strings.Contains(snapshot, "internalOnly") {
		t.Fatalf("el mapa no debió incluir el símbolo no exportado internalOnly, obtenido:\n%s", snapshot)
	}
	if strings.Contains(snapshot, "README.md") {
		t.Fatalf("el mapa no debió mencionar README.md (no es un lenguaje soportado), obtenido:\n%s", snapshot)
	}
}

// TestSnapshotExtraeSimbolosDeTodosLosLenguajes es el test de integración de punta a punta para
// los lenguajes agregados después de Go: un proyecto real en disco con un archivo por extensión
// registrada debe producir un mapa con los símbolos públicos de cada uno (cada archivo pasa por
// su propio módulo wasm real vía Service.parserFor) y sin los privados.
func TestSnapshotExtraeSimbolosDeTodosLosLenguajes(t *testing.T) {
	ws := newTestWorkspace(t)
	files := map[string]string{
		"web/util.js":        "export function jsPublic() {}\nfunction jsPrivate() {}\n",
		"web/View.jsx":       "export const JsxView = () => <div />;\nconst jsxPrivate = 1;\n",
		"web/types.ts":       "export interface TsPublic {}\nconst cast = <any>v;\ninterface TsPrivate {}\n",
		"web/Page.tsx":       "export default function TsxPage() { return <main />; }\nfunction tsxPrivate() {}\n",
		"tools/script.py":    "def py_public():\n    pass\n\ndef _py_private():\n    pass\n",
		"crates/core/lib.rs": "pub fn rs_public() {}\nfn rs_private() {}\n",
	}
	for path, content := range files {
		if err := ws.WriteFile(path, content); err != nil {
			t.Fatalf("WriteFile(%s) falló: %v", path, err)
		}
	}

	s := NewService(ws)
	snapshot := s.Snapshot()

	for path := range files {
		if !strings.Contains(snapshot, path) {
			t.Errorf("esperaba que el mapa mencionara %s", path)
		}
	}
	for _, public := range []string{"jsPublic", "JsxView", "TsPublic", "TsxPage", "py_public", "rs_public"} {
		if !strings.Contains(snapshot, public) {
			t.Errorf("esperaba el símbolo público %s en el mapa", public)
		}
	}
	for _, private := range []string{"jsPrivate", "jsxPrivate", "TsPrivate", "tsxPrivate", "_py_private", "rs_private"} {
		if strings.Contains(snapshot, private) {
			t.Errorf("el mapa no debió incluir el símbolo privado %s", private)
		}
	}
	if t.Failed() {
		t.Logf("mapa obtenido:\n%s", snapshot)
	}
}

// TestSnapshotCacheaHastaInvalidate confirma el contrato de caching: dos llamadas a Snapshot sin
// Invalidate de por medio deben devolver el mismo texto aunque el archivo subyacente cambie en
// disco; tras Invalidate, la siguiente llamada debe reconstruir y reflejar el cambio.
func TestSnapshotCacheaHastaInvalidate(t *testing.T) {
	ws := newTestWorkspace(t)
	if err := ws.WriteFile("main.go", "package main\n\nfunc Original() {}\n"); err != nil {
		t.Fatalf("WriteFile falló: %v", err)
	}

	s := NewService(ws)
	first := s.Snapshot()
	if !strings.Contains(first, "Original") {
		t.Fatalf("esperaba Original en el primer snapshot, obtenido:\n%s", first)
	}

	// Modifica el archivo directamente en disco (fuera del flujo write_file/edit_file) para
	// aislar el comportamiento de caching del de invalidación.
	absPath := filepath.Join(ws.GetActiveDir(), "main.go")
	if err := os.WriteFile(absPath, []byte("package main\n\nfunc Changed() {}\n"), 0o644); err != nil {
		t.Fatalf("no se pudo modificar el archivo en disco: %v", err)
	}

	second := s.Snapshot()
	if second != first {
		t.Fatalf("esperaba que Snapshot devolviera el mapa cacheado sin Invalidate; antes=%q después=%q", first, second)
	}

	s.Invalidate()
	third := s.Snapshot()
	if !strings.Contains(third, "Changed") {
		t.Fatalf("esperaba Changed tras Invalidate, obtenido:\n%s", third)
	}
	if strings.Contains(third, "Original") {
		t.Fatalf("no esperaba Original tras Invalidate, obtenido:\n%s", third)
	}
}
