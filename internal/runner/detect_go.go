package runner

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

func (d *profileDetector) detectGo() {
	req, read := d.request, d.read
	module := len(read("go.mod")) > 0
	entries := []string{"main.go", "server.go", "app.go"}
	if req.Entry != "" {
		entries = []string{req.Entry}
	}
	for _, entry := range entries {
		if filepath.Ext(entry) != ".go" || strings.HasSuffix(entry, "_test.go") || !goApplicationEntry(entry, read(entry)) {
			continue
		}
		target := filepath.ToSlash(entry)
		reason := "Entrada Go independiente detectada en " + target + ". Puede ejecutarse sin go.mod si solo utiliza la biblioteca estándar."
		if module {
			target = "."
			if filepath.Dir(entry) != "." {
				target = "./" + filepath.ToSlash(filepath.Dir(entry))
			}
			reason = "package main, func main y go.mod detectados"
		} else if strings.HasPrefix(target, "-") {
			target = "./" + target
		}
		d.add("Aplicación Go", displayCommand("go", []string{"run", target}), reason, "go", "run", target)
		break
	}
}

// goApplicationEntry recognizes a real application, ignoring comments and string literals.
func goApplicationEntry(name string, source []byte) bool {
	if len(source) == 0 {
		return false
	}
	file, err := parser.ParseFile(token.NewFileSet(), name, source, parser.SkipObjectResolution)
	if err != nil || file.Name.Name != "main" {
		return false
	}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Recv == nil && function.Name.Name == "main" && function.Body != nil && len(function.Type.Params.List) == 0 && (function.Type.Results == nil || len(function.Type.Results.List) == 0) {
			return true
		}
	}
	return false
}
