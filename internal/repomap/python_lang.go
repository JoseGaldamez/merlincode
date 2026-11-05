package repomap

import (
	"context"
	_ "embed"
	"strings"
)

// pythonWasm es el módulo compilado por wasmsrc/build.sh con la gramática de tree-sitter-python
// y su external scanner en C (necesario para INDENT/DEDENT, strings y comentarios).
//
//go:embed wasmsrc/python.wasm
var pythonWasm []byte

// isPublicPythonName aplica la convención de Python para nombres públicos de un módulo: no
// empiezan con guion bajo. Es una convención (PEP 8, y lo que `from m import *` respeta cuando
// no hay __all__), no una regla del lenguaje: Python no tiene modificadores de visibilidad y
// nada impide importar un nombre que empiece con `_`.
func isPublicPythonName(name string) bool {
	return name != "" && !strings.HasPrefix(name, "_")
}

// extractPythonSymbols parsea src con la gramática real de Python (vía wazero + python.wasm) y
// devuelve los nombres públicos definidos en el nivel superior del módulo. Camina el AST real que
// produce tree-sitter-python (no regex), pero el criterio de "público" es necesariamente una
// heurística basada en convención, más débil que la de Go: Python no tiene palabra clave de
// exportación, así que se considera público todo nombre de nivel superior que no empiece con `_`
// (ver isPublicPythonName). En consecuencia también se omiten los nombres dunder como
// `__version__`.
//
// Cubre: `def` / `async def` (kind "func"), `class` (kind "class"), ambos también detrás de
// decoradores; asignaciones a nombres simples (`X = 1`, `x: int = 5`, `y: str` sin valor,
// `a, b = ...`, `[a, b] = ...`, encadenadas `a = b = 0`; kind "var") y `type Alias = ...`
// (Python 3.12, kind "type").
//
// Limitaciones explícitas: no lee `__all__` (si existe, no se usa para filtrar ni para añadir
// nombres); no desciende dentro de bloques `if`/`try`/`with` de nivel superior (p. ej. las
// definiciones bajo `if TYPE_CHECKING:` no se reportan); no reporta métodos de clases; ignora
// asignaciones a atributos o subíndices (`obj.x = 1`), que no definen nombres del módulo; e
// ignora los nombres importados (`from x import y`), aunque técnicamente también son atributos
// del módulo.
func extractPythonSymbols(ctx context.Context, p *parser, src []byte) ([]Symbol, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	var symbols []Symbol

	addName := func(name string, row uint32, kind string) {
		if isPublicPythonName(name) {
			symbols = append(symbols, Symbol{Name: name, Kind: kind, Line: int(row) + 1})
		}
	}
	add := func(nameNode astNode, kind string) {
		addName(nameNode.text(ctx, src), nameNode.startRow(ctx), kind)
	}

	var addDefinition func(def astNode)
	addDefinition = func(def astNode) {
		switch def.typ(ctx) {
		case "function_definition":
			if nameNode, ok := firstFieldNamed(ctx, def, "name"); ok {
				add(nameNode, "func")
				nameNode.free(ctx)
			}
		case "class_definition":
			if nameNode, ok := firstFieldNamed(ctx, def, "name"); ok {
				add(nameNode, "class")
				nameNode.free(ctx)
			}
		case "decorated_definition":
			if inner, ok := firstFieldNamed(ctx, def, "definition"); ok {
				addDefinition(inner)
				inner.free(ctx)
			}
		}
	}

	// addAssignment reporta los nombres ligados por el lado izquierdo de una asignación,
	// siguiendo asignaciones encadenadas (`a = b = 0` es assignment(left=a, right=assignment(b))).
	var addAssignment func(assign astNode)
	addAssignment = func(assign astNode) {
		if left, ok := firstFieldNamed(ctx, assign, "left"); ok {
			for _, b := range pythonTargetNames(ctx, left, src) {
				addName(b.name, b.row, "var")
			}
			left.free(ctx)
		}
		if right, ok := firstFieldNamed(ctx, assign, "right"); ok {
			if right.typ(ctx) == "assignment" {
				addAssignment(right)
			}
			right.free(ctx)
		}
	}

	err := p.parseTopLevel(ctx, src, func(stmt astNode) {
		switch stmt.typ(ctx) {
		case "function_definition", "class_definition", "decorated_definition":
			addDefinition(stmt)

		case "expression_statement":
			for _, assign := range childrenOfType(ctx, stmt, "assignment") {
				addAssignment(assign)
				assign.free(ctx)
			}

		case "type_alias_statement":
			// left es un nodo `type` cuyo único hijo es el identificador (o un generic_type
			// para `type Alias[T] = ...`, cuyo primer hijo identifier es el nombre).
			if left, ok := firstFieldNamed(ctx, stmt, "left"); ok {
				ids := childrenOfType(ctx, left, "identifier")
				if len(ids) == 0 {
					for _, g := range childrenOfType(ctx, left, "generic_type") {
						ids = append(ids, childrenOfType(ctx, g, "identifier")...)
						g.free(ctx)
					}
				}
				if len(ids) > 0 {
					add(ids[0], "type")
				}
				for _, id := range ids {
					id.free(ctx)
				}
				left.free(ctx)
			}
		}
	})
	if err != nil {
		return nil, err
	}
	return symbols, nil
}

// pythonTargetNames devuelve los nombres ligados por un destino de asignación: el propio nodo si
// es un identifier, o los identificadores dentro de pattern_list / tuple_pattern / list_pattern
// (recursivamente, para `(a, (b, c)) = ...`). Atributos y subíndices no ligan nombres del módulo
// y se omiten. No libera node (es responsabilidad del llamador).
func pythonTargetNames(ctx context.Context, node astNode, src []byte) []boundName {
	switch node.typ(ctx) {
	case "identifier":
		return []boundName{{name: node.text(ctx, src), row: node.startRow(ctx)}}
	case "pattern_list", "tuple_pattern", "list_pattern":
		var out []boundName
		count := node.childCount(ctx)
		for i := uint32(0); i < count; i++ {
			child, err := node.child(ctx, i)
			if err != nil {
				continue
			}
			out = append(out, pythonTargetNames(ctx, child, src)...)
			child.free(ctx)
		}
		return out
	}
	return nil
}
