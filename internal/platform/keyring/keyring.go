package keyring

import "github.com/zalando/go-keyring"

// DefaultAIProviderServiceName identifica el servicio bajo el cual se agrupan las credenciales de
// proveedores de IA en el llavero nativo del sistema operativo (Credential Manager en Windows,
// Keychain en macOS, Secret Service/libsecret en Linux).
const DefaultAIProviderServiceName = "merlincode-ai-providers"

// ErrSecretNotFound se retorna cuando una cuenta solicitada no existe en el llavero.
var ErrSecretNotFound = keyring.ErrNotFound

// SecretStore define el contrato para el almacenamiento seguro de credenciales en el llavero del SO.
type SecretStore interface {
	Get(account string) (string, error)
	Set(account string, value string) error
	Delete(account string) error
}

// OSKeyringStore implementa SecretStore usando el llavero nativo del sistema operativo mediante go-keyring.
type OSKeyringStore struct {
	service string
}

// NewOSKeyringStore crea una nueva instancia conectada al servicio predeterminado de Merlin Code.
func NewOSKeyringStore() *OSKeyringStore {
	return NewOSKeyringStoreWithService(DefaultAIProviderServiceName)
}

// NewOSKeyringStoreWithService permite inicializar una instancia con un nombre de servicio personalizado.
func NewOSKeyringStoreWithService(service string) *OSKeyringStore {
	return &OSKeyringStore{service: service}
}

// Get obtiene el secreto almacenado para la cuenta indicada.
func (s *OSKeyringStore) Get(account string) (string, error) {
	return keyring.Get(s.service, account)
}

// Set guarda o actualiza el secreto para la cuenta indicada.
func (s *OSKeyringStore) Set(account string, value string) error {
	return keyring.Set(s.service, account, value)
}

// Delete elimina el secreto para la cuenta indicada.
func (s *OSKeyringStore) Delete(account string) error {
	return keyring.Delete(s.service, account)
}

// AdminKeyringAccount deriva la cuenta del llavero usada para la Admin API Key de un proveedor,
// separada de su clave de API estándar.
func AdminKeyringAccount(providerID string) string {
	return providerID + ":admin"
}
