// Package aiproviders implementa la validación desacoplada de credenciales contra cada
// proveedor oficial de IA. Cada proveedor vive en su propio archivo e implementa la interfaz
// Validator; agregar un proveedor nuevo no requiere modificar el resto del paquete.
package aiproviders

import (
	"context"
	"net/http"
	"time"

	"merlincode/internal/domain"
)

// Validator define el contrato que cada proveedor de IA debe implementar para verificar
// que una API key es válida y, de ser posible, obtener información básica de la cuenta.
type Validator interface {
	// Validate realiza una petición real y mínima contra el proveedor para confirmar
	// que la clave es válida, retornando información de la cuenta cuando está disponible.
	Validate(ctx context.Context, apiKey string) (domain.ProviderValidationResult, error)
}

// UsageFetcher es un contrato opcional: los proveedores que puedan reportar consumo o
// facturación real de la cuenta lo implementan además de Validator. No implementarlo
// simplemente significa que ese proveedor no expone esta información vía API con clave estándar.
type UsageFetcher interface {
	// RequiresAdminKeyForUsage indica si FetchUsage necesita una clave de administrador
	// distinta a la clave de API normal (p. ej. Anthropic exige una Admin API Key de organización).
	RequiresAdminKeyForUsage() bool

	// FetchUsage obtiene el consumo/costo real de la cuenta. adminKey puede venir vacío
	// cuando RequiresAdminKeyForUsage() es false.
	FetchUsage(ctx context.Context, apiKey string, adminKey string) (domain.ProviderUsageResult, error)
}

// httpClient es compartido por todos los validadores con un timeout conservador
// para evitar que la UI quede bloqueada esperando un proveedor caído o lento.
var httpClient = &http.Client{
	Timeout: 15 * time.Second,
}
