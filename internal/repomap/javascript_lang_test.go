package repomap

import "testing"

const sampleJSXSource = `import React from "react";

export function greet(name) { return "hi " + name; }
function internal() {}
export class Widget extends Base {}
export const MAX = 3, MIN = 1;
export let counter = 0;
export var legacy = 1;
export default function App() { return <div className="x">{greet("a")}</div>; }
export async function* stream() {}
export { internal as publicInternal };
export * as ns from "./mod2";
export { helper } from "./helpers";
export const { a, b: renamed, ...rest } = obj;
export const [first, , second = 2] = list;
export const Card = () => <Widget title="t" />;
const notExported = 1;
class AlsoHidden {}
`

// TestExtractJSSymbolsParseaASTReal instancia javascript.wasm (motor núcleo de tree-sitter +
// tree-sitter-javascript + su external scanner) vía wazero, parsea JavaScript con JSX real y
// verifica exactamente los símbolos exportados que produce caminando el AST, sin mocks.
func TestExtractJSSymbolsParseaASTReal(t *testing.T) {
	symbols := extractWithRealWasm(t, javascriptWasm, extractJSTSSymbols, sampleJSXSource)

	assertExactSymbols(t, symbols, map[string]string{
		"greet":          "func",
		"Widget":         "class",
		"MAX":            "const",
		"MIN":            "const",
		"counter":        "let",
		"legacy":         "var",
		"App":            "func",
		"stream":         "func",
		"publicInternal": "export",
		"ns":             "reexport",
		"helper":         "reexport",
		"a":              "const",
		"renamed":        "const",
		"rest":           "const",
		"first":          "const",
		"second":         "const",
		"Card":           "const",
	})

	// greet está en la línea 3 (1-based) de sampleJSXSource; App en la 9, después de que el
	// scanner y el parser procesen JSX en la misma línea.
	if got := symbolLine(symbols, "greet"); got != 3 {
		t.Errorf("esperaba greet en la línea 3, obtenido %d", got)
	}
	if got := symbolLine(symbols, "Card"); got != 16 {
		t.Errorf("esperaba Card en la línea 16, obtenido %d", got)
	}
}

// TestExtractJSSymbolsExportDefaultAnonimoYStar cubre los exports que no introducen un nombre
// local declarado: default anónimo, default de un identificador y `export * from`.
func TestExtractJSSymbolsExportDefaultAnonimoYStar(t *testing.T) {
	anon := extractWithRealWasm(t, javascriptWasm, extractJSTSSymbols, "export default class {}\nexport * from \"./all\";\n")
	assertExactSymbols(t, anon, map[string]string{
		"default":        "default",
		`* from "./all"`: "reexport",
	})

	named := extractWithRealWasm(t, javascriptWasm, extractJSTSSymbols, "const App = 1;\nexport default App;\n")
	assertExactSymbols(t, named, map[string]string{"App": "default"})
}
