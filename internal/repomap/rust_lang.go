package repomap

import (
	"context"
	_ "embed"
	"strings"
)

// rustWasm es el módulo compilado por wasmsrc/build.sh con la gramática de tree-sitter-rust y su
// external scanner en C (strings raw, comentarios de bloque anidados, etc.).
//
//go:embed wasmsrc/rust.wasm
var rustWasm []byte

// rustItemKinds mapea los tipos de nodo de item de tree-sitter-rust que se reportan al kind que
// se muestra en el mapa.
var rustItemKinds = map[string]string{
	"function_item": "fn",
	"struct_item":   "struct",
	"enum_item":     "enum",
	"union_item":    "union",
	"trait_item":    "trait",
	"const_item":    "const",
	"static_item":   "static",
	"type_item":     "type",
	"mod_item":      "mod",
}

// rustIsPub indica si item lleva visibilidad `pub` sin restricción. `pub(crate)`, `pub(super)`,
// `pub(self)` y `pub(in ruta)` se consideran NO públicos para el mapa: son visibles solo dentro
// del crate (o menos), no parte de su API pública. Se decide sobre el nodo visibility_modifier
// real del AST, no sobre el texto de la línea.
func rustIsPub(ctx context.Context, item astNode, src []byte) bool {
	mods := childrenOfType(ctx, item, "visibility_modifier")
	defer func() {
		for _, m := range mods {
			m.free(ctx)
		}
	}()
	return len(mods) > 0 && strings.TrimSpace(mods[0].text(ctx, src)) == "pub"
}

// rustImplTypeName devuelve el nombre del tipo de un bloque impl sin argumentos genéricos
// (`impl<T> Stack<T>` -> "Stack"), para mostrar sus métodos como "Stack.push".
func rustImplTypeName(raw string) string {
	if i := strings.IndexByte(raw, '<'); i >= 0 {
		raw = raw[:i]
	}
	return strings.TrimSpace(raw)
}

// extractRustSymbols parsea src con la gramática real de Rust (vía wazero + rust.wasm) y devuelve
// los items con visibilidad `pub` (sin restricción, ver rustIsPub) caminando el AST real que
// produce tree-sitter-rust:
//   - items de nivel superior: fn, struct, enum, union, trait, const, static, type y mod.
//   - dentro de un `pub mod nombre { ... }` en línea, sus items `pub`, recursivamente, con el
//     nombre calificado por la ruta de módulos ("nombre::item"). No se desciende en módulos no
//     `pub` (sus items no son alcanzables desde fuera salvo re-export) ni, obviamente, en
//     `mod x;` declarado en otro archivo (ese archivo se mapea por separado si está en el
//     proyecto).
//   - métodos `pub` de bloques impl INHERENTES (`impl Tipo { pub fn m() }`), como "Tipo.m"
//     (kind "method"), tanto a nivel superior como dentro de los pub mod recorridos.
//
// Fuera de alcance (no se reporta nada para ellos): métodos de `impl Trait for Tipo` (no llevan
// `pub` propio: su visibilidad es la del trait, y reportarlos inflaría el mapa con cada impl de
// Display/From/etc.), re-exports `pub use`, macros (`macro_rules!` con `#[macro_export]`), items
// de bloques `extern "C" { ... }`, items generados por macros, y items `pub(crate)`/`pub(super)`/
// `pub(in ...)`.
func extractRustSymbols(ctx context.Context, p *parser, src []byte) ([]Symbol, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	var symbols []Symbol

	var visitItem func(item astNode, modPath string)

	// visitDeclarationList recorre el cuerpo (declaration_list) de un mod o impl.
	visitDeclarationList := func(body astNode, fn func(child astNode)) {
		count := body.childCount(ctx)
		for i := uint32(0); i < count; i++ {
			child, err := body.child(ctx, i)
			if err != nil {
				continue
			}
			fn(child)
			child.free(ctx)
		}
	}

	visitItem = func(item astNode, modPath string) {
		typ := item.typ(ctx)

		if typ == "impl_item" {
			// Solo impl inherentes: un impl de trait tiene el campo "trait".
			if traitNode, ok := firstFieldNamed(ctx, item, "trait"); ok {
				traitNode.free(ctx)
				return
			}
			typeNode, ok := firstFieldNamed(ctx, item, "type")
			if !ok {
				return
			}
			typeName := rustImplTypeName(typeNode.text(ctx, src))
			typeNode.free(ctx)
			body, ok := firstFieldNamed(ctx, item, "body")
			if !ok {
				return
			}
			visitDeclarationList(body, func(member astNode) {
				if member.typ(ctx) != "function_item" || !rustIsPub(ctx, member, src) {
					return
				}
				if nameNode, ok := firstFieldNamed(ctx, member, "name"); ok {
					symbols = append(symbols, Symbol{
						Name: modPath + typeName + "." + nameNode.text(ctx, src),
						Kind: "method",
						Line: int(nameNode.startRow(ctx)) + 1,
					})
					nameNode.free(ctx)
				}
			})
			body.free(ctx)
			return
		}

		kind, ok := rustItemKinds[typ]
		if !ok || !rustIsPub(ctx, item, src) {
			return
		}
		nameNode, ok := firstFieldNamed(ctx, item, "name")
		if !ok {
			return
		}
		name := nameNode.text(ctx, src)
		symbols = append(symbols, Symbol{Name: modPath + name, Kind: kind, Line: int(nameNode.startRow(ctx)) + 1})
		nameNode.free(ctx)

		if typ == "mod_item" {
			// `pub mod x { ... }` en línea; `pub mod x;` no tiene campo body.
			if body, ok := firstFieldNamed(ctx, item, "body"); ok {
				visitDeclarationList(body, func(child astNode) {
					visitItem(child, modPath+name+"::")
				})
				body.free(ctx)
			}
		}
	}

	err := p.parseTopLevel(ctx, src, func(item astNode) {
		visitItem(item, "")
	})
	if err != nil {
		return nil, err
	}
	return symbols, nil
}
