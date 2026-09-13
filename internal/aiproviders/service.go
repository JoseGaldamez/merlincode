package aiproviders

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"merlincode/internal/domain"
)

// Service gestiona la persistencia y validación en vivo de las credenciales de cada proveedor de IA.
// Es agnóstico de qué proveedores existen: solo conoce la interfaz Validator y el mapa inyectado.
type Service struct {
	mu          sync.RWMutex
	credentials map[string]domain.ProviderCredential
	filePath    string
	validators  map[string]Validator
}

// resolveConfigFilePath obtiene la ruta absoluta hacia %APPDATA%/merlincode/ai_providers.json
func resolveConfigFilePath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	appDir := filepath.Join(configDir, "merlincode")
	_ = os.MkdirAll(appDir, 0755)
	return filepath.Join(appDir, "ai_providers.json")
}

// NewService crea el servicio productivo usando el registro por defecto de validadores
func NewService() *Service {
	return NewServiceWithOptions(resolveConfigFilePath(), defaultRegistry)
}

// NewServiceWithOptions permite inyectar una ruta y un mapa de validadores personalizados,
// lo que facilita las pruebas unitarias sin depender de la red ni del sistema de archivos real.
func NewServiceWithOptions(filePath string, validators map[string]Validator) *Service {
	return &Service{
		credentials: loadCredentialsFromFile(filePath),
		filePath:    filePath,
		validators:  validators,
	}
}

func loadCredentialsFromFile(filePath string) map[string]domain.ProviderCredential {
	creds := make(map[string]domain.ProviderCredential)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return creds
	}
	_ = json.Unmarshal(data, &creds)
	return creds
}

// persistCredentials serializa las credenciales a disco (debe llamarse con el candado adquirido)
func (s *Service) persistCredentials() {
	data, err := json.MarshalIndent(s.credentials, "", "  ")
	if err == nil {
		_ = os.WriteFile(s.filePath, data, 0644)
	}
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
// separada para reportar su consumo/costo real (p. ej. Anthropic), en vez de usar la key normal.
func (s *Service) supportsAdminKey(providerID string) bool {
	validator, ok := s.validators[providerID]
	if !ok {
		return false
	}
	uf, ok := validator.(UsageFetcher)
	return ok && uf.RequiresAdminKeyForUsage()
}

func toStatus(providerID string, cred domain.ProviderCredential, hasCred bool, supportsAdminKey bool) domain.ProviderStatus {
	if !hasCred || cred.APIKey == "" {
		return domain.ProviderStatus{ProviderID: providerID, Configured: false, SupportsAdminKey: supportsAdminKey}
	}
	status := domain.ProviderStatus{
		ProviderID:       providerID,
		Configured:       true,
		Verified:         cred.Verified,
		VerifiedAt:       cred.VerifiedAt,
		AccountInfo:      cred.AccountInfo,
		MaskedKey:        maskAPIKey(cred.APIKey),
		Models:           cred.Models,
		SupportsAdminKey: supportsAdminKey,
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
// funciona y obtener datos básicos de la cuenta. Solo persiste la clave si la validación es exitosa,
// evitando dejar guardada una credencial que se sabe inválida.
func (s *Service) ValidateAndSaveKey(ctx context.Context, providerID string, apiKey string) (domain.ProviderValidationResult, error) {
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

	s.mu.Lock()
	defer s.mu.Unlock()

	s.credentials[providerID] = domain.ProviderCredential{
		ProviderID:  providerID,
		APIKey:      strings.TrimSpace(apiKey),
		Verified:    true,
		VerifiedAt:  time.Now().UTC().Format(time.RFC3339),
		AccountInfo: result.AccountInfo,
		Models:      result.Models,
	}
	s.persistCredentials()

	return result, nil
}

// ClearKey elimina la credencial guardada de un proveedor
func (s *Service) ClearKey(providerID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.credentials, providerID)
	s.persistCredentials()
}

// SaveAdminKey guarda la Admin API Key de un proveedor que la requiera para reportar uso real
// (p. ej. Anthropic). No se valida contra la red aquí: se usará en la siguiente consulta de uso.
func (s *Service) SaveAdminKey(providerID string, adminKey string) error {
	if !s.supportsAdminKey(providerID) {
		return domain.ErrAdminKeyNotSupported
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	cred := s.credentials[providerID]
	cred.ProviderID = providerID
	cred.AdminAPIKey = strings.TrimSpace(adminKey)
	s.credentials[providerID] = cred
	s.persistCredentials()
	return nil
}

// ClearAdminKey elimina la Admin API Key guardada de un proveedor, sin afectar su clave normal
func (s *Service) ClearAdminKey(providerID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cred, ok := s.credentials[providerID]
	if !ok {
		return
	}
	cred.AdminAPIKey = ""
	s.credentials[providerID] = cred
	s.persistCredentials()
}

// GetProviderUsage consulta en vivo el consumo/costo real de la cuenta de un proveedor, cuando
// su implementación soporta UsageFetcher. Providers sin esta capacidad retornan Available: false.
func (s *Service) GetProviderUsage(ctx context.Context, providerID string) (domain.ProviderUsageResult, error) {
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

// GetRawAPIKey retorna la clave completa almacenada de un proveedor para uso interno del backend
// (por ejemplo, al realizar peticiones reales de chat). Nunca debe exponerse tal cual al frontend.
func (s *Service) GetRawAPIKey(providerID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cred, ok := s.credentials[providerID]
	if !ok || cred.APIKey == "" {
		return "", false
	}
	return cred.APIKey, true
}
