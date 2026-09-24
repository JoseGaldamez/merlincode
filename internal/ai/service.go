package ai

import (
	"fmt"
	"sync"

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
