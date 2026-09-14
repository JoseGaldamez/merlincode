package ai

import (
	"context"

	"merlincode/internal/domain"
)

// Provider define el contrato que cada proveedor de IA debe implementar para verificar
// que una API key es válida y, de ser posible, obtener información básica de la cuenta.
type Provider interface {
	Validate(ctx context.Context, apiKey string) (domain.ProviderValidationResult, error)
}

// Validator es un alias de compatibilidad que mapea al contrato Provider.
type Validator = Provider

// UsageFetcher es un contrato opcional para consultar consumo o facturación real de la cuenta.
type UsageFetcher interface {
	RequiresAdminKeyForUsage() bool
	FetchUsage(ctx context.Context, apiKey string, adminKey string) (domain.ProviderUsageResult, error)
}

// MessageTester es un contrato opcional para enviar un mensaje de prueba real desde la interfaz.
type MessageTester interface {
	SendTestMessage(ctx context.Context, apiKey string, message string, model string) (domain.TestMessageResult, error)
}
