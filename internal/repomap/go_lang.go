package repomap

import (
	"context"
	_ "embed"
	"unicode"
	"unicode/utf8"
)

// goWasm es el módulo compilado por wasmsrc/build.sh: el motor núcleo de tree-sitter enlazado
// estáticamente con la gramática de tree-sitter-go (ver la cabecera de runtime.go para el porqué
// de esta arquitectura en vez de replicar el binding de navegador).
//
//go:embed wasmsrc/go.wasm
var goWasm []byte

// collectFieldNamed devuelve, en orden, los hijos directos de node cuyo nombre de campo sea
// fieldName. Se usa ts_node_field_name_for_child por cada hijo (no
// ts_node_child_by_field_name) porque la gramática de Go reutiliza el mismo nombre de campo
// ("name") para varios hermanos en declaraciones como `const A, B = 1, 2`, y
// child_by_field_name solo devolvería la primera coincidencia. El llamador es responsable de
// liberar (astNode.free) cada nodo devuelto.
func collectFieldNamed(ctx context.Context, node astNode, fieldName string) []astNode {
	var out []astNode
	count := node.childCount(ctx)
	for i := uint32(0); i < count; i++ {
		if node.fieldNameForChild(ctx, i) != fieldName {
			continue
		}
		child, err := node.child(ctx, i)
		if err != nil {
			continue
		}
		out = append(out, child)
	}
	return out
}

// firstFieldNamed devuelve el primer hijo con ese nombre de campo (para campos de un solo valor
// como "receiver" o "type"), liberando cualquier coincidencia adicional. El llamador debe liberar
// el nodo devuelto.
func firstFieldNamed(ctx context.Context, node astNode, fieldName string) (astNode, bool) {
	matches := collectFieldNamed(ctx, node, fieldName)
	if len(matches) == 0 {
		return astNode{}, false
	}
	for _, extra := range matches[1:] {
		extra.free(ctx)
	}
	return matches[0], true
}

// childrenOfType devuelve los hijos directos de node cuyo tipo de nodo sea nodeType (p. ej. los
// type_spec/const_spec/var_spec dentro de una declaración, posiblemente agrupada entre
// paréntesis). El llamador debe liberar cada nodo devuelto.
func childrenOfType(ctx context.Context, node astNode, nodeType string) []astNode {
	var out []astNode
	count := node.childCount(ctx)
	for i := uint32(0); i < count; i++ {
		child, err := node.child(ctx, i)
		if err != nil {
			continue
		}
		if child.typ(ctx) == nodeType {
			out = append(out, child)
		} else {
			child.free(ctx)
		}
	}
	return out
}

// isExportedGoName replica la regla de exportación real de Go: un identificador es exportado si
// su primer rune es una letra mayúscula Unicode.
func isExportedGoName(name string) bool {
	if name == "" {
		return false
	}
	r, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(r)
}

// trimReceiverType limpia el texto fuente de un campo "type" de receptor (p. ej. "*Config" o
// "Config") a solo el nombre del tipo, para mostrarlo como "Config.Method" en el mapa.
func trimReceiverType(raw string) string {
	for len(raw) > 0 && (raw[0] == '*' || raw[0] == ' ' || raw[0] == '\t') {
		raw = raw[1:]
	}
	return raw
}

// extractGoSymbols parsea src con la gramática de Go real (vía wazero + go.wasm) y devuelve los
// símbolos exportados declarados en el nivel superior del archivo: funciones, métodos (como
// "Receptor.Método"), tipos, constantes y variables. No es una heurística de texto: camina el AST
// real que produce tree-sitter-go, incluyendo el manejo de declaraciones agrupadas
// (`const (...)`, `type (...)`) y de múltiples nombres por spec (`const A, B = 1, 2`).
func extractGoSymbols(ctx context.Context, p *parser, src []byte) ([]Symbol, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	var symbols []Symbol

	addName := func(nameNode astNode, kind string) {
		name := nameNode.text(ctx, src)
		if isExportedGoName(name) {
			symbols = append(symbols, Symbol{
				Name: name,
				Kind: kind,
				Line: int(nameNode.startRow(ctx)) + 1,
			})
		}
	}

	err := p.parseTopLevel(ctx, src, func(decl astNode) {
		switch decl.typ(ctx) {
		case "function_declaration":
			if nameNode, ok := firstFieldNamed(ctx, decl, "name"); ok {
				addName(nameNode, "func")
				nameNode.free(ctx)
			}

		case "method_declaration":
			nameNode, ok := firstFieldNamed(ctx, decl, "name")
			if !ok {
				return
			}
			methodName := nameNode.text(ctx, src)
			nameNode.free(ctx)
			if !isExportedGoName(methodName) {
				return
			}
			display := methodName
			if receiver, ok := firstFieldNamed(ctx, decl, "receiver"); ok {
				// receiver es el parameter_list completo, p. ej. "(c *Config)"; el campo "type"
				// vive un nivel más abajo, en su único parameter_declaration hijo.
				paramDecls := childrenOfType(ctx, receiver, "parameter_declaration")
				if len(paramDecls) > 0 {
					if recvType, ok := firstFieldNamed(ctx, paramDecls[0], "type"); ok {
						if t := trimReceiverType(recvType.text(ctx, src)); t != "" {
							display = t + "." + methodName
						}
						recvType.free(ctx)
					}
				}
				for _, pd := range paramDecls {
					pd.free(ctx)
				}
				receiver.free(ctx)
			}
			symbols = append(symbols, Symbol{Name: display, Kind: "method", Line: int(decl.startRow(ctx)) + 1})

		case "type_declaration":
			for _, spec := range append(childrenOfType(ctx, decl, "type_spec"), childrenOfType(ctx, decl, "type_alias")...) {
				if nameNode, ok := firstFieldNamed(ctx, spec, "name"); ok {
					addName(nameNode, "type")
					nameNode.free(ctx)
				}
				spec.free(ctx)
			}

		case "const_declaration":
			for _, spec := range childrenOfType(ctx, decl, "const_spec") {
				for _, nameNode := range collectFieldNamed(ctx, spec, "name") {
					addName(nameNode, "const")
					nameNode.free(ctx)
				}
				spec.free(ctx)
			}

		case "var_declaration":
			for _, spec := range childrenOfType(ctx, decl, "var_spec") {
				for _, nameNode := range collectFieldNamed(ctx, spec, "name") {
					addName(nameNode, "var")
					nameNode.free(ctx)
				}
				spec.free(ctx)
			}
		}
	})
	if err != nil {
		return nil, err
	}
	return symbols, nil
}
