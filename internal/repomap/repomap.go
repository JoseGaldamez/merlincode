// Package repomap builds a compact, per-file symbol summary of the active project and injects
// it as a system message at the start of every agent turn, so the model has a map of what
// exists before deciding which files to read_file. Go files get a real AST (go/parser, already
// zero-CGO like the rest of the project); every other supported language gets a regex
// heuristic (see regexLanguages), since no mature pure-Go parser exists for them and anything
// closer to a real parser (tree-sitter, etc.) would require CGO or a WASM runtime bridge this
// package does not implement. Each heuristic is anchored to the language's own way of marking
// something public (an "export"/"pub"/"public" keyword, or column-0 placement where the
// language has no such keyword) and will miss multi-line signatures, decorators on their own
// line, re-exports and unusual formatting — acceptable here because the map only has to orient
// the model, not replace read_file.
package repomap

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"merlincode/internal/workspace"
)

// maxFiles bounds how many files are scanned per build; maxBytes bounds the final injected
// text, so the map never competes meaningfully with the conversation for context budget.
const (
	maxFiles = 400
	maxBytes = 6000
)

var supportedExts = buildSupportedExts()

func buildSupportedExts() map[string]bool {
	exts := map[string]bool{".go": true}
	for ext := range regexLanguages {
		exts[ext] = true
	}
	return exts
}

// Service caches the last built map per project and only rebuilds it after Invalidate is
// called (on a successful write_file/edit_file) or the active project changes.
type Service struct {
	mu           sync.Mutex
	ws           *workspace.Service
	dirty        bool
	cache        string
	cacheProject string
}

func NewService(ws *workspace.Service) *Service {
	return &Service{ws: ws, dirty: true}
}

// Invalidate marks the cached map stale. Cheap and safe to call on every file edit.
func (s *Service) Invalidate() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dirty = true
}

// Snapshot returns the current repo map, rebuilding it first if needed. Returns "" when there
// is no active project or no supported file produced any symbol, so callers can skip injecting
// an empty system message.
func (s *Service) Snapshot() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	project := s.ws.GetActiveDir()
	if project == "" {
		return ""
	}
	if !s.dirty && s.cacheProject == project {
		return s.cache
	}
	s.cache = s.build()
	s.cacheProject = project
	s.dirty = false
	return s.cache
}

func (s *Service) build() string {
	files, err := s.ws.WalkFiles(supportedExts, maxFiles)
	if err != nil || len(files) == 0 {
		return ""
	}
	sort.Strings(files)

	var body strings.Builder
	remaining := maxBytes
	for _, path := range files {
		entry := describe(s.ws, path)
		if entry == "" {
			continue
		}
		if len(entry) > remaining {
			break
		}
		body.WriteString(entry)
		remaining -= len(entry)
	}
	if body.Len() == 0 {
		return ""
	}
	return "Mapa del repositorio (símbolos exportados por archivo, para orientarte antes de usar read_file; puede estar incompleto o truncado):\n\n" + body.String()
}

func describe(ws *workspace.Service, path string) string {
	content, err := ws.ReadFile(path)
	if err != nil {
		return ""
	}
	ext := filepath.Ext(path)
	var symbols []string
	if ext == ".go" {
		symbols = goSymbols(content)
	} else {
		symbols = regexSymbols(ext, content)
	}
	if len(symbols) == 0 {
		return ""
	}
	return path + ": " + strings.Join(symbols, ", ") + "\n"
}

// goSymbols lists exported top-level declarations using the standard library's own parser —
// no new dependency, no CGO. Unexported helpers are omitted to keep the map within budget.
func goSymbols(src string) []string {
	file, err := parser.ParseFile(token.NewFileSet(), "", src, parser.SkipObjectResolution)
	if err != nil {
		return nil
	}
	var out []string
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if !d.Name.IsExported() {
				continue
			}
			name := d.Name.Name
			if d.Recv != nil && len(d.Recv.List) > 0 {
				name = receiverTypeName(d.Recv.List[0].Type) + "." + name
			}
			out = append(out, "func "+name)
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if s.Name.IsExported() {
						out = append(out, typeKind(s.Type)+" "+s.Name.Name)
					}
				case *ast.ValueSpec:
					for _, name := range s.Names {
						if name.IsExported() {
							out = append(out, name.Name)
						}
					}
				}
			}
		}
	}
	return out
}

func receiverTypeName(expr ast.Expr) string {
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}
	return "?"
}

func typeKind(expr ast.Expr) string {
	switch expr.(type) {
	case *ast.StructType:
		return "struct"
	case *ast.InterfaceType:
		return "interface"
	default:
		return "type"
	}
}

// Every pattern below has exactly two capture groups — (kind, identifier) — so regexSymbols can
// join them uniformly regardless of language. None of these are real parsers; see the package
// doc comment for what that trade-off costs.
var (
	// tsExportPattern catches the common top-level export forms used in this project's
	// frontend; misses default exports, re-exports and destructured declarations.
	tsExportPattern = regexp.MustCompile(`(?m)^\s*export\s+(?:default\s+)?(?:abstract\s+)?(?:async\s+)?(function\*?|class|interface|type|enum|const|let|var)\s+([A-Za-z_$][\w$]*)`)
	// pyPattern matches column-0 (module-level) def/class; Python has no visibility keyword,
	// so leading-underscore names are filtered separately via underscoreFiltered.
	pyPattern = regexp.MustCompile(`(?m)^(?:async\s+)?(def|class)\s+([A-Za-z_]\w*)`)
	// rustPattern requires an explicit pub (optionally pub(crate)/pub(super)/...).
	rustPattern = regexp.MustCompile(`(?m)^\s*pub(?:\([^)]*\))?\s+(fn|struct|enum|trait|type|const|static)\s+([A-Za-z_]\w*)`)
	// javaPattern matches top-level public type declarations; methods are skipped to avoid
	// false positives from generics and multi-line signatures.
	javaPattern = regexp.MustCompile(`(?m)^\s*public\s+(?:static\s+)?(?:final\s+)?(?:abstract\s+)?(class|interface|enum|record)\s+([A-Za-z_]\w*)`)
	// kotlinPattern matches top-level declarations; Kotlin's default visibility is public, so
	// the absence of private/internal (which would appear in place of the optional keyword) is
	// what makes a line match at column 0.
	kotlinPattern = regexp.MustCompile(`(?m)^(?:public\s+)?(fun|class|interface|object)\s+([A-Za-z_]\w*)`)
	csharpPattern = regexp.MustCompile(`(?m)^\s*public\s+(?:static\s+)?(?:sealed\s+)?(?:abstract\s+)?(?:partial\s+)?(class|interface|enum|struct|record)\s+([A-Za-z_]\w*)`)
	// phpTypePattern and phpFuncPattern both carry a synthetic literal kind group so they join
	// the same way as every other language's (kind, identifier) pairs.
	phpTypePattern = regexp.MustCompile(`(?m)^(?:abstract\s+|final\s+)?(class|interface|trait)\s+([A-Za-z_]\w*)`)
	phpFuncPattern = regexp.MustCompile(`(?m)^(function)\s+([A-Za-z_]\w*)`)
	// rubyPattern allows one level of indentation since Ruby methods are typically nested
	// inside a class/module block; it cannot see dynamic `private`/`protected` calls, so a
	// method declared after one of those still matches (documented heuristic gap).
	rubyPattern  = regexp.MustCompile(`(?m)^\s{0,2}(def|class|module)\s+([A-Za-z_]\w*[!?]?)`)
	swiftPattern = regexp.MustCompile(`(?m)^\s*public\s+(?:final\s+)?(func|class|struct|enum|protocol)\s+([A-Za-z_]\w*)`)
)

// regexLanguages maps a file extension to the patterns used to extract its public/top-level
// declarations. Adding a language is just adding an entry here plus its pattern(s) above.
var regexLanguages = map[string][]*regexp.Regexp{
	".ts":    {tsExportPattern},
	".tsx":   {tsExportPattern},
	".js":    {tsExportPattern},
	".jsx":   {tsExportPattern},
	".py":    {pyPattern},
	".rs":    {rustPattern},
	".java":  {javaPattern},
	".kt":    {kotlinPattern},
	".cs":    {csharpPattern},
	".php":   {phpTypePattern, phpFuncPattern},
	".rb":    {rubyPattern},
	".swift": {swiftPattern},
}

// underscoreFiltered marks languages with no visibility keyword, where a leading underscore is
// the community convention for "not part of the public surface" instead.
var underscoreFiltered = map[string]bool{".py": true, ".rb": true}

func regexSymbols(ext, src string) []string {
	var out []string
	seen := map[string]bool{}
	for _, pattern := range regexLanguages[ext] {
		for _, match := range pattern.FindAllStringSubmatch(src, -1) {
			kind, name := match[1], match[2]
			if underscoreFiltered[ext] && strings.HasPrefix(name, "_") {
				continue
			}
			entry := kind + " " + name
			if !seen[entry] {
				seen[entry] = true
				out = append(out, entry)
			}
		}
	}
	return out
}
