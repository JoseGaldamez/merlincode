#!/usr/bin/env bash
# build.sh reconstruye los módulos <lenguaje>.wasm desde cero. No se ejecuta como parte de
# `go build`/`go test`: los .wasm ya están commiteados (vía go:embed en los <lenguaje>_lang.go)
# precisamente para que ningún usuario final ni el pipeline de CI necesiten un compilador de C.
# Este script solo documenta, de forma reproducible, cómo se generaron esos artefactos, por si
# hace falta reconstruirlos o agregar un lenguaje nuevo.
#
# Arquitectura: en vez de replicar el ABI del binding de navegador `web-tree-sitter` (que en
# versiones modernas usa Emscripten embind con glue JS generado, y cuyas gramáticas .wasm
# publicadas en npm son "side modules" de enlazado dinámico pensados para un motor nativo
# respaldado por wasmtime — ninguna de las dos cosas es viable de reimplementar a mano de forma
# verificable), este build compila y enlaza estáticamente, por cada lenguaje, un solo módulo wasm
# autocontenido:
#   1. El motor núcleo de tree-sitter (lib/src/*.c), C portable sin dependencias más allá de libc.
#   2. El parser.c generado de UNA gramática, más su external scanner (scanner.c) si la gramática
#      lo declara (todas excepto Go lo necesitan; ver la tabla más abajo).
#   3. shim.c: un ABI plano hecho a mano (en este mismo directorio) que expone únicamente
#      funciones con parámetros/retornos escalares (i32/i64), porque wazero (el runtime wasm
#      puro-Go usado en tiempo de ejecución) solo puede pasar/recibir escalares a través de la
#      frontera de llamada, nunca structs de C por valor como TSNode. shim.c es agnóstico al
#      lenguaje: solo cambia el símbolo constructor que se le inyecta con
#      -DTREE_SITTER_LANGUAGE_FN=tree_sitter_<lenguaje>.
#
# Fuentes y versiones exactas usadas para los .wasm commiteados:
#   - Motor núcleo + tree_sitter/api.h: paquete npm web-tree-sitter@0.25.6 (vendoriza el C
#     portable de lib/src de tree-sitter/tree-sitter en el tag v0.25.6). El paquete también trae
#     lib/lib.c (amalgamación que #include-a todos los demás .c, y que duplicaría símbolos si se
#     compila junto a ellos) y lib/tree-sitter.c (glue de Emscripten, requiere emscripten.h):
#     ambos se excluyen explícitamente. Nota de honestidad: la versión anterior de este script
#     no los excluía y por tanto NO compilaba tal cual; se verificó que excluyéndolos se
#     reproduce byte a byte el go.wasm commiteado originalmente
#     (sha256 7f3e4f199fae899fcb07949d6359384f5634c767885b3f387a61816024b3e3c5).
#   - Cabeceras internas unicode/*.h y portable/endian.h: tree-sitter/tree-sitter, tag v0.25.6,
#     bajo lib/src/unicode/ y lib/src/portable/.
#   - Gramáticas (commit exacto pineado; LANGUAGE_VERSION leído del parser.c generado, el núcleo
#     v0.25.6 acepta ABI 13..15):
#       go          tree-sitter/tree-sitter-go          2346a3ab1bb3857b48b29d779a1ef9799a248cd7
#                   (rama master; ABI 15; sin external scanner)
#       javascript  tree-sitter/tree-sitter-javascript  44c892e0be055ac465d5eeddae6d3e194424e7de
#                   (tag v0.25.0; ABI 15; external scanner en C: src/scanner.c). Cubre .js y .jsx:
#                   la gramática de JavaScript incluye JSX de forma nativa.
#       typescript  tree-sitter/tree-sitter-typescript  f975a621f4e7f532fe322e13c4f79495e0a7b2e7
#                   (tag v0.23.2; ABI 14; subdirectorio typescript/; external scanner en C:
#                   typescript/src/scanner.c, que #include-a ../../common/scanner.h)
#       tsx         mismo repo y commit que typescript, pero subdirectorio tsx/: es una gramática
#                   SEPARADA con su propio parser.c generado (TS + JSX) y su propio constructor
#                   tree_sitter_tsx; la gramática typescript/ no parsea JSX y viceversa la de tsx/
#                   no acepta los type assertions `<T>expr`, por eso son dos .wasm distintos.
#       python      tree-sitter/tree-sitter-python      293fdc02038ee2bf0e2e206711b69c90ac0d413f
#                   (tag v0.25.0; ABI 15; external scanner en C: src/scanner.c, indentación)
#       rust        tree-sitter/tree-sitter-rust        77a3747266f4d621d0757825e6b11edcbf991ca5
#                   (tag v0.24.2; ABI 15; external scanner en C: src/scanner.c)
#     Ninguna de estas gramáticas usa un scanner en C++ (scanner.cc) en los commits pineados.
#     Para las gramáticas con scanner se descargan también sus cabeceras src/tree_sitter/
#     (parser.h, alloc.h, array.h), y la gramática se compila en una unidad aparte con
#     -I <gramática>/src para que parser.c, scanner.c y common/scanner.h resuelvan las cabeceras
#     contra las que fueron generados. Go no descarga sus cabeceras y usa el parser.h de
#     web-tree-sitter, exactamente como el go.wasm original (así se reproduce byte a byte).
#   - Compilador: Zig 0.17.0 (`zig cc`, zig-x86_64-windows-0.17.0.zip de ziglang.org, sha256
#     b5663f69581dcf391293fbf16c06cb80d81d806545ce618b4d0bab7f0eb8c428), usado solo como
#     cross-compilador C -> wasm32-wasi en tiempo de build; los binarios finales no dependen de
#     Zig ni de ningún compilador C.
#
# Uso: ./build.sh <directorio-de-trabajo-vacio> [lenguaje...]
#   lenguaje ∈ go javascript typescript tsx python rust (por defecto, todos).
#   Cada artefacto queda en <directorio>/out/<lenguaje>.wasm.
set -euo pipefail

WORKDIR="${1:?uso: build.sh <directorio-de-trabajo-vacio> [lenguaje...]}"
shift
if [ "$#" -eq 0 ]; then
  set -- go javascript typescript tsx python rust
fi

TS_TAG="v0.25.6"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RAW="https://raw.githubusercontent.com/tree-sitter/tree-sitter/${TS_TAG}"
ZIG_FLAGS=(-target wasm32-wasi -Os)

mkdir -p "$WORKDIR"/{include/tree_sitter,src/unicode,src/portable,out}
cd "$WORKDIR"

# --- Motor núcleo (compartido por todos los lenguajes) ---------------------------------------

# Cabecera pública del API de C.
curl -sfL "$RAW/lib/include/tree_sitter/api.h" -o include/tree_sitter/api.h

# Paquete npm web-tree-sitter@0.25.6: trae vendorizado el motor núcleo completo (lib/src/*.c)
# más lib/parser.h (la cabecera interna que #include-a el parser.c generado de cada gramática).
curl -sfL "https://registry.npmjs.org/web-tree-sitter/-/web-tree-sitter-0.25.6.tgz" -o wts.tgz
tar xzf wts.tgz
cp package/lib/parser.h include/tree_sitter/parser.h
cp package/lib/*.c package/lib/*.h src/
# lib.c es una amalgamación de los demás .c (símbolos duplicados si se compila junto a ellos) y
# tree-sitter.c es el glue de Emscripten del binding JS (requiere emscripten.h): fuera.
rm -f src/lib.c src/tree-sitter.c src/wasm_store.c.orig

# Dos cabeceras internas que lib/src/lexer.c y lib/src/*.c necesitan y que el tarball de npm no
# incluye (son del árbol lib/src/{unicode,portable} del repo principal, no del paquete publicado).
for f in ptypes.h umachine.h urename.h utf.h utf16.h utf8.h; do
  curl -sfL "$RAW/lib/src/unicode/$f" -o "src/unicode/$f"
done
curl -sfL "$RAW/lib/src/portable/endian.h" -o src/portable/endian.h

cp "$SCRIPT_DIR/shim.c" shim.c

# --- Gramáticas ------------------------------------------------------------------------------

# build_lang <nombre> <repo> <commit> <subdir-o-""> <tree_sitter_fn> <con-scanner:0|1>
build_lang() {
  local name="$1" repo="$2" commit="$3" subdir="$4" fn="$5" scanner="$6"
  local base="https://raw.githubusercontent.com/tree-sitter/${repo}/${commit}"
  local prefix=""
  [ -n "$subdir" ] && prefix="${subdir}/"
  local gdir="grammar-${name}"
  local gsrc="${gdir}/${prefix}src"

  rm -rf "$gdir"
  mkdir -p "$gsrc/tree_sitter"
  curl -sfL "$base/${prefix}src/parser.c" -o "$gsrc/parser.c"

  local objs=()
  if [ "$scanner" = "1" ]; then
    # Si parser.c no declara el external scanner, esta tabla está mal: abortar en vez de
    # enlazar un scanner sobrante (o, peor, omitir uno requerido).
    grep -q "${fn}_external_scanner_scan" "$gsrc/parser.c" || {
      echo "ERROR: $name: parser.c no referencia ${fn}_external_scanner_scan" >&2
      exit 1
    }
    curl -sfL "$base/${prefix}src/scanner.c" -o "$gsrc/scanner.c"
    for h in parser.h alloc.h array.h; do
      curl -sfL "$base/${prefix}src/tree_sitter/$h" -o "$gsrc/tree_sitter/$h"
    done
    if [ "$repo" = "tree-sitter-typescript" ]; then
      mkdir -p "$gdir/common"
      curl -sfL "$base/common/scanner.h" -o "$gdir/common/scanner.h"
    fi
    zig cc "${ZIG_FLAGS[@]}" -I "$gsrc" -c "$gsrc/parser.c" -o "out/${name}-parser.o"
    zig cc "${ZIG_FLAGS[@]}" -I "$gsrc" -c "$gsrc/scanner.c" -o "out/${name}-scanner.o"
    objs=("out/${name}-parser.o" "out/${name}-scanner.o")
  else
    if grep -q "${fn}_external_scanner_scan" "$gsrc/parser.c"; then
      echo "ERROR: $name: parser.c requiere external scanner pero la tabla dice que no" >&2
      exit 1
    fi
    objs=("$gsrc/parser.c")
  fi

  zig cc "${ZIG_FLAGS[@]}" \
    -Wl,--strip-all \
    -mexec-model=reactor \
    -DTREE_SITTER_LANGUAGE_FN="$fn" \
    -I include -I src \
    -o "out/${name}.wasm" \
    src/*.c "${objs[@]}" shim.c

  echo "Artefacto generado en: $WORKDIR/out/${name}.wasm"
}

for lang in "$@"; do
  case "$lang" in
    go)
      build_lang go tree-sitter-go 2346a3ab1bb3857b48b29d779a1ef9799a248cd7 "" tree_sitter_go 0 ;;
    javascript)
      build_lang javascript tree-sitter-javascript 44c892e0be055ac465d5eeddae6d3e194424e7de "" tree_sitter_javascript 1 ;;
    typescript)
      build_lang typescript tree-sitter-typescript f975a621f4e7f532fe322e13c4f79495e0a7b2e7 typescript tree_sitter_typescript 1 ;;
    tsx)
      build_lang tsx tree-sitter-typescript f975a621f4e7f532fe322e13c4f79495e0a7b2e7 tsx tree_sitter_tsx 1 ;;
    python)
      build_lang python tree-sitter-python 293fdc02038ee2bf0e2e206711b69c90ac0d413f "" tree_sitter_python 1 ;;
    rust)
      build_lang rust tree-sitter-rust 77a3747266f4d621d0757825e6b11edcbf991ca5 "" tree_sitter_rust 1 ;;
    *)
      echo "lenguaje desconocido: $lang" >&2
      exit 1 ;;
  esac
done

echo "Copia los out/<lenguaje>.wasm a internal/repomap/wasmsrc/ para reemplazar los commiteados."
