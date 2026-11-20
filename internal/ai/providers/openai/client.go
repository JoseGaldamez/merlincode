package openai

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"merlincode/internal/ai/transport"
	"merlincode/internal/domain"
)

// Client verifica credenciales, consulta consumo y envía mensajes a OpenAI.
type Client struct{}

const defaultModel = "gpt-6.2-sol"

type openAIModelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

type openAIUsageResponse struct {
	Data []struct {
		Results []struct {
			InputTokens       int64 `json:"input_tokens"`
			OutputTokens      int64 `json:"output_tokens"`
			InputCachedTokens int64 `json:"input_cached_tokens"`
		} `json:"results"`
	} `json:"data"`
}

type openAICostsResponse struct {
	Data []struct {
		Results []struct {
			Amount struct {
				Value    float64 `json:"value"`
				Currency string  `json:"currency"`
			} `json:"amount"`
		} `json:"results"`
	} `json:"data"`
}

// Validate realiza una petición real contra /v1/models para confirmar que la clave de API es válida.
func (Client) Validate(ctx context.Context, apiKey string) (domain.ProviderValidationResult, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return domain.ProviderValidationResult{Valid: false, Message: "La clave de API no puede estar vacía."}, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.openai.com/v1/models", nil)
	if err != nil {
		return domain.ProviderValidationResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := transport.Client.Do(req)
	if err != nil {
		return domain.ProviderValidationResult{Valid: false, Message: transport.SanitizeTransportError(err, "OpenAI")}, nil
	}
	defer transport.CloseHTTPResponse(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		var parsed openAIModelsResponse
		if err := transport.DecodeJSONLimited(resp.Body, &parsed, transport.DefaultMaxResponseBytes); err != nil {
			return domain.ProviderValidationResult{Valid: false, Message: transport.SanitizeTransportError(err, "OpenAI")}, nil
		}
		return domain.ProviderValidationResult{
			Valid:       true,
			Message:     "Conexión establecida correctamente con OpenAI.",
			AccountInfo: "Conectado a OpenAI Platform",
		}, nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return domain.ProviderValidationResult{Valid: false, Message: "Clave de API inválida o revocada."}, nil
	default:
		return domain.ProviderValidationResult{Valid: false, Message: fmt.Sprintf("OpenAI respondió con estado inesperado (%d).", resp.StatusCode)}, nil
	}
}

// RequiresAdminKeyForUsage indica si FetchUsage necesita una Admin API Key de organización.
func (Client) RequiresAdminKeyForUsage() bool { return true }

// FetchUsage consulta la Admin API de OpenAI para obtener el consumo real de tokens y costo del día.
func (Client) FetchUsage(ctx context.Context, _ string, adminKey string) (domain.ProviderUsageResult, error) {
	adminKey = strings.TrimSpace(adminKey)
	if adminKey == "" {
		return domain.ProviderUsageResult{
			Available: false,
			Message:   "Configura tu Admin API Key de OpenAI para consultar el consumo real de la organización.",
		}, nil
	}

	startTime := time.Now().UTC().Truncate(24 * time.Hour).Unix()

	prompt, completion, err := fetchOpenAIUsageTotals(ctx, adminKey, startTime)
	if err != nil {
		return domain.ProviderUsageResult{Available: false, Message: transport.SanitizeTransportError(err, "OpenAI")}, nil
	}

	costUsd, _ := fetchOpenAICostTotal(ctx, adminKey, startTime)

	return domain.ProviderUsageResult{
		Available:        true,
		Message:          "Consumo obtenido en vivo desde la Admin API de OpenAI.",
		TokensPrompt:     prompt,
		TokensCompletion: completion,
		CostUsd:          costUsd,
	}, nil
}

// SendTestMessage prueba el mismo modelo y endpoint que usa el chat.
func (Client) SendTestMessage(ctx context.Context, apiKey, message, model string) (domain.TestMessageResult, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return domain.TestMessageResult{Success: false, Message: "Configura y verifica una clave de API primero."}, nil
	}
	message = strings.TrimSpace(message)
	if message == "" {
		message = "Responde brevemente: ¿estás funcionando correctamente?"
	}
	model = strings.TrimSpace(model)
	if model == "" {
		model = defaultModel
	}
	req, err := newResponsesRequest(ctx, apiKey, responsesRequest{
		Model: model, Input: []any{responseMessage{Role: "user", Content: message}},
		Reasoning: reasoningForModel(model), MaxOutputTokens: 256, Store: false,
	})
	if err != nil {
		return domain.TestMessageResult{}, err
	}

	resp, err := transport.Client.Do(req)
	if err != nil {
		return domain.TestMessageResult{Success: false, Message: transport.SanitizeTransportError(err, "OpenAI")}, nil
	}
	defer transport.CloseHTTPResponse(resp)

	var parsed openAIResponse
	decodeErr := transport.DecodeJSONLimited(resp.Body, &parsed, transport.DefaultMaxResponseBytes)
	if resp.StatusCode != http.StatusOK {
		return domain.TestMessageResult{Success: false, Message: responseFailure(resp.StatusCode, parsed.Error).Error()}, nil
	}
	if decodeErr != nil {
		return domain.TestMessageResult{Success: false, Message: transport.SanitizeTransportError(decodeErr, "OpenAI")}, nil
	}
	result, err := parsed.completion(model)
	if err != nil {
		return domain.TestMessageResult{Success: false, Message: err.Error()}, nil
	}

	return domain.TestMessageResult{
		Success:      true,
		Message:      "El modelo respondió correctamente. Este consumo debería aparecer en el reporte de uso en unos minutos.",
		ResponseText: result.Content,
		Model:        result.Model,
	}, nil
}

func fetchOpenAIUsageTotals(ctx context.Context, adminKey string, startTime int64) (prompt int64, completion int64, err error) {
	endpoint := "https://api.openai.com/v1/organization/usage/completions?start_time=" +
		strconv.FormatInt(startTime, 10) + "&bucket_width=1d&limit=1"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+adminKey)

	resp, err := transport.Client.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer transport.CloseHTTPResponse(resp)

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("estado inesperado (%d)", resp.StatusCode)
	}

	var parsed openAIUsageResponse
	if err := transport.DecodeJSONLimited(resp.Body, &parsed, transport.DefaultMaxResponseBytes); err != nil {
		return 0, 0, err
	}

	for _, bucket := range parsed.Data {
		for _, r := range bucket.Results {
			prompt += r.InputTokens + r.InputCachedTokens
			completion += r.OutputTokens
		}
	}
	return prompt, completion, nil
}

func fetchOpenAICostTotal(ctx context.Context, adminKey string, startTime int64) (float64, error) {
	endpoint := "https://api.openai.com/v1/organization/costs?start_time=" + strconv.FormatInt(startTime, 10) + "&limit=1"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+adminKey)

	resp, err := transport.Client.Do(req)
	if err != nil {
		return 0, err
	}
	defer transport.CloseHTTPResponse(resp)

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("estado inesperado (%d)", resp.StatusCode)
	}

	var parsed openAICostsResponse
	if err := transport.DecodeJSONLimited(resp.Body, &parsed, transport.DefaultMaxResponseBytes); err != nil {
		return 0, err
	}

	var total float64
	for _, bucket := range parsed.Data {
		for _, r := range bucket.Results {
			total += r.Amount.Value
		}
	}
	return total, nil
}
