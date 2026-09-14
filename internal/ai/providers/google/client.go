package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"merlincode/internal/ai/transport"
	"merlincode/internal/domain"
)

// Client verifica credenciales y envía mensajes a Google Gemini mediante el encabezado x-goog-api-key.
type Client struct{}

type googleModelsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

// Validate realiza una petición real contra /v1beta/models para confirmar que la clave es válida.
func (Client) Validate(ctx context.Context, apiKey string) (domain.ProviderValidationResult, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return domain.ProviderValidationResult{Valid: false, Message: "La clave de API no puede estar vacía."}, nil
	}

	endpoint := "https://generativelanguage.googleapis.com/v1beta/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return domain.ProviderValidationResult{}, err
	}
	req.Header.Set("x-goog-api-key", apiKey)

	resp, err := transport.Client.Do(req)
	if err != nil {
		return domain.ProviderValidationResult{Valid: false, Message: transport.SanitizeTransportError(err, "Google AI")}, nil
	}
	defer transport.CloseHTTPResponse(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		var parsed googleModelsResponse
		if err := transport.DecodeJSONLimited(resp.Body, &parsed, transport.DefaultMaxResponseBytes); err != nil {
			return domain.ProviderValidationResult{Valid: false, Message: transport.SanitizeTransportError(err, "Google AI")}, nil
		}
		return domain.ProviderValidationResult{
			Valid:       true,
			Message:     "Conexión establecida correctamente con Google AI.",
			AccountInfo: "Conectado a Google AI Studio",
		}, nil
	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden:
		return domain.ProviderValidationResult{Valid: false, Message: "Clave de API inválida o revocada."}, nil
	default:
		return domain.ProviderValidationResult{Valid: false, Message: fmt.Sprintf("Google AI respondió con estado inesperado (%d).", resp.StatusCode)}, nil
	}
}

type googleGenerationConfig struct {
	MaxOutputTokens int `json:"maxOutputTokens"`
}

type googleGenerateContentRequest struct {
	Contents []struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"contents"`
	GenerationConfig *googleGenerationConfig `json:"generationConfig,omitempty"`
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

// SendTestMessage envía un mensaje de prueba a generateContent usando autenticación por encabezado.
func (Client) SendTestMessage(ctx context.Context, apiKey string, message string, model string) (domain.TestMessageResult, error) {
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
		model = "gemini-3.8-flash"
	}

	var reqBody googleGenerateContentRequest
	reqBody.Contents = []struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	}{{Parts: []struct {
		Text string `json:"text"`
	}{{Text: message}}}}
	reqBody.GenerationConfig = &googleGenerationConfig{
		MaxOutputTokens: 256,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return domain.TestMessageResult{}, err
	}

	endpoint := "https://generativelanguage.googleapis.com/v1beta/models/" + url.PathEscape(model) + ":generateContent"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(payload)))
	if err != nil {
		return domain.TestMessageResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", apiKey)

	resp, err := transport.Client.Do(req)
	if err != nil {
		return domain.TestMessageResult{Success: false, Message: transport.SanitizeTransportError(err, "Google AI")}, nil
	}
	defer transport.CloseHTTPResponse(resp)

	var parsed googleGenerateContentResponse
	if err := transport.DecodeJSONLimited(resp.Body, &parsed, transport.DefaultMaxResponseBytes); err != nil {
		return domain.TestMessageResult{Success: false, Message: transport.SanitizeTransportError(err, "Google AI")}, nil
	}

	if resp.StatusCode != http.StatusOK {
		if parsed.Error != nil && parsed.Error.Message != "" {
			errLow := strings.ToLower(parsed.Error.Message)
			if strings.Contains(errLow, "quota") || strings.Contains(errLow, "resource_exhausted") {
				return domain.TestMessageResult{
					Success: false,
					Message: "Google Gemini reporta que excediste la cuota de peticiones permitida (Resource Exhausted). Intenta en unos minutos.",
				}, nil
			}
			if strings.Contains(errLow, "not found") || strings.Contains(errLow, "model") {
				return domain.TestMessageResult{
					Success: false,
					Message: fmt.Sprintf("El modelo '%s' no existe en Google Gemini o no está disponible.", model),
				}, nil
			}
		}
		return domain.TestMessageResult{Success: false, Message: transport.SafeProviderHTTPError("Google AI", resp.StatusCode)}, nil
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
