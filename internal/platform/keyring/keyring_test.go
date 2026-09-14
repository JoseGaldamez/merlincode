package keyring

import (
	"errors"
	"testing"
)

func TestMemorySecretStoreBasic(t *testing.T) {
	store := NewMemorySecretStore()

	// Get inexistente
	_, err := store.Get("anthropic")
	if !errors.Is(err, ErrSecretNotFound) {
		t.Fatalf("Esperado ErrSecretNotFound, obtenido %v", err)
	}

	// Set y Get exitosos
	if err := store.Set("anthropic", "sk-ant-test-123"); err != nil {
		t.Fatalf("Set falló: %v", err)
	}

	val, err := store.Get("anthropic")
	if err != nil {
		t.Fatalf("Get falló: %v", err)
	}
	if val != "sk-ant-test-123" {
		t.Fatalf("Esperado sk-ant-test-123, obtenido %s", val)
	}

	// Delete
	if err := store.Delete("anthropic"); err != nil {
		t.Fatalf("Delete falló: %v", err)
	}

	// Delete inexistente
	if err := store.Delete("anthropic"); !errors.Is(err, ErrSecretNotFound) {
		t.Fatalf("Esperado ErrSecretNotFound al borrar cuenta inexistente, obtenido %v", err)
	}
}

func TestAdminKeyringAccount(t *testing.T) {
	acc := AdminKeyringAccount("anthropic")
	if acc != "anthropic:admin" {
		t.Fatalf("Esperado anthropic:admin, obtenido %s", acc)
	}
}
