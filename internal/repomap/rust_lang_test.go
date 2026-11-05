package repomap

import "testing"

const sampleRustSource = `use std::fmt;

pub fn greet(name: &str) -> String { format!("hi {}", name) }
fn private() {}
pub(crate) fn crate_only() {}
pub(super) struct SuperOnly;
pub(in crate::a) fn restricted() {}
pub struct Config { pub name: String }
pub enum Color { Red, Green }
pub union Bits { a: u32 }
pub trait Describe { fn describe(&self) -> String; }
pub const MAX: u32 = 3;
pub static GLOBAL: i32 = 0;
pub type Alias = Vec<u8>;
pub async unsafe fn danger() {}

pub mod nested {
    pub fn inner() {}
    fn inner_private() {}
    pub mod deeper { pub struct Deep; }
    impl Helper { pub fn assist() {} }
}
mod private_mod { pub fn unreachable() {} }
pub mod external;

impl Config {
    pub fn new() -> Self { todo!() }
    fn hidden(&self) {}
}
impl<T: Clone> Stack<T> { pub fn push(&mut self, _v: T) {} }
impl fmt::Display for Config {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result { Ok(()) }
}

#[derive(Debug)]
pub struct Derived;
macro_rules! my_macro { () => {} }
pub use crate::nested::inner;
struct Hidden;
`

// TestExtractRustSymbolsParseaASTReal instancia rust.wasm (tree-sitter-rust + su external
// scanner) vía wazero y verifica exactamente los items `pub` sin restricción de código Rust real,
// incluyendo métodos de impl inherentes y items de `pub mod` en línea, sin mocks.
func TestExtractRustSymbolsParseaASTReal(t *testing.T) {
	symbols := extractWithRealWasm(t, rustWasm, extractRustSymbols, sampleRustSource)

	// Ausentes a propósito: private, crate_only/SuperOnly/restricted (visibilidad restringida),
	// nested::inner_private, private_mod y su contenido (mod no pub), Config.hidden (no pub),
	// los métodos de `impl Display for Config` (impl de trait), my_macro, el `pub use` y Hidden.
	assertExactSymbols(t, symbols, map[string]string{
		"greet":                 "fn",
		"Config":                "struct",
		"Color":                 "enum",
		"Bits":                  "union",
		"Describe":              "trait",
		"MAX":                   "const",
		"GLOBAL":                "static",
		"Alias":                 "type",
		"danger":                "fn",
		"nested":                "mod",
		"nested::inner":         "fn",
		"nested::deeper":        "mod",
		"nested::deeper::Deep":  "struct",
		"nested::Helper.assist": "method",
		"external":              "mod",
		"Config.new":            "method",
		"Stack.push":            "method",
		"Derived":               "struct",
	})

	if got := symbolLine(symbols, "greet"); got != 3 {
		t.Errorf("esperaba greet en la línea 3, obtenido %d", got)
	}
	if got := symbolLine(symbols, "Config.new"); got != 27 {
		t.Errorf("esperaba Config.new en la línea 27, obtenido %d", got)
	}
	// El atributo #[derive] es un nodo hermano previo: la línea es la del nombre del struct.
	if got := symbolLine(symbols, "Derived"); got != 36 {
		t.Errorf("esperaba Derived en la línea 36, obtenido %d", got)
	}
}
