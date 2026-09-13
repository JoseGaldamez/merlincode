package aiproviders

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"merlincode/internal/domain"
)

// GoogleValidator verifica claves de API contra el endpoint oficial de Google Generative Language (Gemini)
type GoogleValidator struct{}

type googleModelsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

func (GoogleValidator) Validate(ctx context.Context, apiKey string) (domain.ProviderValidationResult, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return domain.ProviderValidationResult{Valid: false, Message: "La clave de API no puede estar vacía."}, nil
	}

	endpoint := "https://generativelanguage.googleapis.com/v1beta/models?key=" + url.QueryEscape(apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return domain.ProviderValidationResult{}, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return domain.ProviderValidationResult{Valid: false, Message: "No se pudo conectar con Google: " + err.Error()}, nil
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusOK:
		var parsed googleModelsResponse
		_ = json.NewDecoder(resp.Body).Decode(&parsed)
		models := make([]string, 0, len(parsed.Models))
		for _, m := range parsed.Models {
			models = append(models, strings.TrimPrefix(m.Name, "models/"))
		}
		return domain.ProviderValidationResult{
			Valid:       true,
			Message:     "Conexión establecida correctamente con Google AI.",
			AccountInfo: fmt.Sprintf("%d modelos disponibles para esta cuenta", len(models)),
			Models:      models,
		}, nil
	case resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return domain.ProviderValidationResult{Valid: false, Message: "Clave de API inválida o revocada."}, nil
	default:
		return domain.ProviderValidationResult{Valid: false, Message: fmt.Sprintf("Google AI respondió con estado inesperado (%d).", resp.StatusCode)}, nil
	}
}
