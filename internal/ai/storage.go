package ai

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"merlincode/internal/domain"
	"merlincode/internal/platform/config"
)

// LegacyCredentialMigrationSupportUntil documenta el plazo de compatibilidad del formato con claves en JSON.
const LegacyCredentialMigrationSupportUntil = "2027-03-31"

// persistedProviderMetadata es el DTO privado que se persiste en disco.
// NO contiene claves secretas (APIKey, AdminAPIKey) ni información financiera/balances.
type persistedProviderMetadata struct {
	ProviderID string   `json:"providerId"`
	Verified   bool     `json:"verified"`
	VerifiedAt string   `json:"verifiedAt,omitempty"`
	Models     []string `json:"models,omitempty"`
}

// legacyProviderCredentialDTO se usa exclusivamente para leer formatos de archivo antiguos
// que guardaban claves en texto plano, a fin de migrarlas al llavero de forma segura.
type legacyProviderCredentialDTO struct {
	ProviderID  string   `json:"providerId"`
	APIKey      string   `json:"apiKey,omitempty"`
	AdminAPIKey string   `json:"adminApiKey,omitempty"`
	Verified    bool     `json:"verified"`
	VerifiedAt  string   `json:"verifiedAt,omitempty"`
	AccountInfo string   `json:"accountInfo,omitempty"`
	Models      []string `json:"models,omitempty"`
}

func ensurePrivateDirectory(dir string) error {
	if err := config.EnsurePrivateDirectory(dir); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrProviderMetadataUnavailable, err)
	}
	return nil
}

// loadAndMigrate lee metadata y migra el formato heredado. La compatibilidad se mantiene
// hasta LegacyCredentialMigrationSupportUntil para permitir actualizaciones escalonadas.
func (s *Service) loadAndMigrate(filePath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if filePath == "" {
		return fmt.Errorf("%w: ruta de metadata vacía", domain.ErrProviderMetadataUnavailable)
	}
	if err := ensurePrivateDirectory(filepath.Dir(filePath)); err != nil {
		return err
	}
	if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return fmt.Errorf("%w: no se pudo inspeccionar la metadata: %v", domain.ErrProviderMetadataUnavailable, err)
	}
	// Restringir el archivo antes de leer posibles secretos heredados.
	if err := config.EnsurePrivateFile(filePath); err != nil {
		return fmt.Errorf("%w: no se pudieron restringir los permisos de la metadata: %v", domain.ErrProviderMetadataUnavailable, err)
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("%w: no se pudo leer la metadata: %v", domain.ErrProviderMetadataUnavailable, err)
	}

	var legacyMap map[string]legacyProviderCredentialDTO
	if err := json.Unmarshal(data, &legacyMap); err != nil {
		return fmt.Errorf("%w: el archivo de metadata no contiene JSON válido", domain.ErrProviderMetadataUnavailable)
	}

	needsSanitizedRewrite := false

	for id, item := range legacyMap {
		cred := domain.ProviderCredential{
			ProviderID: id,
			Verified:   item.Verified,
			VerifiedAt: item.VerifiedAt,
			Models:     item.Models,
		}

		if item.APIKey != "" {
			needsSanitizedRewrite = true
			if err := s.setSecretVerified(id, item.APIKey); err != nil {
				return fmt.Errorf("%w: %v", domain.ErrCredentialMigrationFailed, err)
			}
			cred.APIKey = item.APIKey
		}

		if item.AdminAPIKey != "" {
			needsSanitizedRewrite = true
			if err := s.setSecretVerified(adminKeyringAccount(id), item.AdminAPIKey); err != nil {
				return fmt.Errorf("%w: %v", domain.ErrCredentialMigrationFailed, err)
			}
			cred.AdminAPIKey = item.AdminAPIKey
		}

		s.credentials[id] = cred
		if item.AccountInfo != "" {
			needsSanitizedRewrite = true
		}
	}

	// Reescribir sin secretos únicamente después de verificar toda la migración.
	if needsSanitizedRewrite {
		if err := s.persistCredentialsLocked(); err != nil {
			return fmt.Errorf("%w: no se pudo reescribir la metadata sanitizada: %v", domain.ErrCredentialMigrationFailed, err)
		}
	}
	return nil
}

// hydrateFromKeyring completa las claves secretas leyendo el llavero del sistema operativo.
// Además, examina las cuentas de todos los proveedores conocidos para detectar secretos huérfanos.
func (s *Service) hydrateFromKeyring() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	providerIDs := make(map[string]struct{}, len(KnownProviderIDs)+len(s.credentials))
	for _, id := range KnownProviderIDs {
		providerIDs[id] = struct{}{}
	}
	for id := range s.credentials {
		providerIDs[id] = struct{}{}
	}

	for id := range providerIDs {
		apiKey, hasAPIKey, err := s.readSecret(id)
		if err != nil {
			return err
		}
		adminKey, hasAdminKey, err := s.readSecret(adminKeyringAccount(id))
		if err != nil {
			return err
		}
		cred, hasMetadata := s.credentials[id]
		if !hasMetadata && !hasAPIKey && !hasAdminKey {
			continue
		}
		cred.ProviderID = id
		cred.APIKey = apiKey
		cred.AdminAPIKey = adminKey
		if !hasMetadata && hasAPIKey {
			cred.Verified = true
			cred.AccountInfo = "Credencial sincronizada desde el llavero del sistema"
		}
		s.credentials[id] = cred
	}
	return nil
}

// persistCredentialsLocked serializa los metadatos no sensibles a disco de forma atómica.
// Requiere tener s.mu adquirido en modo Lock.
func (s *Service) persistCredentialsLocked() error {
	if s.filePath == "" {
		return domain.ErrProviderMetadataUnavailable
	}
	redacted := make(map[string]persistedProviderMetadata, len(s.credentials))
	for id, cred := range s.credentials {
		redacted[id] = persistedProviderMetadata{
			ProviderID: id,
			Verified:   cred.Verified,
			VerifiedAt: cred.VerifiedAt,
			Models:     cred.Models,
		}
	}

	data, err := json.MarshalIndent(redacted, "", "  ")
	if err != nil {
		return fmt.Errorf("%w: no se pudo serializar la metadata: %v", domain.ErrProviderMetadataUnavailable, err)
	}

	dir := filepath.Dir(s.filePath)
	if err := ensurePrivateDirectory(dir); err != nil {
		return err
	}
	if _, err := os.Stat(s.filePath); err == nil {
		if err := config.EnsurePrivateFile(s.filePath); err != nil {
			return fmt.Errorf("%w: no se pudieron corregir los permisos existentes: %v", domain.ErrProviderMetadataUnavailable, err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%w: no se pudo inspeccionar la metadata existente: %v", domain.ErrProviderMetadataUnavailable, err)
	}

	tmpFile, err := os.CreateTemp(dir, "ai_providers_*.tmp")
	if err != nil {
		return fmt.Errorf("%w: no se pudo crear el archivo temporal: %v", domain.ErrProviderMetadataUnavailable, err)
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }() // Limpieza best-effort; el error principal tiene prioridad.

	if err := config.EnsurePrivateFile(tmpPath); err != nil {
		if closeErr := tmpFile.Close(); closeErr != nil {
			return fmt.Errorf("%w: no se pudieron restringir los permisos temporales: %v", domain.ErrProviderMetadataUnavailable, errors.Join(err, closeErr))
		}
		return fmt.Errorf("%w: no se pudieron restringir los permisos temporales: %v", domain.ErrProviderMetadataUnavailable, err)
	}

	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("%w: no se pudo escribir la metadata temporal: %v", domain.ErrProviderMetadataUnavailable, errors.Join(err, tmpFile.Close()))
	}

	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("%w: no se pudo sincronizar la metadata temporal: %v", domain.ErrProviderMetadataUnavailable, errors.Join(err, tmpFile.Close()))
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("%w: no se pudo cerrar la metadata temporal: %v", domain.ErrProviderMetadataUnavailable, err)
	}

	if err := os.Rename(tmpPath, s.filePath); err != nil {
		return fmt.Errorf("%w: no se pudo reemplazar atómicamente la metadata: %v", domain.ErrProviderMetadataUnavailable, err)
	}
	if err := config.EnsurePrivateFile(s.filePath); err != nil {
		return fmt.Errorf("%w: no se pudieron fijar permisos privados en la metadata: %v", domain.ErrProviderMetadataUnavailable, err)
	}
	return nil
}
