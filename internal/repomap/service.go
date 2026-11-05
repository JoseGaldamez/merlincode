package repomap

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"
	"sync"

	"github.com/tetratelabs/wazero"

	"merlincode/internal/workspace"
)

// Symbol es un símbolo exportado/público extraído del AST real (no heurístico) de un archivo
// fuente del proyecto.
type Symbol struct {
	Name string
	Kind string // depende del lenguaje: "func", "method", "type", "const", "var", "class", "fn", ...
	Line int    // 1-based
}

// languageSupport asocia una extensión de archivo con su módulo wasm compilado (motor núcleo de
// tree-sitter + una gramática, ver wasmsrc/build.sh) y la función que sabe caminar el AST de ese
// lenguaje. Agregar un lenguaje nuevo es: compilar otro .wasm con el mismo esquema, embeberlo con
// go:embed, y agregar una entrada aquí — ni runtime.go ni el resto de service.go necesitan
// cambios, ya que ambos son agnósticos al lenguaje.
type languageSupport struct {
	ext     string
	wasm    []byte
	extract func(ctx context.Context, p *parser, src []byte) ([]Symbol, error)
}

var supportedLanguages = []languageSupport{
	{ext: ".go", wasm: goWasm, extract: extractGoSymbols},
	// JavaScript y JSX comparten gramática (tree-sitter-javascript incluye JSX). Como parserFor
	// indexa las instancias por extensión, .js y .jsx instancian cada una su propia copia del
	// mismo módulo; es correcto, solo cuesta una instancia wasm extra si el proyecto tiene ambas.
	{ext: ".js", wasm: javascriptWasm, extract: extractJSTSSymbols},
	{ext: ".jsx", wasm: javascriptWasm, extract: extractJSTSSymbols},
	// TypeScript y TSX son gramáticas distintas (ver typescript_lang.go), con el mismo extractor.
	{ext: ".ts", wasm: typescriptWasm, extract: extractJSTSSymbols},
	{ext: ".tsx", wasm: tsxWasm, extract: extractJSTSSymbols},
	{ext: ".py", wasm: pythonWasm, extract: extractPythonSymbols},
	{ext: ".rs", wasm: rustWasm, extract: extractRustSymbols},
}

const (
	// maxFilesPerSnapshot acota cuántos archivos se incluyen en un solo mapa, para que construirlo
	// (y el tamaño del mensaje de sistema inyectado al modelo) se mantenga acotado en proyectos
	// grandes.
	maxFilesPerSnapshot = 300
	// maxSnapshotBytes acota el tamaño del texto final del mapa por la misma razón.
	maxSnapshotBytes = 24 * 1024
)

// Service mantiene un mapa cacheado de los símbolos exportados del proyecto activo, reconstruido
// de forma perezosa solo cuando algo lo marca obsoleto vía Invalidate. El runtime wazero y las
// instancias wasm por lenguaje también se crean de forma perezosa, para no pagar ese costo si
// nunca se usa el mapa.
type Service struct {
	ws *workspace.Service

	mu      sync.Mutex
	dirty   bool
	cached  string
	rt      wazero.Runtime
	parsers map[string]*parser // por extensión, p. ej. ".go"
}

// NewService crea el servicio de mapa de repositorio para ws. No abre el proyecto ni instancia
// wasm todavía; eso ocurre en la primera llamada real a Snapshot.
func NewService(ws *workspace.Service) *Service {
	return &Service{ws: ws, dirty: true}
}

// Invalidate marca el mapa cacheado como obsoleto; la próxima llamada a Snapshot lo reconstruye
// desde el proyecto activo. Debe llamarse después de cada write_file/edit_file exitoso.
func (s *Service) Invalidate() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dirty = true
}

// Snapshot devuelve el mapa de símbolos del proyecto activo, reconstruyéndolo si está marcado
// como obsoleto. Devuelve "" si no hay nada que mostrar: sin proyecto activo, sin archivos de un
// lenguaje soportado, o si la reconstrucción falla. No hay telemetría ni logs de este fallo
// porque podrían filtrar rutas de archivos del usuario; el servicio simplemente conserva el
// último mapa bueno conocido y reintenta en la siguiente llamada.
func (s *Service) Snapshot() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	activeDir := s.ws.GetActiveDir()
	if activeDir == "" {
		// Nunca se cachea el estado "sin proyecto": así, cuando el usuario active un proyecto,
		// la siguiente llamada lo recoge sin depender de que algo más llame a Invalidate.
		s.dirty = true
		s.cached = ""
		return ""
	}

	if !s.dirty {
		return s.cached
	}

	snapshot, err := s.build(context.Background())
	if err != nil {
		return s.cached
	}
	s.cached = snapshot
	s.dirty = false
	return s.cached
}

// build reconstruye el mapa recorriendo todo el proyecto activo (sin límite de profundidad, vía
// workspace.Service.WalkFiles) en busca de archivos de los lenguajes soportados, extrayendo sus
// símbolos exportados vía AST real y formateándolos en un texto legible para humanos y modelos.
func (s *Service) build(ctx context.Context) (string, error) {
	exts := make(map[string]bool, len(supportedLanguages))
	byExt := make(map[string]languageSupport, len(supportedLanguages))
	for _, lang := range supportedLanguages {
		exts[lang.ext] = true
		byExt[lang.ext] = lang
	}

	paths, err := s.ws.WalkFiles(exts, maxFilesPerSnapshot)
	if err != nil {
		return "", err
	}
	if len(paths) == 0 {
		return "", nil
	}
	sort.Strings(paths)

	var out strings.Builder
	out.WriteString("Mapa del repositorio (símbolos exportados/públicos, extraídos con AST real de tree-sitter vía wazero):\n")
	wrote := false

	for _, filePath := range paths {
		lang, ok := byExt[strings.ToLower(path.Ext(filePath))]
		if !ok {
			continue
		}
		content, err := s.ws.ReadFile(filePath)
		if err != nil {
			// Archivo no legible (demasiado grande, symlink roto, etc.): se omite ese archivo sin
			// abortar el mapa completo.
			continue
		}
		p, err := s.parserFor(ctx, lang)
		if err != nil {
			// Ese lenguaje no pudo instanciarse en este proceso (no debería ocurrir con el .wasm
			// embebido, pero no es razón para perder el resto del mapa).
			continue
		}
		symbols, err := lang.extract(ctx, p, []byte(content))
		if err != nil || len(symbols) == 0 {
			continue
		}

		block := formatFileBlock(filePath, symbols)
		if out.Len()+len(block) > maxSnapshotBytes {
			break
		}
		out.WriteString(block)
		wrote = true
	}

	if !wrote {
		return "", nil
	}
	return out.String(), nil
}

func formatFileBlock(filePath string, symbols []Symbol) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\n%s\n", filePath)
	for _, sym := range symbols {
		fmt.Fprintf(&b, "  %s %s (línea %d)\n", sym.Kind, sym.Name, sym.Line)
	}
	return b.String()
}

// parserFor devuelve (creando de forma perezosa si es necesario) la instancia wasm reutilizable
// para el lenguaje de lang. El runtime wazero se crea una sola vez por Service y se comparte
// entre todos los lenguajes.
func (s *Service) parserFor(ctx context.Context, lang languageSupport) (*parser, error) {
	if s.rt == nil {
		rt, err := newRuntime(ctx)
		if err != nil {
			return nil, err
		}
		s.rt = rt
		s.parsers = make(map[string]*parser)
	}
	if p, ok := s.parsers[lang.ext]; ok {
		return p, nil
	}
	p, err := newParser(ctx, s.rt, lang.wasm)
	if err != nil {
		return nil, err
	}
	s.parsers[lang.ext] = p
	return p, nil
}
