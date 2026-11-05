package repomap

import "testing"

const sampleTSSource = `import { Base } from "./base";

export interface Props { name: string }
export type ID = string | number;
export enum Color { Red, Green }
export const enum Direction { Up }
export abstract class Shape<T> implements Drawable { abstract area(): number; }
export function overloaded(a: string): string;
export function overloaded(a: number): number;
export function overloaded(a: any): any { return a; }
export declare function ambient(x: number): void;
export declare const VERSION: string;
export namespace Util { export const inner = 1; }
export type { Foo } from "./foo";
export default interface Settings {}
interface Hidden {}
type AlsoHidden = number;
const cast = <any>value;
`

// TestExtractTSSymbolsParseaASTReal instancia typescript.wasm (gramática typescript/ de
// tree-sitter-typescript + su external scanner) vía wazero y verifica exactamente los símbolos
// exportados de código TypeScript real, incluyendo un type assertion con ángulos (`<any>value`),
// que solo la gramática typescript (no la tsx) acepta.
func TestExtractTSSymbolsParseaASTReal(t *testing.T) {
	symbols := extractWithRealWasm(t, typescriptWasm, extractJSTSSymbols, sampleTSSource)

	assertExactSymbols(t, symbols, map[string]string{
		"Props":      "interface",
		"ID":         "type",
		"Color":      "enum",
		"Direction":  "enum",
		"Shape":      "class",
		"overloaded": "func", // tres firmas, un solo símbolo
		"ambient":    "func",
		"VERSION":    "const",
		"Util":       "namespace",
		"Foo":        "reexport",
		"Settings":   "interface",
	})

	if got := symbolLine(symbols, "overloaded"); got != 8 {
		t.Errorf("esperaba overloaded en la línea 8 (primera firma), obtenido %d", got)
	}
}

const sampleTSXSource = `import React from "react";

export interface ButtonProps { label: string }
export const Button = ({ label }: ButtonProps) => <button>{label}</button>;
export default function Page(): JSX.Element { return <Button label="x" />; }
function internalHelper() {}
`

// TestExtractTSXSymbolsParseaASTReal instancia tsx.wasm (gramática tsx/, separada de la
// typescript/) y verifica los exports de un componente React con anotaciones de tipo y JSX.
func TestExtractTSXSymbolsParseaASTReal(t *testing.T) {
	symbols := extractWithRealWasm(t, tsxWasm, extractJSTSSymbols, sampleTSXSource)

	assertExactSymbols(t, symbols, map[string]string{
		"ButtonProps": "interface",
		"Button":      "const",
		"Page":        "func",
	})
	if got := symbolLine(symbols, "Page"); got != 5 {
		t.Errorf("esperaba Page en la línea 5, obtenido %d", got)
	}
}

// TestGramaticaTypeScriptNoSirveParaTSX documenta con el parser real por qué .tsx necesita su
// propio módulo: la gramática typescript/ interpreta `<button>` como un type assertion y se
// traga el resto del archivo, perdiendo el `export default function Page` posterior. Si algún
// día este test falla porque typescript.wasm sí extrae Page, la separación seguiría siendo
// correcta pero el comentario de typescript_lang.go debería revisarse.
func TestGramaticaTypeScriptNoSirveParaTSX(t *testing.T) {
	symbols := extractWithRealWasm(t, typescriptWasm, extractJSTSSymbols, sampleTSXSource)
	if symbolLine(symbols, "Page") != 0 {
		t.Fatalf("se esperaba que la gramática typescript (sin JSX) no reconociera Page tras el JSX; obtenido %+v", symbols)
	}
}

// TestSupportedLanguagesUsaLaGramaticaCorrectaPorExtension verifica que la tabla de
// service.go asocia cada extensión con el módulo wasm correcto (comparando identidad del slice
// embebido, no solo longitud), en particular .ts -> typescript y .tsx -> tsx.
func TestSupportedLanguagesUsaLaGramaticaCorrectaPorExtension(t *testing.T) {
	want := map[string][]byte{
		".go":  goWasm,
		".js":  javascriptWasm,
		".jsx": javascriptWasm,
		".ts":  typescriptWasm,
		".tsx": tsxWasm,
		".py":  pythonWasm,
		".rs":  rustWasm,
	}
	if len(supportedLanguages) != len(want) {
		t.Fatalf("esperaba %d lenguajes registrados, hay %d", len(want), len(supportedLanguages))
	}
	for _, lang := range supportedLanguages {
		expected, ok := want[lang.ext]
		if !ok {
			t.Fatalf("extensión inesperada registrada: %s", lang.ext)
		}
		if len(lang.wasm) == 0 || len(expected) == 0 || &lang.wasm[0] != &expected[0] {
			t.Fatalf("la extensión %s no apunta al módulo wasm esperado", lang.ext)
		}
	}
}
