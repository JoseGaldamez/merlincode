// Package repomap extrae símbolos exportados/públicos de los archivos del proyecto activo usando
// AST reales de tree-sitter, no heurísticas de texto. tree-sitter corre compilado a WebAssembly y
// se ejecuta con wazero (github.com/tetratelabs/wazero), un runtime wasm 100% Go — cero CGO, igual
// que el resto del proyecto (ver AGENTS.md: este proyecto ya rechazó smacker/go-tree-sitter por
// depender de CGO).
//
// El módulo .wasm embebido (wasmsrc/go.wasm) NO replica el ABI del binding de navegador
// `web-tree-sitter`: ese binding usa, en versiones modernas, Emscripten embind (clases C++ con
// glue JS generado) para la interfaz principal, y sus gramáticas .wasm publicadas en npm son
// "side modules" de enlazado dinámico pensados para cargarse en un motor nativo respaldado por
// wasmtime — ninguna de las dos cosas es practicable de reimplementar a mano sin un oráculo de
// referencia. En vez de eso, go.wasm enlaza estáticamente en un solo módulo autocontenido: el
// motor núcleo portable de tree-sitter + la gramática generada de un lenguaje + un ABI plano
// hecho a mano (wasmsrc/shim.c) que solo pasa/retorna escalares (i32), porque wazero solo puede
// cruzar la frontera de llamada con escalares, nunca structs de C por valor como TSNode. Ver
// wasmsrc/build.sh para las fuentes exactas y versiones pineadas usadas para compilarlo.
//
// Agregar un lenguaje nuevo significa: compilar otro .wasm con el mismo shim pero la gramática de
// ese lenguaje (wasmsrc/build.sh es reutilizable con otro parser.c), embeberlo aquí, y añadir un
// extractor (ver go_lang.go) que conozca la forma de su AST. El bridge de esta página es agnóstico
// al lenguaje.
package repomap

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// cStringMaxLen acota cuántos bytes se leen al buscar el NUL final de una cadena C devuelta por el
// wasm (nombres de tipo de nodo o de campo de la gramática; siempre identificadores cortos).
const cStringMaxLen = 256

// parser envuelve una instancia wazero de go.wasm (el motor núcleo de tree-sitter + exactamente
// una gramática, enlazados estáticamente) y expone el ABI plano de escalares de wasmsrc/shim.c.
// Una sola instancia de módulo wasm no es segura para llamadas concurrentes (comparte una única
// memoria lineal): todo acceso debe serializarse con mu, incluyendo la secuencia completa de
// llamadas de un parseo (alloc -> parse -> caminar el árbol -> liberar), no solo cada llamada
// individual.
type parser struct {
	mu       sync.Mutex
	mod      api.Module
	tsParser uint32 // handle TSParser* (vía tsm_new_parser), creado y reutilizado de forma perezosa
}

// newRuntime crea un wazero.Runtime con soporte WASI, requerido porque go.wasm se compiló para el
// target wasm32-wasi (aunque este módulo en particular no realiza E/S real: es una "reactor"
// library, no un comando con _start).
func newRuntime(ctx context.Context) (wazero.Runtime, error) {
	rt := wazero.NewRuntime(ctx)
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, rt); err != nil {
		rt.Close(ctx)
		return nil, fmt.Errorf("repomap: no se pudo instanciar WASI: %w", err)
	}
	return rt, nil
}

// newParser instancia wasmBytes (un módulo compilado por wasmsrc/build.sh) dentro de rt.
func newParser(ctx context.Context, rt wazero.Runtime, wasmBytes []byte) (*parser, error) {
	mod, err := rt.Instantiate(ctx, wasmBytes)
	if err != nil {
		return nil, fmt.Errorf("repomap: no se pudo instanciar el módulo wasm: %w", err)
	}
	return &parser{mod: mod}, nil
}

// call invoca una función exportada del módulo por nombre con argumentos escalares, devolviendo su
// único valor de retorno (0 si no retorna nada). El llamador debe tener mu adquirido.
func (p *parser) call(ctx context.Context, name string, args ...uint64) (uint64, error) {
	fn := p.mod.ExportedFunction(name)
	if fn == nil {
		return 0, fmt.Errorf("repomap: función exportada %q no existe en el módulo wasm", name)
	}
	res, err := fn.Call(ctx, args...)
	if err != nil {
		return 0, fmt.Errorf("repomap: fallo al invocar %q: %w", name, err)
	}
	if len(res) == 0 {
		return 0, nil
	}
	return res[0], nil
}

// readCString lee una cadena NUL-terminated de la memoria lineal del módulo a partir de ptr. Las
// cadenas que expone el shim (nombres de tipo de nodo, nombres de campo) viven en el almacenamiento
// estático de la gramática y siempre son identificadores cortos, por lo que cStringMaxLen es
// generoso; si ptr es 0 (NULL, "sin nombre de campo") devuelve cadena vacía.
func (p *parser) readCString(ptr uint32) string {
	if ptr == 0 {
		return ""
	}
	mem := p.mod.Memory()
	n := cStringMaxLen
	if size := mem.Size(); ptr >= size {
		return ""
	} else if remaining := size - ptr; uint32(n) > remaining {
		n = int(remaining)
	}
	buf, ok := mem.Read(ptr, uint32(n))
	if !ok {
		return ""
	}
	if i := bytes.IndexByte(buf, 0); i >= 0 {
		return string(buf[:i])
	}
	return string(buf)
}

// ensureTSParser crea (una sola vez, de forma perezosa) y reutiliza el TSParser* interno de este
// módulo. ts_parser_parse_string resetea el estado necesario en cada llamada, así que un solo
// TSParser puede reutilizarse de forma segura para parsear muchos archivos del mismo lenguaje.
func (p *parser) ensureTSParser(ctx context.Context) (uint32, error) {
	if p.tsParser != 0 {
		return p.tsParser, nil
	}
	h, err := p.call(ctx, "tsm_new_parser")
	if err != nil {
		return 0, err
	}
	if h == 0 {
		return 0, fmt.Errorf("repomap: tsm_new_parser devolvió NULL (no se pudo enlazar el lenguaje)")
	}
	p.tsParser = uint32(h)
	return p.tsParser, nil
}

// astNode identifica un TSNode "boxeado" (puntero heap dentro de la memoria lineal del módulo; ver
// box_node en wasmsrc/shim.c) junto con el parser al que pertenece, para poder invocar los
// accesores correctos. No es válido fuera de la llamada a parseTopLevel que lo produjo.
type astNode struct {
	p   *parser
	ptr uint32
}

func (n astNode) typ(ctx context.Context) string {
	ptr, err := n.p.call(ctx, "tsm_node_type", uint64(n.ptr))
	if err != nil {
		return ""
	}
	return n.p.readCString(uint32(ptr))
}

func (n astNode) childCount(ctx context.Context) uint32 {
	v, err := n.p.call(ctx, "tsm_node_child_count", uint64(n.ptr))
	if err != nil {
		return 0
	}
	return uint32(v)
}

func (n astNode) child(ctx context.Context, i uint32) (astNode, error) {
	ptr, err := n.p.call(ctx, "tsm_node_child", uint64(n.ptr), uint64(i))
	if err != nil {
		return astNode{}, err
	}
	return astNode{p: n.p, ptr: uint32(ptr)}, nil
}

func (n astNode) fieldNameForChild(ctx context.Context, i uint32) string {
	ptr, err := n.p.call(ctx, "tsm_node_field_name_for_child", uint64(n.ptr), uint64(i))
	if err != nil {
		return ""
	}
	return n.p.readCString(uint32(ptr))
}

func (n astNode) byteRange(ctx context.Context) (start, end uint32) {
	s, _ := n.p.call(ctx, "tsm_node_start_byte", uint64(n.ptr))
	e, _ := n.p.call(ctx, "tsm_node_end_byte", uint64(n.ptr))
	return uint32(s), uint32(e)
}

func (n astNode) startRow(ctx context.Context) uint32 {
	r, _ := n.p.call(ctx, "tsm_node_start_row", uint64(n.ptr))
	return uint32(r)
}

func (n astNode) text(ctx context.Context, src []byte) string {
	start, end := n.byteRange(ctx)
	if end < start || end > uint32(len(src)) {
		return ""
	}
	return string(src[start:end])
}

func (n astNode) free(ctx context.Context) {
	if n.ptr == 0 {
		return
	}
	_, _ = n.p.call(ctx, "tsm_node_free", uint64(n.ptr))
}

// parseTopLevel parsea src y entrega a visit cada hijo directo (nodo "boxeado", liberado
// automáticamente después de que visit retorna) del nodo raíz del árbol — es decir, las
// declaraciones de nivel superior del archivo. El árbol completo y el buffer de origen copiado a
// la memoria wasm se liberan antes de retornar. Debe llamarse con p.mu ya adquirido.
func (p *parser) parseTopLevel(ctx context.Context, src []byte, visit func(astNode)) error {
	tsParser, err := p.ensureTSParser(ctx)
	if err != nil {
		return err
	}

	srcLen := uint32(len(src))
	srcPtr, err := p.call(ctx, "tsm_alloc", uint64(srcLen))
	if err != nil {
		return err
	}
	if srcPtr == 0 {
		return fmt.Errorf("repomap: tsm_alloc devolvió NULL")
	}
	defer p.call(ctx, "tsm_free", uint64(srcPtr))

	if srcLen > 0 && !p.mod.Memory().Write(uint32(srcPtr), src) {
		return fmt.Errorf("repomap: no se pudo escribir el código fuente en memoria wasm")
	}

	treePtr, err := p.call(ctx, "tsm_parse", uint64(tsParser), srcPtr, uint64(srcLen))
	if err != nil {
		return err
	}
	if treePtr == 0 {
		return fmt.Errorf("repomap: tsm_parse devolvió NULL")
	}
	defer p.call(ctx, "tsm_tree_delete", treePtr)

	rootPtr, err := p.call(ctx, "tsm_tree_root_node", treePtr)
	if err != nil {
		return err
	}
	root := astNode{p: p, ptr: uint32(rootPtr)}
	defer root.free(ctx)

	count := root.childCount(ctx)
	for i := uint32(0); i < count; i++ {
		child, err := root.child(ctx, i)
		if err != nil {
			return err
		}
		visit(child)
		child.free(ctx)
	}
	return nil
}
