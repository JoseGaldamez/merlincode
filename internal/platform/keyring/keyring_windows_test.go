package keyring

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestWindowsCredentialManagerRoundTrip(t *testing.T) {
	// Isolated service name; never read or modify the user's provider accounts.
	store := NewOSKeyringStoreWithService(fmt.Sprintf("merlincode-security-test-%d", time.Now().UnixNano()))
	const account = "synthetic-test"
	t.Cleanup(func() {
		if err := store.Delete(account); err != nil && !errors.Is(err, ErrSecretNotFound) {
			t.Errorf("cleanup: %v", err)
		}
	})
	if err := store.Set(account, "synthetic-sentinel"); err != nil {
		t.Fatal(err)
	}
	if value, err := store.Get(account); err != nil || value != "synthetic-sentinel" {
		t.Fatal("native read-back failed")
	}
	if err := store.Delete(account); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(account); !errors.Is(err, ErrSecretNotFound) {
		t.Fatalf("native delete not confirmed: %v", err)
	}
}
