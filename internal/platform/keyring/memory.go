package keyring

import "sync"

// MemorySecretStore provee una implementación en memoria de SecretStore para pruebas y desarrollo.
type MemorySecretStore struct {
	mu           sync.RWMutex
	secrets      map[string]string
	deleteError  error
	deleteErrors map[string]error
	getError     error
	setError     error
}

// NewMemorySecretStore crea una nueva instancia de MemorySecretStore aislada.
func NewMemorySecretStore() *MemorySecretStore {
	return &MemorySecretStore{
		secrets:      make(map[string]string),
		deleteErrors: make(map[string]error),
	}
}

// Get retorna el secreto en memoria para la cuenta especificada.
func (m *MemorySecretStore) Get(account string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.getError != nil {
		return "", m.getError
	}
	val, ok := m.secrets[account]
	if !ok {
		return "", ErrSecretNotFound
	}
	return val, nil
}

// Set guarda el secreto en memoria para la cuenta especificada.
func (m *MemorySecretStore) Set(account string, secret string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.setError != nil {
		return m.setError
	}
	m.secrets[account] = secret
	return nil
}

// Delete elimina el secreto en memoria para la cuenta especificada.
func (m *MemorySecretStore) Delete(account string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.deleteError != nil {
		return m.deleteError
	}
	if err := m.deleteErrors[account]; err != nil {
		return err
	}
	if _, ok := m.secrets[account]; !ok {
		return ErrSecretNotFound
	}
	delete(m.secrets, account)
	return nil
}

// SetGetError inyecta un error configurable en las lecturas de secretos.
func (m *MemorySecretStore) SetGetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getError = err
}

// SetSetError inyecta un error configurable en las escrituras de secretos.
func (m *MemorySecretStore) SetSetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.setError = err
}

// SetDeleteError inyecta un error global configurable en las eliminaciones de secretos.
func (m *MemorySecretStore) SetDeleteError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deleteError = err
}

// SetAccountDeleteError inyecta un error específico para una cuenta al eliminar.
func (m *MemorySecretStore) SetAccountDeleteError(account string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deleteErrors[account] = err
}
