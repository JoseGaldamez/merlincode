// shim.c - flat, scalar-only C ABI exposing a minimal slice of tree-sitter's
// public API (api.h) for consumption from a wasm host that cannot pass or
// receive C struct values across the wasm function-call boundary (wazero can
// only pass/return i32/i64/f32/f64). Each TSNode (a value struct) is "boxed":
// heap-allocated inside this module's own linear memory, and the exported
// functions hand back/accept that pointer as a plain i32 handle. The host
// (Go) never interprets the bytes of a TSNode directly.
//
// This file is intentionally language-agnostic: tree_sitter_<language> is
// supplied by whichever grammar object file is linked alongside it (see the
// build script in internal/repomap/wasmsrc/build.sh). Adding another grammar
// means linking its generated parser.c against this same shim and renaming
// the extern declaration below (or adding a second exported constructor).

#include "tree_sitter/api.h"
#include <stdlib.h>

// Provided by the grammar's generated parser.c linked into this module.
extern const TSLanguage *TREE_SITTER_LANGUAGE_FN(void);

__attribute__((export_name("tsm_alloc")))
uint8_t *tsm_alloc(uint32_t size) {
  return (uint8_t *)malloc(size > 0 ? size : 1);
}

__attribute__((export_name("tsm_free")))
void tsm_free(void *ptr) {
  free(ptr);
}

__attribute__((export_name("tsm_new_parser")))
TSParser *tsm_new_parser(void) {
  TSParser *parser = ts_parser_new();
  if (!parser) {
    return NULL;
  }
  if (!ts_parser_set_language(parser, TREE_SITTER_LANGUAGE_FN())) {
    ts_parser_delete(parser);
    return NULL;
  }
  return parser;
}

__attribute__((export_name("tsm_parser_delete")))
void tsm_parser_delete(TSParser *parser) {
  if (parser) {
    ts_parser_delete(parser);
  }
}

__attribute__((export_name("tsm_parse")))
TSTree *tsm_parse(TSParser *parser, const char *source, uint32_t length) {
  return ts_parser_parse_string(parser, NULL, source, length);
}

__attribute__((export_name("tsm_tree_delete")))
void tsm_tree_delete(TSTree *tree) {
  if (tree) {
    ts_tree_delete(tree);
  }
}

static TSNode *box_node(TSNode node) {
  TSNode *box = (TSNode *)malloc(sizeof(TSNode));
  *box = node;
  return box;
}

__attribute__((export_name("tsm_tree_root_node")))
TSNode *tsm_tree_root_node(TSTree *tree) {
  return box_node(ts_tree_root_node(tree));
}

__attribute__((export_name("tsm_node_free")))
void tsm_node_free(TSNode *node) {
  free(node);
}

__attribute__((export_name("tsm_node_child_count")))
uint32_t tsm_node_child_count(TSNode *node) {
  return ts_node_child_count(*node);
}

__attribute__((export_name("tsm_node_child")))
TSNode *tsm_node_child(TSNode *node, uint32_t index) {
  return box_node(ts_node_child(*node, index));
}

__attribute__((export_name("tsm_node_is_named")))
int32_t tsm_node_is_named(TSNode *node) {
  return ts_node_is_named(*node) ? 1 : 0;
}

__attribute__((export_name("tsm_node_is_null")))
int32_t tsm_node_is_null(TSNode *node) {
  return ts_node_is_null(*node) ? 1 : 0;
}

// Returns a pointer to a NUL-terminated string living in the language's
// static symbol-name table. Valid for the lifetime of the parser; the host
// reads it directly out of linear memory without needing to free it.
__attribute__((export_name("tsm_node_type")))
const char *tsm_node_type(TSNode *node) {
  return ts_node_type(*node);
}

__attribute__((export_name("tsm_node_start_byte")))
uint32_t tsm_node_start_byte(TSNode *node) {
  return ts_node_start_byte(*node);
}

__attribute__((export_name("tsm_node_end_byte")))
uint32_t tsm_node_end_byte(TSNode *node) {
  return ts_node_end_byte(*node);
}

__attribute__((export_name("tsm_node_start_row")))
uint32_t tsm_node_start_row(TSNode *node) {
  return ts_node_start_point(*node).row;
}

// Returns the field name (NUL-terminated, static storage) tree-sitter
// assigned to the child at `index`, or NULL if that child has no field name.
// Needed instead of ts_node_child_by_field_name because grammars like Go's
// const_spec/var_spec reuse the same field name ("name") for multiple
// sibling children (e.g. `const A, B = 1, 2`), and child_by_field_name only
// ever returns the first match.
__attribute__((export_name("tsm_node_field_name_for_child")))
const char *tsm_node_field_name_for_child(TSNode *node, uint32_t index) {
  return ts_node_field_name_for_child(*node, index);
}
