package domain

// ProviderCredential almacena de forma persistida el estado de la credencial de un proveedor de IA
type ProviderCredential struct {
	ProviderID  string   `json:"providerId"`
	APIKey      string   `json:"apiKey"`
	AdminAPIKey string   `json:"adminApiKey,omitempty"`
	Verified    bool     `json:"verified"`
	VerifiedAt  string   `json:"verifiedAt,omitempty"`
	AccountInfo string   `json:"accountInfo,omitempty"`
	Models      []string `json:"models,omitempty"`
}

// ProviderStatus es la vista segura del estado de un proveedor expuesta al frontend (sin las keys completas)
type ProviderStatus struct {
	ProviderID       string   `json:"providerId"`
	Configured       bool     `json:"configured"`
	Verified         bool     `json:"verified"`
	VerifiedAt       string   `json:"verifiedAt,omitempty"`
	AccountInfo      string   `json:"accountInfo,omitempty"`
	MaskedKey        string   `json:"maskedKey,omitempty"`
	Models           []string `json:"models,omitempty"`
	SupportsAdminKey bool     `json:"supportsAdminKey"`
	HasAdminKey      bool     `json:"hasAdminKey"`
	MaskedAdminKey   string   `json:"maskedAdminKey,omitempty"`
}

// ProviderValidationResult es el resultado de intentar validar una API key directamente contra el proveedor
type ProviderValidationResult struct {
	Valid       bool     `json:"valid"`
	Message     string   `json:"message"`
	AccountInfo string   `json:"accountInfo,omitempty"`
	Models      []string `json:"models,omitempty"`
}

// ProviderUsageResult es el consumo/costo real de la cuenta reportado en vivo por el proveedor,
// cuando su API lo expone (uso vía tokens y costo, o un saldo prepagado como en DeepSeek).
type ProviderUsageResult struct {
	Available        bool    `json:"available"`
	Message          string  `json:"message"`
	TokensPrompt     int64   `json:"tokensPrompt,omitempty"`
	TokensCompletion int64   `json:"tokensCompletion,omitempty"`
	CostUsd          float64 `json:"costUsd,omitempty"`
	BalanceText      string  `json:"balanceText,omitempty"`
}

// TestMessageResult es la respuesta real obtenida al enviar un mensaje de prueba a un proveedor
// de IA, usado para confirmar en la interfaz que la clave de API funciona de punta a punta.
type TestMessageResult struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	ResponseText string `json:"responseText,omitempty"`
	Model        string `json:"model,omitempty"`
}
