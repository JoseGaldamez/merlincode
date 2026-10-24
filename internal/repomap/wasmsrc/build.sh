#!/usr/bin/env bash
# build.sh reconstruye go.wasm desde cero. No se ejecuta como parte de `go build`/`go test`:
# go.wasm ya está commiteado (vía go:embed en runtime.go) precisamente para que ningún usuario
# final ni el pipeline de CI necesiten un compilador de C. Este script solo documenta, de forma
# reproducible, cómo se generó ese artefacto, por si hace falta reconstruirlo o agregar un
# lenguaje nuevo.
#
# Arquitectura: en vez de replicar el ABI del binding de navegador `web-tree-sitter` (que en
# versiones modernas usa Emscripten embind con glue JS generado, y cuyas gramáticas .wasm
# publicadas en npm son "side modules" de enlazado dinámico pensados para un motor nativo
# respaldado por wasmtime — ninguna de las dos cosas es viable de reimplementar a mano de forma
# verificable), este build compila y enlaza estáticamente en un solo módulo wasm autocontenido:
#   1. El motor núcleo de tree-sitter (lib/src/*.c), C portable sin dependencias más allá de libc.
#   2. El parser.c generado de UNA gramática (aquí, tree-sitter-go).
#   3. shim.c: un ABI plano hecho a mano (en este mismo directorio) que expone únicamente
#      funciones con parámetros/retornos escalares (i32/i64), porque wazero (el runtime wasm
#      puro-Go usado en tiempo de ejecución) solo puede pasar/recibir escalares a través de la
#      frontera de llamada, nunca structs de C por valor como TSNode.
#
# Agregar otro lenguaje = repetir este mismo proceso con el parser.c de otra gramática y producir
# un .wasm adicional (p. ej. python.wasm); el núcleo y shim.c se reutilizan sin cambios.
#
# Fuentes y versiones exactas usadas para el go.wasm commiteado:
#   - Motor núcleo + tree_sitter/api.h: paquete npm web-tree-sitter@0.25.6 (vendoriza el C
#     portable de lib/src de tree-sitter/tree-sitter en el tag v0.25.6).
#   - Cabeceras internas unicode/*.h y portable/endian.h: tree-sitter/tree-sitter, tag v0.25.6,
#     bajo lib/src/unicode/ y lib/src/portable/.
#   - Gramática: tree-sitter/tree-sitter-go, rama master, commit
#     2346a3ab1bb3857b48b29d779a1ef9799a248cd7 (LANGUAGE_VERSION=15, sin external scanner).
#   - Compilador: Zig 0.17.0 (`zig cc`), usado solo como cross-compilador C -> wasm32-wasi en
#     tiempo de build; el binario final no depende de Zig ni de ningún compilador C.
#
# Uso: ./build.sh <directorio-de-trabajo-vacio>
set -euo pipefail

WORKDIR="${1:?uso: build.sh <directorio-de-trabajo-vacio>}"
TS_TAG="v0.25.6"
GO_GRAMMAR_COMMIT="2346a3ab1bb3857b48b29d779a1ef9799a248cd7"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

mkdir -p "$WORKDIR"/{include/tree_sitter,src/unicode,src/portable,grammar,out}
cd "$WORKDIR"

RAW="https://raw.githubusercontent.com/tree-sitter/tree-sitter/${TS_TAG}"

# Cabecera pública del API de C.
curl -sfL "$RAW/lib/include/tree_sitter/api.h" -o include/tree_sitter/api.h

# Paquete npm web-tree-sitter@0.25.6: trae vendorizado el motor núcleo completo (lib/src/*.c)
# más lib/parser.h (la cabecera interna que #include-a el parser.c generado de cada gramática).
curl -sfL "https://registry.npmjs.org/web-tree-sitter/-/web-tree-sitter-0.25.6.tgz" -o wts.tgz
tar xzf wts.tgz
cp package/lib/parser.h include/tree_sitter/parser.h
cp package/lib/*.c package/lib/*.h src/
rm -f src/wasm_store.c.orig

# Dos cabeceras internas que lib/src/lexer.c y lib/src/*.c necesitan y que el tarball de npm no
# incluye (son del árbol lib/src/{unicode,portable} del repo principal, no del paquete publicado).
for f in ptypes.h umachine.h urename.h utf.h utf16.h utf8.h; do
  curl -sfL "$RAW/lib/src/unicode/$f" -o "src/unicode/$f"
done
curl -sfL "$RAW/lib/src/portable/endian.h" -o src/portable/endian.h

# Gramática de Go, pineada al commit exacto verificado arriba.
curl -sfL "https://raw.githubusercontent.com/tree-sitter/tree-sitter-go/${GO_GRAMMAR_COMMIT}/src/parser.c" \
  -o grammar/parser.c

cp "$SCRIPT_DIR/shim.c" shim.c

zig cc -target wasm32-wasi \
  -Os -Wl,--strip-all \
  -mexec-model=reactor \
  -DTREE_SITTER_LANGUAGE_FN=tree_sitter_go \
  -I include -I src \
  -o out/go.wasm \
  src/*.c grammar/parser.c shim.c

echo "Artefacto generado en: $WORKDIR/out/go.wasm"
echo "Cópialo a internal/repomap/wasmsrc/go.wasm para reemplazar el commiteado."
