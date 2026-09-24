package ai

import (
	"errors"
	"fmt"

	"merlincode/internal/domain"
)

// readSecret lee una cuenta del llavero y traduce el caso no encontrado a un booleano seguro.
func (s *Service) readSecret(account string) (string, bool, error) {
	secret, err := s.secretStore.Get(account)
	if errors.Is(err, ErrSecretNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("%w: no se pudo leer la cuenta %q: %v", domain.ErrCredentialStoreUnavailable, account, err)
	}
	return secret, secret != "", nil
}

// setSecretVerified escribe una credencial en el llavero y verifica inmediatamente leyéndola de vuelta.
func (s *Service) setSecretVerified(account string, secret string) error {
	if err := s.secretStore.Set(account, secret); err != nil {
		return fmt.Errorf("%w: no se pudo guardar la cuenta %q: %v", domain.ErrCredentialStoreUnavailable, account, err)
	}
	stored, exists, err := s.readSecret(account)
	if err != nil {
		return err
	}
	if !exists || stored != secret {
		return fmt.Errorf("%w: el llavero no confirmó la cuenta %q", domain.ErrCredentialStoreUnavailable, account)
	}
	return nil
}

// deleteSecretVerified elimina una credencial del llavero y verifica que ya no exista.
func (s *Service) deleteSecretVerified(account string) error {
	if err := s.secretStore.Delete(account); err != nil && !errors.Is(err, ErrSecretNotFound) {
		return fmt.Errorf("%w: no se pudo eliminar la cuenta %q: %v", domain.ErrCredentialStoreUnavailable, account, err)
	}
	_, exists, err := s.readSecret(account)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("%w: el llavero no confirmó la eliminación de la cuenta %q", domain.ErrCredentialStoreUnavailable, account)
	}
	return nil
}

// restoreSecret restaura o elimina una credencial en el llavero durante compensaciones o rollbacks.
func (s *Service) restoreSecret(account string, previous string, existed bool) error {
	if existed {
		return s.setSecretVerified(account, previous)
	}
	return s.deleteSecretVerified(account)
}

// refreshProviderSecretsLocked vuelve a consultar el llavero tras una operación parcial.
// Requiere s.mu en modo escritura y solo actualiza memoria cuando ambas lecturas son concluyentes.
func (s *Service) refreshProviderSecretsLocked(providerID string) (bool, bool, error) {
	apiKey, hasAPIKey, err := s.readSecret(providerID)
	if err != nil {
		return false, false, err
	}
	adminKey, hasAdminKey, err := s.readSecret(adminKeyringAccount(providerID))
	if err != nil {
		return false, false, err
	}

	cred, exists := s.credentials[providerID]
	if !exists && !hasAPIKey && !hasAdminKey {
		return false, false, nil
	}
	cred.ProviderID = providerID
	changedKey := cred.APIKey != apiKey
	cred.APIKey = apiKey
	cred.AdminAPIKey = adminKey
	if !hasAPIKey || changedKey {
		cred.Verified = false
		cred.VerifiedAt = ""
		cred.AccountInfo = ""
		cred.Models = nil
	}
	if !hasAPIKey && !hasAdminKey {
		delete(s.credentials, providerID)
	} else {
		s.credentials[providerID] = cred
	}
	return hasAPIKey, hasAdminKey, nil
}
