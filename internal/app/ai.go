package app

import (
	"context"
	"errors"
	"time"

	"merlincode/internal/domain"
)

func sanitizeAIProviderError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return errors.New("la operación con el proveedor fue cancelada o excedió el tiempo de espera")
	case errors.Is(err, domain.ErrRuntimeNotInitialized):
		return domain.ErrRuntimeNotInitialized
	case errors.Is(err, domain.ErrUnknownAIProvider):
		return domain.ErrUnknownAIProvider
	case errors.Is(err, domain.ErrAdminKeyNotSupported):
		return domain.ErrAdminKeyNotSupported
	case errors.Is(err, domain.ErrInvalidModel):
		return domain.ErrInvalidModel
	case errors.Is(err, domain.ErrMessageTooLarge):
		return domain.ErrMessageTooLarge
	case errors.Is(err, domain.ErrRateLimited):
		return domain.ErrRateLimited
	case errors.Is(err, domain.ErrConcurrentRequestBlocked):
		return domain.ErrConcurrentRequestBlocked
	case errors.Is(err, domain.ErrCredentialMigrationFailed):
		return domain.ErrCredentialMigrationFailed
	case errors.Is(err, domain.ErrCredentialStoreUnavailable):
		return domain.ErrCredentialStoreUnavailable
	case errors.Is(err, domain.ErrProviderMetadataUnavailable):
		return domain.ErrProviderMetadataUnavailable
	default:
		return domain.ErrAIProviderOperationFailed
	}
}

func (a *App) ensureAIProviderReady() error {
	if a.aiProviderService == nil {
		return domain.ErrAIProviderOperationFailed
	}
	return sanitizeAIProviderError(a.aiProviderService.InitializationError())
}

// ListAIProviderStatuses retorna el estado (configurado/verificado/masked key) de cada proveedor de IA soportado
func (a *App) ListAIProviderStatuses() ([]domain.ProviderStatus, error) {
	if err := a.ensureAIProviderReady(); err != nil {
		return nil, err
	}
	return a.aiProviderService.ListStatuses(), nil
}

// ValidateAndSaveAIProviderKey conecta en vivo contra el proveedor indicado para confirmar que la
// clave de API es correcta y obtener datos básicos de la cuenta; solo persiste si la validación es exitosa
func (a *App) ValidateAndSaveAIProviderKey(providerID string, apiKey string) (domain.ProviderValidationResult, error) {
	if err := a.ensureAIProviderReady(); err != nil {
		return domain.ProviderValidationResult{}, err
	}
	ctx, cancel, err := a.getAppContext(20 * time.Second)
	if err != nil {
		return domain.ProviderValidationResult{}, sanitizeAIProviderError(err)
	}
	defer cancel()
	result, err := a.aiProviderService.ValidateAndSaveKey(ctx, providerID, apiKey)
	return result, sanitizeAIProviderError(err)
}

// ClearAIProviderKey elimina la credencial guardada de un proveedor de IA tanto del llavero como de metadata
func (a *App) ClearAIProviderKey(providerID string) error {
	if err := a.ensureAIProviderReady(); err != nil {
		return err
	}
	return sanitizeAIProviderError(a.aiProviderService.ClearKey(providerID))
}

// SaveAIProviderAdminKey guarda la Admin API Key de un proveedor que la requiera (p. ej. Anthropic)
// para poder consultar su consumo/costo real de organización
func (a *App) SaveAIProviderAdminKey(providerID string, adminKey string) error {
	if err := a.ensureAIProviderReady(); err != nil {
		return err
	}
	return sanitizeAIProviderError(a.aiProviderService.SaveAdminKey(providerID, adminKey))
}

// ClearAIProviderAdminKey elimina la Admin API Key guardada de un proveedor del llavero y de metadata
func (a *App) ClearAIProviderAdminKey(providerID string) error {
	if err := a.ensureAIProviderReady(); err != nil {
		return err
	}
	return sanitizeAIProviderError(a.aiProviderService.ClearAdminKey(providerID))
}

// GetAIProviderUsage consulta en vivo el consumo/costo o saldo real de la cuenta de un proveedor
func (a *App) GetAIProviderUsage(providerID string) (domain.ProviderUsageResult, error) {
	if err := a.ensureAIProviderReady(); err != nil {
		return domain.ProviderUsageResult{}, err
	}
	ctx, cancel, err := a.getAppContext(20 * time.Second)
	if err != nil {
		return domain.ProviderUsageResult{}, sanitizeAIProviderError(err)
	}
	defer cancel()
	result, err := a.aiProviderService.GetProviderUsage(ctx, providerID)
	return result, sanitizeAIProviderError(err)
}

// SendAIProviderTestMessage envía un mensaje real de prueba al proveedor indicado usando su clave
// de API estándar y el modelo seleccionado, para confirmar en la interfaz que la conexión funciona
// de punta a punta con ese modelo específico
func (a *App) SendAIProviderTestMessage(providerID string, message string, model string) (domain.TestMessageResult, error) {
	if err := a.ensureAIProviderReady(); err != nil {
		return domain.TestMessageResult{}, err
	}
	ctx, cancel, err := a.getAppContext(30 * time.Second)
	if err != nil {
		return domain.TestMessageResult{}, sanitizeAIProviderError(err)
	}
	defer cancel()
	result, err := a.aiProviderService.SendTestMessage(ctx, providerID, message, model)
	return result, sanitizeAIProviderError(err)
}
