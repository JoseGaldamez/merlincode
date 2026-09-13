package aiproviders

// KnownProviderIDs enumera los proveedores soportados, en el orden en que deben mostrarse.
// Es la única lista que debe actualizarse al agregar un proveedor nuevo.
var KnownProviderIDs = []string{"anthropic", "openai", "google", "deepseek"}

// defaultRegistry mapea cada proveedor a su implementación de Validator.
// Agregar un proveedor nuevo consiste en: crear su archivo con un Validator propio,
// registrarlo aquí y añadir su ID a KnownProviderIDs; el resto del paquete no cambia.
var defaultRegistry = map[string]Validator{
	"anthropic": AnthropicValidator{},
	"openai":    OpenAIValidator{},
	"google":    GoogleValidator{},
	"deepseek":  DeepSeekValidator{},
}

// GetValidator retorna el validador registrado para un proveedor, si existe
func GetValidator(providerID string) (Validator, bool) {
	v, ok := defaultRegistry[providerID]
	return v, ok
}
