package ai

import (
	"context"
	"errors"
	"strings"
	"time"

	"merlincode/internal/domain"
)

// maskAPIKey oculta el cuerpo de la clave conservando solo un fragmento de inicio y fin
func maskAPIKey(key string) string {
	key = strings.TrimSpace(key)
	if len(key) <= 8 {
		return "••••••••"
	}
	return key[:4] + "••••••••" + key[len(key)-4:]
}

// supportsAdminKey indica si el proveedor implementa UsageFetcher y requiere una Admin API Key
func (s *Service) supportsAdminKey(providerID string) bool {
	validator, ok := s.validators[providerID]
	if !ok {
		return false
	}
	uf, ok := validator.(UsageFetcher)
	return ok && uf.RequiresAdminKeyForUsage()
}

func toStatus(providerID string, cred domain.ProviderCredential, hasCred bool, supportsAdminKey bool) domain.ProviderStatus {
	status := domain.ProviderStatus{
		ProviderID:       providerID,
		SupportsAdminKey: supportsAdminKey,
	}
	if !hasCred {
		return status
	}
	if cred.APIKey != "" {
		status.Configured = true
		status.Verified = cred.Verified
		status.VerifiedAt = cred.VerifiedAt
		status.AccountInfo = cred.AccountInfo
		status.MaskedKey = maskAPIKey(cred.APIKey)
		status.Models = cred.Models
	}
	if cred.AdminAPIKey != "" {
		status.HasAdminKey = true
		status.MaskedAdminKey = maskAPIKey(cred.AdminAPIKey)
	}
	return status
}

// GetStatus retorna el estado seguro (sin exponer las keys completas) de un proveedor
func (s *Service) GetStatus(providerID string) domain.ProviderStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cred, ok := s.credentials[providerID]
	return toStatus(providerID, cred, ok, s.supportsAdminKey(providerID))
}

// ListStatuses retorna el estado de todos los proveedores conocidos, en un orden estable
func (s *Service) ListStatuses() []domain.ProviderStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	statuses := make([]domain.ProviderStatus, 0, len(KnownProviderIDs))
	for _, id := range KnownProviderIDs {
		cred, ok := s.credentials[id]
		statuses = append(statuses, toStatus(id, cred, ok, s.supportsAdminKey(id)))
	}
	return statuses
}

// ValidateAndSaveKey conecta en vivo contra el proveedor real para confirmar que la clave
// funciona y obtener datos básicos de la cuenta. Solo persiste la clave si la validación es exitosa.
func (s *Service) ValidateAndSaveKey(ctx context.Context, providerID string, apiKey string) (domain.ProviderValidationResult, error) {
	if err := s.InitializationError(); err != nil {
		return domain.ProviderValidationResult{}, err
	}
	validator, ok := s.validators[providerID]
	if !ok {
		return domain.ProviderValidationResult{}, domain.ErrUnknownAIProvider
	}

	result, err := validator.Validate(ctx, apiKey)
	if err != nil {
		return domain.ProviderValidationResult{}, err
	}

	if !result.Valid {
		return result, nil
	}

	trimmedKey := strings.TrimSpace(apiKey)
	s.mu.Lock()
	defer s.mu.Unlock()
	previousSecret, previousExists, err := s.readSecret(providerID)
	if err != nil {
		return domain.ProviderValidationResult{}, err
	}
	if err := s.setSecretVerified(providerID, trimmedKey); err != nil {
		_, _, refreshErr := s.refreshProviderSecretsLocked(providerID)
		return domain.ProviderValidationResult{}, errors.Join(err, refreshErr)
	}

	models := result.Models
	if len(models) == 0 {
		models = AllowedModelsForProvider(providerID)
	}
	result.Models = models

	previousCredential, hadPreviousCredential := s.credentials[providerID]
	s.credentials[providerID] = domain.ProviderCredential{
		ProviderID:  providerID,
		APIKey:      trimmedKey,
		AdminAPIKey: s.credentials[providerID].AdminAPIKey,
		Verified:    true,
		VerifiedAt:  time.Now().UTC().Format(time.RFC3339),
		AccountInfo: result.AccountInfo,
		Models:      models,
	}
	if err := s.persistCredentialsLocked(); err != nil {
		if hadPreviousCredential {
			s.credentials[providerID] = previousCredential
		} else {
			delete(s.credentials, providerID)
		}
		rollbackErr := s.restoreSecret(providerID, previousSecret, previousExists)
		_, _, refreshErr := s.refreshProviderSecretsLocked(providerID)
		return domain.ProviderValidationResult{}, errors.Join(err, rollbackErr, refreshErr)
	}

	return result, nil
}

// ClearKey elimina la credencial guardada de un proveedor, tanto del llavero nativo del sistema
// operativo como de los metadatos en disco. Retorna error si el llavero falla.
func (s *Service) ClearKey(providerID string) error {
	if err := s.InitializationError(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	deleteErr := s.deleteSecretVerified(providerID)
	if deleteErr == nil {
		deleteErr = s.deleteSecretVerified(adminKeyringAccount(providerID))
	}

	_, _, refreshErr := s.refreshProviderSecretsLocked(providerID)
	if refreshErr != nil {
		return errors.Join(deleteErr, refreshErr)
	}
	persistErr := s.persistCredentialsLocked()
	return errors.Join(deleteErr, persistErr)
}

// SaveAdminKey guarda la Admin API Key de un proveedor que la requiera para reportar uso real
func (s *Service) SaveAdminKey(providerID string, adminKey string) error {
	if err := s.InitializationError(); err != nil {
		return err
	}
	if !s.supportsAdminKey(providerID) {
		return domain.ErrAdminKeyNotSupported
	}

	trimmedAdminKey := strings.TrimSpace(adminKey)
	account := adminKeyringAccount(providerID)
	s.mu.Lock()
	defer s.mu.Unlock()
	previousSecret, previousExists, err := s.readSecret(account)
	if err != nil {
		return err
	}
	if err := s.setSecretVerified(account, trimmedAdminKey); err != nil {
		_, _, refreshErr := s.refreshProviderSecretsLocked(providerID)
		return errors.Join(err, refreshErr)
	}

	previousCredential, hadPreviousCredential := s.credentials[providerID]
	cred := previousCredential
	cred.ProviderID = providerID
	cred.AdminAPIKey = trimmedAdminKey
	s.credentials[providerID] = cred
	if err := s.persistCredentialsLocked(); err != nil {
		if hadPreviousCredential {
			s.credentials[providerID] = previousCredential
		} else {
			delete(s.credentials, providerID)
		}
		rollbackErr := s.restoreSecret(account, previousSecret, previousExists)
		_, _, refreshErr := s.refreshProviderSecretsLocked(providerID)
		return errors.Join(err, rollbackErr, refreshErr)
	}
	return nil
}

// ClearAdminKey elimina la Admin API Key guardada de un proveedor (del llavero y de metadata)
func (s *Service) ClearAdminKey(providerID string) error {
	if err := s.InitializationError(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	deleteErr := s.deleteSecretVerified(adminKeyringAccount(providerID))
	_, _, refreshErr := s.refreshProviderSecretsLocked(providerID)
	if refreshErr != nil {
		return errors.Join(deleteErr, refreshErr)
	}
	return errors.Join(deleteErr, s.persistCredentialsLocked())
}

// GetProviderUsage consulta en vivo el consumo/costo real de la cuenta de un proveedor
func (s *Service) GetProviderUsage(ctx context.Context, providerID string) (domain.ProviderUsageResult, error) {
	if err := s.InitializationError(); err != nil {
		return domain.ProviderUsageResult{}, err
	}
	validator, ok := s.validators[providerID]
	if !ok {
		return domain.ProviderUsageResult{}, domain.ErrUnknownAIProvider
	}

	uf, ok := validator.(UsageFetcher)
	if !ok {
		return domain.ProviderUsageResult{
			Available: false,
			Message:   "Este proveedor no expone datos de uso o facturación vía API con la clave estándar.",
		}, nil
	}

	s.mu.RLock()
	cred := s.credentials[providerID]
	s.mu.RUnlock()

	if cred.APIKey == "" {
		return domain.ProviderUsageResult{Available: false, Message: "Configura y verifica una clave de API primero."}, nil
	}

	return uf.FetchUsage(ctx, cred.APIKey, cred.AdminAPIKey)
}
