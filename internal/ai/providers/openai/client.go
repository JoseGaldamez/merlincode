package openai

import (
	"context"
	"encoding/json"
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

type openAIChatRequest struct {
	Model     string `json:"model"`
	MaxTokens int    `json:"max_tokens"`
	Messages  []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

type openAIChatResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    any    `json:"code"`
	} `json:"error"`
}

// SendTestMessage envía un mensaje de prueba a /v1/chat/completions usando la clave de API estándar.
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
		model = "gpt-5.6-terra"
	}

	reqBody := openAIChatRequest{Model: model, MaxTokens: 256}
	reqBody.Messages = []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}{{Role: "user", Content: message}}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return domain.TestMessageResult{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", strings.NewReader(string(payload)))
	if err != nil {
		return domain.TestMessageResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := transport.Client.Do(req)
	if err != nil {
		return domain.TestMessageResult{Success: false, Message: transport.SanitizeTransportError(err, "OpenAI")}, nil
	}
	defer transport.CloseHTTPResponse(resp)

	var parsed openAIChatResponse
	if err := transport.DecodeJSONLimited(resp.Body, &parsed, transport.DefaultMaxResponseBytes); err != nil {
		return domain.TestMessageResult{Success: false, Message: transport.SanitizeTransportError(err, "OpenAI")}, nil
	}

	if resp.StatusCode != http.StatusOK {
		if parsed.Error != nil && parsed.Error.Message != "" {
			errLow := strings.ToLower(parsed.Error.Message)
			codeStr := strings.ToLower(fmt.Sprintf("%v", parsed.Error.Code))
			if strings.Contains(errLow, "quota") || strings.Contains(errLow, "billing") || codeStr == "insufficient_quota" {
				return domain.TestMessageResult{
					Success: false,
					Message: "Tu cuenta de OpenAI no tiene saldo disponible o excedió su cuota de uso (insufficient_quota). Recarga créditos en platform.openai.com/settings/organization/billing.",
				}, nil
			}
			if strings.Contains(errLow, "does not exist") || strings.Contains(errLow, "not found") || codeStr == "model_not_found" {
				return domain.TestMessageResult{
					Success: false,
					Message: fmt.Sprintf("El modelo '%s' no existe en la API de OpenAI o tu cuenta no tiene acceso a él.", model),
				}, nil
			}
			if resp.StatusCode == http.StatusTooManyRequests {
				return domain.TestMessageResult{
					Success: false,
					Message: "OpenAI limitó temporalmente las solicitudes por exceso de peticiones por minuto (Rate Limit). Intenta de nuevo más tarde.",
				}, nil
			}
		}
		return domain.TestMessageResult{Success: false, Message: transport.SafeProviderHTTPError("OpenAI", resp.StatusCode)}, nil
	}

	var text string
	if len(parsed.Choices) > 0 {
		text = parsed.Choices[0].Message.Content
	}

	return domain.TestMessageResult{
		Success:      true,
		Message:      "El modelo respondió correctamente. Este consumo debería aparecer en el reporte de uso en unos minutos.",
		ResponseText: text,
		Model:        parsed.Model,
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
