package repomap

import "testing"

const samplePythonSource = `"""Módulo de ejemplo."""
import os
from typing import List

MAX_RETRIES = 3
_private = 1
__all__ = ["greet"]
a, b = 1, 2
(g, (h, _skipped)) = 1, (2, 3)
x: int = 5
declared_only: str
c = d = 0
obj.attr = 3

def greet(name: str) -> str:
    return "hi " + name

def _helper():
    pass

async def fetch():
    pass

class Config(Base):
    def method(self):
        pass

class _Hidden:
    pass

@decorator
def decorated():
    pass

@dataclass
class Point:
    x: int

type Pair[T] = tuple[T, T]

if TYPE_CHECKING:
    def conditional():
        pass
`

// TestExtractPythonSymbolsParseaASTReal instancia python.wasm (tree-sitter-python + su external
// scanner de indentación) vía wazero y verifica exactamente los nombres públicos por convención
// (sin `_` inicial) del nivel superior de un módulo Python real, sin mocks.
func TestExtractPythonSymbolsParseaASTReal(t *testing.T) {
	symbols := extractWithRealWasm(t, pythonWasm, extractPythonSymbols, samplePythonSource)

	// Ausentes a propósito: _private, __all__ y _skipped (empiezan con `_`), os/List
	// (importados), obj.attr (atributo), _helper/_Hidden (privados por convención), method (es
	// de una clase) y conditional (dentro de un if de nivel superior: fuera de alcance).
	assertExactSymbols(t, symbols, map[string]string{
		"MAX_RETRIES":   "var",
		"a":             "var",
		"b":             "var",
		"g":             "var",
		"h":             "var",
		"x":             "var",
		"declared_only": "var",
		"c":             "var",
		"d":             "var",
		"greet":         "func",
		"fetch":         "func",
		"Config":        "class",
		"decorated":     "func",
		"Point":         "class",
		"Pair":          "type",
	})

	if got := symbolLine(symbols, "greet"); got != 15 {
		t.Errorf("esperaba greet en la línea 15, obtenido %d", got)
	}
	// El nombre de una función decorada se reporta en la línea del `def`, no del decorador.
	if got := symbolLine(symbols, "decorated"); got != 32 {
		t.Errorf("esperaba decorated en la línea 32, obtenido %d", got)
	}
}
