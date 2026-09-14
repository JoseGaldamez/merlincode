package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"merlincode/internal/domain"
	"merlincode/internal/platform/config"
	"merlincode/internal/platform/keyring"
)

// SecretStore define el contrato para el almacenamiento seguro de credenciales (re-exportado desde platform/keyring).
type SecretStore = keyring.SecretStore

// OSKeyringStore implementa SecretStore usando el llavero nativo del sistema operativo.
type OSKeyringStore = keyring.OSKeyringStore

// NewOSKeyringStore crea una nueva instancia conectada al servicio predeterminado en el llavero.
var NewOSKeyringStore = keyring.NewOSKeyringStore

// ErrSecretNotFound se retorna cuando una cuenta solicitada no existe en el llavero.
var ErrSecretNotFound = keyring.ErrSecretNotFound

// adminKeyringAccount deriva la cuenta del llavero usada para la Admin API Key de un proveedor.
var adminKeyringAccount = keyring.AdminKeyringAccount

// Constantes de seguridad para limitar abusos y consumo accidental
const (
	MaxTestMessageLength    = 4096 // 4 KiB
	MaxResponseTextLength   = 4096 // 4 KiB
	TestMessageCooldownTime = 2 * time.Second
	// LegacyCredentialMigrationSupportUntil documenta el plazo de compatibilidad del formato con claves en JSON.
	LegacyCredentialMigrationSupportUntil = "2027-03-31"
)

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

type providerTestState struct {
	mu       sync.Mutex
	lastTest time.Time
}

// Service gestiona la persistencia segura y validación en vivo de las credenciales de cada proveedor de IA.
// Es agnóstico de qué proveedores existen: solo conoce la interfaz Provider/Validator y el mapa inyectado.
type Service struct {
	mu           sync.RWMutex
	credentials  map[string]domain.ProviderCredential
	filePath     string
	validators   map[string]Provider
	secretStore  SecretStore
	initErr      error
	streamMu     sync.Mutex
	testStatesMu sync.Mutex
	testStates   map[string]*providerTestState
}

// resolveConfigFilePath obtiene la ruta absoluta hacia %APPDATA%/merlincode/ai_providers.json
func resolveConfigFilePath() (string, error) {
	filePath, err := config.GetConfigFilePath("ai_providers.json")
	if err != nil {
		return "", fmt.Errorf("%w: no se pudo resolver el directorio de configuración: %v", domain.ErrProviderMetadataUnavailable, err)
	}
	return filePath, nil
}

// NewService crea el servicio productivo usando el registro por defecto y el llavero nativo del SO
func NewService() *Service {
	filePath, err := resolveConfigFilePath()
	if err != nil {
		return &Service{
			credentials: make(map[string]domain.ProviderCredential),
			validators:  defaultRegistry,
			secretStore: NewOSKeyringStore(),
			testStates:  make(map[string]*providerTestState),
			initErr:     err,
		}
	}
	return NewServiceWithStore(filePath, defaultRegistry, NewOSKeyringStore())
}

// NewServiceWithOptions permite inyectar una ruta y un mapa de validadores personalizados
func NewServiceWithOptions(filePath string, validators map[string]Provider) *Service {
	return NewServiceWithStore(filePath, validators, NewOSKeyringStore())
}

// NewServiceWithStore permite inyectar un SecretStore mock para pruebas unitarias sin dependencias externas
func NewServiceWithStore(filePath string, validators map[string]Provider, store SecretStore) *Service {
	s := &Service{
		credentials: make(map[string]domain.ProviderCredential),
		filePath:    filePath,
		validators:  validators,
		secretStore: store,
		testStates:  make(map[string]*providerTestState),
	}
	if store == nil {
		s.initErr = fmt.Errorf("%w: almacén de secretos no configurado", domain.ErrCredentialStoreUnavailable)
		return s
	}
	if err := s.loadAndMigrate(filePath); err != nil {
		s.initErr = err
		return s
	}
	if err := s.hydrateFromKeyring(); err != nil {
		s.initErr = err
	}
	return s
}

// InitializationError informa fallos de configuración, migración o acceso al llavero detectados al arrancar.
func (s *Service) InitializationError() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.initErr
}

func ensurePrivateDirectory(dir string) error {
	if err := config.EnsurePrivateDirectory(dir); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrProviderMetadataUnavailable, err)
	}
	return nil
}

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

func (s *Service) getProviderTestState(providerID string) *providerTestState {
	s.testStatesMu.Lock()
	defer s.testStatesMu.Unlock()
	state, exists := s.testStates[providerID]
	if !exists {
		state = &providerTestState{}
		s.testStates[providerID] = state
	}
	return state
}

// SendTestMessage envía un mensaje real de prueba validando modelo, tamaño, concurrencia y cooldown.
func (s *Service) SendTestMessage(ctx context.Context, providerID string, message string, model string) (domain.TestMessageResult, error) {
	if err := s.InitializationError(); err != nil {
		return domain.TestMessageResult{}, err
	}
	validator, ok := s.validators[providerID]
	if !ok {
		return domain.TestMessageResult{}, domain.ErrUnknownAIProvider
	}

	mt, ok := validator.(MessageTester)
	if !ok {
		return domain.TestMessageResult{
			Success: false,
			Message: "Este proveedor no admite el envío de mensajes de prueba desde la interfaz.",
		}, nil
	}

	// 1. Limitar tamaño del mensaje de prueba (máximo 4 KiB)
	if len(message) > MaxTestMessageLength {
		return domain.TestMessageResult{
			Success: false,
			Message: "El mensaje de prueba excede el límite máximo permitido de 4 KiB.",
		}, domain.ErrMessageTooLarge
	}

	s.mu.RLock()
	cred, hasCred := s.credentials[providerID]
	s.mu.RUnlock()

	if !hasCred || cred.APIKey == "" {
		return domain.TestMessageResult{Success: false, Message: "Configura y verifica una clave de API primero."}, nil
	}

	model = strings.TrimSpace(model)
	if model == "" {
		model = GetOrchestratorModel(providerID)
	}

	// 2. Validar que el modelo solicitado pertenece a los permitidos para este proveedor
	if !IsModelAllowedForProvider(providerID, model, cred.Models) {
		return domain.TestMessageResult{
			Success: false,
			Message: "El modelo seleccionado no está permitido o no es compatible con este proveedor.",
		}, domain.ErrInvalidModel
	}

	// 3. Control de concurrencia y cooldown por proveedor
	state := s.getProviderTestState(providerID)
	if !state.mu.TryLock() {
		return domain.TestMessageResult{
			Success: false,
			Message: "Ya hay una solicitud de prueba en curso para este proveedor. Espera a que finalice.",
		}, domain.ErrConcurrentRequestBlocked
	}
	defer state.mu.Unlock()

	if time.Since(state.lastTest) < TestMessageCooldownTime {
		return domain.TestMessageResult{
			Success: false,
			Message: "Espera unos segundos antes de enviar otro mensaje de prueba.",
		}, domain.ErrRateLimited
	}

	result, err := mt.SendTestMessage(ctx, cred.APIKey, message, model)
	state.lastTest = time.Now()

	// 4. Limitar tamaño de respuesta devuelta al frontend (máximo 4 KiB)
	if len(result.ResponseText) > MaxResponseTextLength {
		result.ResponseText = result.ResponseText[:MaxResponseTextLength] + "..."
	}

	return result, err
}

// GetRawAPIKey retorna la clave completa almacenada de un proveedor para uso interno del backend
func (s *Service) GetRawAPIKey(providerID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cred, ok := s.credentials[providerID]
	if !ok || cred.APIKey == "" {
		return "", false
	}
	return cred.APIKey, true
}

// ValidateChatRequest rejects untrusted model IDs and oversized history before persistence or HTTP.
const MaxChatInputBytes = 256 * 1024
const MaxChatMessages = 200
const MaxChatOutputBytes = 2 * 1024 * 1024
const ChatTimeout = 5 * time.Minute

func ValidateChatRequest(providerID, model string, messages []domain.ChatMessage) error {
	if _, ok := GetProviderConfig(providerID); !ok {
		return domain.ErrUnknownAIProvider
	}
	if model == "" {
		model = GetOrchestratorModel(providerID)
	}
	if !IsModelAllowedForProvider(providerID, model, nil) {
		return domain.ErrInvalidModel
	}
	if len(messages) == 0 {
		return domain.ErrInvalidChat
	}
	if len(messages) > MaxChatMessages {
		return domain.ErrChatTooLarge
	}
	size := 0
	for _, m := range messages {
		if m.Role != domain.ChatRoleUser && m.Role != domain.ChatRoleAssistant && m.Role != domain.ChatRoleSystem {
			return domain.ErrInvalidChat
		}
		if len(m.Content) > MaxChatInputBytes-size {
			return domain.ErrChatTooLarge
		}
		size += len(m.Content)
	}
	return nil
}

// StreamChat bounds input, output, duration and concurrency independently of the WebView.
func (s *Service) StreamChat(ctx context.Context, providerID, model string, messages []domain.ChatMessage,
	onChunk func(domain.StreamChunk) error) (*domain.ChatCompletionResult, error) {
	if err := s.InitializationError(); err != nil {
		return nil, err
	}
	providerID = NormalizeProviderID(providerID)
	if model == "" {
		model = GetOrchestratorModel(providerID)
	}
	if err := ValidateChatRequest(providerID, model, messages); err != nil {
		return nil, err
	}
	if !s.streamMu.TryLock() {
		return nil, domain.ErrChatBusy
	}
	defer s.streamMu.Unlock()
	streamer, ok := s.validators[providerID].(Streamer)
	if !ok {
		return nil, domain.ErrAIProviderOperationFailed
	}
	s.mu.RLock()
	apiKey := s.credentials[providerID].APIKey
	if apiKey == "" {
		secret, _, err := s.readSecret(providerID)
		if err != nil {
			s.mu.RUnlock()
			return nil, err
		}
		apiKey = secret
	}
	s.mu.RUnlock()
	if apiKey == "" {
		return nil, errors.New("no hay una clave de API configurada para el proveedor")
	}
	ctx, cancel := context.WithTimeout(ctx, ChatTimeout)
	defer cancel()
	size := 0
	result, err := streamer.StreamChat(ctx, apiKey, model, messages, func(chunk domain.StreamChunk) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if len(chunk.Text) > MaxChatOutputBytes-size {
			return domain.ErrChatOutputTooLarge
		}
		size += len(chunk.Text)
		if len(chunk.Thinking) > MaxChatOutputBytes-size {
			return domain.ErrChatOutputTooLarge
		}
		size += len(chunk.Thinking)
		if onChunk != nil {
			return onChunk(chunk)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if result == nil {
		return nil, domain.ErrAIProviderOperationFailed
	}
	if len(result.Content) > MaxChatOutputBytes || len(result.Thinking) > MaxChatOutputBytes-len(result.Content) {
		return nil, domain.ErrChatOutputTooLarge
	}
	return result, nil
}
