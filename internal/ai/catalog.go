package ai

import (
	"embed"
	"encoding/json"
	"io/fs"
	"strings"
	"sync"

	"merlincode/internal/ai/providers/anthropic"
	"merlincode/internal/ai/providers/deepseek"
	"merlincode/internal/ai/providers/google"
	"merlincode/internal/ai/providers/openai"
	"merlincode/internal/domain"
)

//go:embed configs/*.json
var configsFS embed.FS

// KnownProviderIDs enumera los proveedores soportados, en el orden en que deben mostrarse.
var KnownProviderIDs = []string{"anthropic", "openai", "google", "deepseek"}

// defaultRegistry mapea cada proveedor a su cliente / adaptador correspondiente.
var defaultRegistry = map[string]Provider{
	"anthropic": anthropic.Client{},
	"openai":    openai.Client{},
	"google":    google.Client{},
	"deepseek":  deepseek.Client{},
}

var (
	configMu        sync.RWMutex
	providerConfigs = map[string]domain.ProviderConfig{}
)

func init() {
	loadEmbeddedConfigs()
}

func loadEmbeddedConfigs() {
	configMu.Lock()
	defer configMu.Unlock()

	files, err := fs.Glob(configsFS, "configs/*.json")
	if err != nil {
		return
	}
	for _, file := range files {
		data, err := configsFS.ReadFile(file)
		if err != nil {
			continue
		}
		var cfg domain.ProviderConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			continue
		}
		key := NormalizeProviderID(cfg.Provider)
		providerConfigs[key] = cfg
		if key == "google" {
			providerConfigs["gemini"] = cfg
		}
	}
}

// NormalizeProviderID normaliza el identificador de proveedor permitiendo alias como 'gemini' -> 'google'.
func NormalizeProviderID(providerID string) string {
	providerID = strings.TrimSpace(strings.ToLower(providerID))
	if providerID == "gemini" {
		return "google"
	}
	return providerID
}

// GetProviderConfig retorna la configuración del proveedor leída desde su archivo JSON.
func GetProviderConfig(providerID string) (domain.ProviderConfig, bool) {
	configMu.RLock()
	defer configMu.RUnlock()
	cfg, ok := providerConfigs[NormalizeProviderID(providerID)]
	return cfg, ok
}

// GetOrchestratorModel retorna el modelo orquestador por defecto para el proveedor según su JSON.
func GetOrchestratorModel(providerID string) string {
	cfg, ok := GetProviderConfig(providerID)
	if !ok {
		return ""
	}
	return cfg.OrchestratorModel
}

// DefaultModelForProvider es un alias de compatibilidad con GetOrchestratorModel.
func DefaultModelForProvider(providerID string) string {
	return GetOrchestratorModel(providerID)
}

// AllowedModelsForProvider retorna la lista de IDs de modelos configurados en el JSON para el proveedor.
func AllowedModelsForProvider(providerID string) []string {
	cfg, ok := GetProviderConfig(providerID)
	if !ok {
		return nil
	}
	res := make([]string, len(cfg.Models))
	for i, m := range cfg.Models {
		res[i] = m.ID
	}
	return res
}

// GetModelForTier retorna el modelo según el nivel de complejidad requerido ("fast", "balanced", "complex").
func GetModelForTier(providerID string, tier string) string {
	cfg, ok := GetProviderConfig(providerID)
	if !ok {
		return ""
	}
	tier = strings.TrimSpace(strings.ToLower(tier))
	for _, m := range cfg.Models {
		if strings.EqualFold(m.Tier, tier) && m.Status == "stable" {
			return m.ID
		}
	}
	return cfg.OrchestratorModel
}

// IsModelAllowedForProvider valida que un modelo pertenezca a los modelos configurados en el JSON.
func IsModelAllowedForProvider(providerID string, model string, dynamicModels []string) bool {
	model = strings.TrimSpace(model)
	if model == "" {
		return true // usará el modelo orquestador por defecto
	}
	for _, m := range AllowedModelsForProvider(providerID) {
		if strings.EqualFold(m, model) {
			return true
		}
	}
	for _, m := range dynamicModels {
		if strings.EqualFold(m, model) {
			return true
		}
	}
	return false
}

// GetProvider retorna el cliente registrado para un proveedor si existe (soporta alias).
func GetProvider(providerID string) (Provider, bool) {
	p, ok := defaultRegistry[NormalizeProviderID(providerID)]
	return p, ok
}

// GetValidator retorna el proveedor registrado (alias de compatibilidad con GetProvider).
func GetValidator(providerID string) (Provider, bool) {
	return GetProvider(providerID)
}
