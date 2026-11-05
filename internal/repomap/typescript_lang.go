package repomap

import (
	_ "embed"
)

// typescriptWasm y tsxWasm son dos módulos distintos compilados por wasmsrc/build.sh a partir del
// MISMO repo tree-sitter-typescript, que en realidad publica dos gramáticas generadas por
// separado (typescript/src/parser.c y tsx/src/parser.c, cada una con su external scanner que
// comparte common/scanner.h). No son intercambiables, verificado parseando con ambas:
//   - la gramática typescript no conoce JSX: `<button>{x}</button>` lo parsea como un type
//     assertion `<button>` seguido de una expresión, tragándose las declaraciones siguientes
//     (un `export default function Page` después del JSX se perdería).
//   - la gramática tsx no acepta type assertions con ángulos (`<any>value`, que en .tsx es
//     ilegal también para el compilador de TypeScript) y produce un nodo ERROR.
//
// Por eso .ts usa typescriptWasm y .tsx usa tsxWasm. Ambas comparten nombres de nodo con
// JavaScript, así que el extractor es extractJSTSSymbols (javascript_lang.go).
//
//go:embed wasmsrc/typescript.wasm
var typescriptWasm []byte

//go:embed wasmsrc/tsx.wasm
var tsxWasm []byte
