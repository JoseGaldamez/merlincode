package repomap

import (
	"context"
	_ "embed"
	"strings"
)

// javascriptWasm es el módulo compilado por wasmsrc/build.sh con la gramática de
// tree-sitter-javascript (que incluye JSX de forma nativa, por eso sirve tanto para .js como
// para .jsx) y su external scanner en C.
//
//go:embed wasmsrc/javascript.wasm
var javascriptWasm []byte

// extractJSTSSymbols parsea src con la gramática real con la que se instanció p (JavaScript,
// TypeScript o TSX: las gramáticas de TypeScript extienden la de JavaScript y comparten sus
// nombres de nodo, así que un mismo extractor sirve para las tres) y devuelve los símbolos
// exportados con la sintaxis de módulos ES (`export`) en el nivel superior del archivo. No es una
// heurística de texto: camina el AST real que produce tree-sitter.
//
// Cubre, verificado con tests sobre las gramáticas reales:
//   - export de declaraciones: function / function* / async function, class, abstract class,
//     const/let/var (incluyendo varios declaradores y patrones de desestructuración), y en
//     TypeScript interface, type, enum, namespace/module, firmas de sobrecarga (deduplicadas por
//     nombre) y `export declare ...`.
//   - export default: con nombre (`export default function App`) se reporta el nombre con su
//     tipo; anónimo (`export default class {}`, `export default 42`) se reporta como "default"
//     con kind "default".
//   - listas de export y re-exports: `export { a, b as c }`, `export { x } from "./m"`,
//     `export * as ns from "./m"` (kind "export" o "reexport"); `export * from "./m"` se reporta
//     con el origen como nombre porque no introduce ningún nombre local.
//
// Fuera de alcance (no se reporta nada para ellos): CommonJS (`module.exports = ...`,
// `exports.x = ...`), que no es sintaxis de exportación sino asignaciones en tiempo de ejecución;
// `export = x` e `export import A = B` de TypeScript; y los exports anidados dentro de cuerpos de
// `namespace` o de `declare module "x" { ... }` (solo se reporta el namespace en sí).
func extractJSTSSymbols(ctx context.Context, p *parser, src []byte) ([]Symbol, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	var symbols []Symbol
	seen := map[string]bool{}

	add := func(name, kind string, line uint32) {
		if name == "" {
			return
		}
		// Las sobrecargas de TypeScript (`export function f(a: string): void;` repetido antes de
		// la implementación) producirían el mismo símbolo varias veces.
		key := kind + "\x00" + name
		if seen[key] {
			return
		}
		seen[key] = true
		symbols = append(symbols, Symbol{Name: name, Kind: kind, Line: int(line) + 1})
	}

	addNameField := func(node astNode, kind string) {
		if nameNode, ok := firstFieldNamed(ctx, node, "name"); ok {
			add(nameNode.text(ctx, src), kind, nameNode.startRow(ctx))
			nameNode.free(ctx)
		}
	}

	// addDeclaration maneja el nodo del campo "declaration" de un export_statement. Devuelve
	// false si el tipo de nodo no es una declaración conocida.
	var addDeclaration func(decl astNode) bool
	addDeclaration = func(decl astNode) bool {
		switch decl.typ(ctx) {
		case "function_declaration", "generator_function_declaration", "function_signature":
			addNameField(decl, "func")
		case "class_declaration", "abstract_class_declaration":
			addNameField(decl, "class")
		case "interface_declaration":
			addNameField(decl, "interface")
		case "type_alias_declaration":
			addNameField(decl, "type")
		case "enum_declaration":
			addNameField(decl, "enum")
		case "internal_module", "module":
			// `namespace X {}` / `module X {}`. Un nombre de tipo string solo aparece en
			// `declare module "x"`, que no es un export_statement.
			addNameField(decl, "namespace")
		case "lexical_declaration", "variable_declaration":
			kind := "var"
			if kindNode, ok := firstFieldNamed(ctx, decl, "kind"); ok {
				kind = kindNode.text(ctx, src) // "const" o "let"
				kindNode.free(ctx)
			}
			for _, declarator := range childrenOfType(ctx, decl, "variable_declarator") {
				if nameNode, ok := firstFieldNamed(ctx, declarator, "name"); ok {
					for _, b := range collectBindingIdentifiers(ctx, nameNode, src) {
						add(b.name, kind, b.row)
					}
					nameNode.free(ctx)
				}
				declarator.free(ctx)
			}
		case "ambient_declaration":
			// `export declare function f(): void;`: la declaración real es el primer hijo con
			// nombre después de la palabra clave `declare`.
			count := decl.childCount(ctx)
			for i := uint32(0); i < count; i++ {
				child, err := decl.child(ctx, i)
				if err != nil {
					continue
				}
				handled := addDeclaration(child)
				child.free(ctx)
				if handled {
					break
				}
			}
		default:
			return false
		}
		return true
	}

	err := p.parseTopLevel(ctx, src, func(stmt astNode) {
		if stmt.typ(ctx) != "export_statement" {
			return
		}
		line := stmt.startRow(ctx)

		if decl, ok := firstFieldNamed(ctx, stmt, "declaration"); ok {
			addDeclaration(decl)
			decl.free(ctx)
			return
		}

		// `export default <expresión>` sin declaración con nombre (clase/función anónima,
		// literal, identificador).
		if value, ok := firstFieldNamed(ctx, stmt, "value"); ok {
			name := "default"
			if value.typ(ctx) == "identifier" {
				name = value.text(ctx, src)
			}
			value.free(ctx)
			add(name, "default", line)
			return
		}

		source, hasSource := firstFieldNamed(ctx, stmt, "source")
		sourceText := ""
		if hasSource {
			sourceText = source.text(ctx, src)
			source.free(ctx)
		}
		exportKind := "export"
		if hasSource {
			exportKind = "reexport"
		}

		handledClause := false
		count := stmt.childCount(ctx)
		for i := uint32(0); i < count; i++ {
			child, err := stmt.child(ctx, i)
			if err != nil {
				continue
			}
			switch child.typ(ctx) {
			case "export_clause":
				handledClause = true
				for _, spec := range childrenOfType(ctx, child, "export_specifier") {
					// El nombre visible desde fuera es el alias si existe (`a as b` -> b).
					field := "name"
					if alias, ok := firstFieldNamed(ctx, spec, "alias"); ok {
						alias.free(ctx)
						field = "alias"
					}
					if n, ok := firstFieldNamed(ctx, spec, field); ok {
						add(trimJSStringQuotes(n.text(ctx, src)), exportKind, n.startRow(ctx))
						n.free(ctx)
					}
					spec.free(ctx)
				}
			case "namespace_export":
				handledClause = true
				// `export * as ns from "./m"`: el identificador es el último hijo con nombre.
				for _, id := range childrenOfType(ctx, child, "identifier") {
					add(id.text(ctx, src), exportKind, id.startRow(ctx))
					id.free(ctx)
				}
			}
			child.free(ctx)
		}

		if !handledClause && hasSource {
			// `export * from "./m"`: no introduce un nombre propio; se reporta el origen.
			add("* from "+sourceText, exportKind, line)
		}
	})
	if err != nil {
		return nil, err
	}
	return symbols, nil
}

// boundName es un nombre introducido por una declaración, con su fila 0-based en el fuente.
type boundName struct {
	name string
	row  uint32
}

// collectBindingIdentifiers devuelve los nombres que un patrón de declaración introduce
// realmente: el propio nodo si es un identifier, o, para desestructuración, los identificadores
// ligados dentro de object_pattern/array_pattern (incluyendo valores por defecto y ...rest),
// recorriendo el AST real. Las claves de propiedad (`{ b: renamed }` -> solo renamed) no
// introducen nombres y se omiten. No libera node (es responsabilidad del llamador).
func collectBindingIdentifiers(ctx context.Context, node astNode, src []byte) []boundName {
	switch node.typ(ctx) {
	case "identifier", "shorthand_property_identifier_pattern":
		return []boundName{{name: node.text(ctx, src), row: node.startRow(ctx)}}
	case "pair_pattern":
		if value, ok := firstFieldNamed(ctx, node, "value"); ok {
			out := collectBindingIdentifiers(ctx, value, src)
			value.free(ctx)
			return out
		}
		return nil
	case "assignment_pattern", "object_assignment_pattern":
		if left, ok := firstFieldNamed(ctx, node, "left"); ok {
			out := collectBindingIdentifiers(ctx, left, src)
			left.free(ctx)
			return out
		}
		return nil
	case "object_pattern", "array_pattern", "rest_pattern":
		var out []boundName
		count := node.childCount(ctx)
		for i := uint32(0); i < count; i++ {
			child, err := node.child(ctx, i)
			if err != nil {
				continue
			}
			out = append(out, collectBindingIdentifiers(ctx, child, src)...)
			child.free(ctx)
		}
		return out
	}
	return nil
}

// trimJSStringQuotes quita las comillas de un nombre de export en forma de string
// (`export { x as "nombre con espacios" }`, permitido desde ES2022).
func trimJSStringQuotes(s string) string {
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1]
	}
	return strings.TrimSpace(s)
}
