package desktop

import (
	"errors"
	"testing"

	"merlincode/internal/domain"
)

func TestOpenInExplorerRejectsEmptyPath(t *testing.T) {
	err := OpenInExplorer("")
	if !errors.Is(err, domain.ErrNoTargetPath) {
		t.Fatalf("Esperado ErrNoTargetPath, obtenido: %v", err)
	}
}

func TestOpenInExplorerRejectsNonExistentPath(t *testing.T) {
	err := OpenInExplorer("Z:\\this\\path\\does\\not\\exist_for_merlincode_test")
	if err == nil {
		t.Fatal("Esperado error para ruta inexistente, pero se obtuvo nil")
	}
}
