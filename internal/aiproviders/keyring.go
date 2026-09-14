package aiproviders

import (
	"errors"

	"github.com/zalando/go-keyring"
)

// keyringService identifica el servicio bajo el cual se agrupan todas las credenciales de
// Merlin Code en el llavero nativo del sistema operativo (Credential Manager en Windows,
// Keychain en macOS, Secret Service/libsecret en Linux).
const keyringService = "merlincode-ai-providers"

// adminKeyringAccount deriva la cuenta del llavero usada para la Admin API Key de un proveedor,
// separada de su clave de API estándar.
func adminKeyringAccount(providerID string) string {
	return providerID + ":admin"
}

// setKeyringSecret guarda (o sobrescribe) un secreto en el llavero nativo del sistema operativo.
func setKeyringSecret(account string, value string) error {
	return keyring.Set(keyringService, account, value)
}

// getKeyringSecret lee un secreto del llavero nativo. El segundo valor es false si no existe.
func getKeyringSecret(account string) (string, bool) {
	value, err := keyring.Get(keyringService, account)
	if err != nil {
		return "", false
	}
	return value, true
}

// deleteKeyringSecret elimina un secreto del llavero nativo, ignorando el caso en que ya no exista.
func deleteKeyringSecret(account string) {
	err := keyring.Delete(keyringService, account)
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		_ = err // el llavero puede no estar disponible (p. ej. sesión sin GUI en Linux); no es fatal
	}
}
