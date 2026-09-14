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

// googleGenerateContentRequest y googleGenerateContentResponse reflejan la forma mínima necesaria
// de generateContent (https://generativelanguage.googleapis.com) para enviar y leer un mensaje de prueba.
type googleGenerateContentRequest struct {
	Contents []struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"contents"`
}

type googleGenerateContentResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// SendTestMessage envía un mensaje real y mínimo a generateContent usando la misma clave de API
// estándar; Google no tiene un concepto de Admin Key separado para esta cuenta.
func (GoogleValidator) SendTestMessage(ctx context.Context, apiKey string, message string, model string) (domain.TestMessageResult, error) {
	apiKey = strings.TrimSpace(apiKey)
	message = strings.TrimSpace(message)
	model = strings.TrimSpace(model)
	if apiKey == "" {
		return domain.TestMessageResult{Success: false, Message: "Configura y verifica una clave de API primero."}, nil
	}
	if message == "" {
		message = "Responde brevemente: ¿estás funcionando correctamente?"
	}
	if model == "" {
		model = "gemini-1.5-flash"
	}

	var reqBody googleGenerateContentRequest
	reqBody.Contents = []struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	}{{Parts: []struct {
		Text string `json:"text"`
	}{{Text: message}}}}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return domain.TestMessageResult{}, err
	}

	endpoint := "https://generativelanguage.googleapis.com/v1beta/models/" + model + ":generateContent?key=" + url.QueryEscape(apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(payload)))
	if err != nil {
		return domain.TestMessageResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return domain.TestMessageResult{Success: false, Message: "No se pudo conectar con Google: " + err.Error()}, nil
	}
	defer resp.Body.Close()

	var parsed googleGenerateContentResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return domain.TestMessageResult{Success: false, Message: "No se pudo interpretar la respuesta de Google."}, nil
	}

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Google AI respondió con estado inesperado (%d).", resp.StatusCode)
		if parsed.Error != nil && parsed.Error.Message != "" {
			errMsg = parsed.Error.Message
		}
		return domain.TestMessageResult{Success: false, Message: errMsg}, nil
	}

	var text string
	if len(parsed.Candidates) > 0 && len(parsed.Candidates[0].Content.Parts) > 0 {
		text = parsed.Candidates[0].Content.Parts[0].Text
	}

	return domain.TestMessageResult{
		Success:      true,
		Message:      "El modelo respondió correctamente.",
		ResponseText: text,
		Model:        model,
	}, nil
}
