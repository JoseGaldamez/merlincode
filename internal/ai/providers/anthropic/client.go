package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"merlincode/internal/ai/transport"
	"merlincode/internal/domain"
)

// Client verifica credenciales, consulta consumo y envía mensajes a Anthropic.
type Client struct{}

type anthropicModelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

type anthropicUsageReportResponse struct {
	Data []struct {
		Results []struct {
			UncachedInputTokens  int64 `json:"uncached_input_tokens"`
			CacheReadInputTokens int64 `json:"cache_read_input_tokens"`
			OutputTokens         int64 `json:"output_tokens"`
		} `json:"results"`
	} `json:"data"`
}

type anthropicCostReportResponse struct {
	Data []struct {
		Results []struct {
			Amount   string `json:"amount"`
			Currency string `json:"currency"`
		} `json:"results"`
	} `json:"data"`
}

// Validate realiza una petición real contra /v1/models para confirmar que la clave de API es válida.
func (Client) Validate(ctx context.Context, apiKey string) (domain.ProviderValidationResult, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return domain.ProviderValidationResult{Valid: false, Message: "La clave de API no puede estar vacía."}, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.anthropic.com/v1/models", nil)
	if err != nil {
		return domain.ProviderValidationResult{}, err
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := transport.Client.Do(req)
	if err != nil {
		return domain.ProviderValidationResult{Valid: false, Message: transport.SanitizeTransportError(err, "Anthropic")}, nil
	}
	defer transport.CloseHTTPResponse(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		var parsed anthropicModelsResponse
		if err := transport.DecodeJSONLimited(resp.Body, &parsed, transport.DefaultMaxResponseBytes); err != nil {
			return domain.ProviderValidationResult{Valid: false, Message: transport.SanitizeTransportError(err, "Anthropic")}, nil
		}
		return domain.ProviderValidationResult{
			Valid:       true,
			Message:     "Conexión establecida correctamente con Anthropic.",
			AccountInfo: "Conectado a Anthropic Console",
		}, nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return domain.ProviderValidationResult{Valid: false, Message: "Clave de API inválida o revocada."}, nil
	default:
		return domain.ProviderValidationResult{Valid: false, Message: fmt.Sprintf("Anthropic respondió con estado inesperado (%d).", resp.StatusCode)}, nil
	}
}

// RequiresAdminKeyForUsage indica si FetchUsage necesita una Admin API Key de organización.
func (Client) RequiresAdminKeyForUsage() bool { return true }

// FetchUsage consulta la Admin API de Anthropic para obtener el consumo real de tokens y costo del día.
func (Client) FetchUsage(ctx context.Context, _ string, adminKey string) (domain.ProviderUsageResult, error) {
	adminKey = strings.TrimSpace(adminKey)
	if adminKey == "" {
		return domain.ProviderUsageResult{
			Available: false,
			Message:   "Configura tu Admin API Key de Anthropic para consultar el consumo real de la organización.",
		}, nil
	}

	startingAt := time.Now().UTC().Format("2006-01-02") + "T00:00:00Z"

	prompt, completion, err := fetchAnthropicUsageTotals(ctx, adminKey, startingAt)
	if err != nil {
		return domain.ProviderUsageResult{Available: false, Message: transport.SanitizeTransportError(err, "Anthropic")}, nil
	}

	costUsd, _ := fetchAnthropicCostTotal(ctx, adminKey, startingAt)

	return domain.ProviderUsageResult{
		Available:        true,
		Message:          "Consumo obtenido en vivo desde la Admin API de Anthropic.",
		TokensPrompt:     prompt,
		TokensCompletion: completion,
		CostUsd:          costUsd,
	}, nil
}

func fetchAnthropicUsageTotals(ctx context.Context, adminKey string, startingAt string) (prompt int64, completion int64, err error) {
	endpoint := "https://api.anthropic.com/v1/organizations/usage_report/messages?starting_at=" +
		url.QueryEscape(startingAt) + "&bucket_width=1d"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, 0, err
	}
	req.Header.Set("x-api-key", adminKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := transport.Client.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer transport.CloseHTTPResponse(resp)

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("estado inesperado (%d)", resp.StatusCode)
	}

	var parsed anthropicUsageReportResponse
	if err := transport.DecodeJSONLimited(resp.Body, &parsed, transport.DefaultMaxResponseBytes); err != nil {
		return 0, 0, err
	}

	for _, bucket := range parsed.Data {
		for _, r := range bucket.Results {
			prompt += r.UncachedInputTokens + r.CacheReadInputTokens
			completion += r.OutputTokens
		}
	}
	return prompt, completion, nil
}

type anthropicMessagesRequest struct {
	Model     string `json:"model"`
	MaxTokens int    `json:"max_tokens"`
	Messages  []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

type anthropicMessagesResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Model string `json:"model"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// SendTestMessage envía un mensaje de prueba al modelo usando la clave de API estándar.
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
		model = "claude-sonnet-5"
	}

	reqBody := anthropicMessagesRequest{Model: model, MaxTokens: 256}
	reqBody.Messages = []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}{{Role: "user", Content: message}}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return domain.TestMessageResult{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/messages", strings.NewReader(string(payload)))
	if err != nil {
		return domain.TestMessageResult{}, err
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	resp, err := transport.Client.Do(req)
	if err != nil {
		return domain.TestMessageResult{Success: false, Message: transport.SanitizeTransportError(err, "Anthropic")}, nil
	}
	defer transport.CloseHTTPResponse(resp)

	var parsed anthropicMessagesResponse
	if err := transport.DecodeJSONLimited(resp.Body, &parsed, transport.DefaultMaxResponseBytes); err != nil {
		return domain.TestMessageResult{Success: false, Message: transport.SanitizeTransportError(err, "Anthropic")}, nil
	}

	if resp.StatusCode != http.StatusOK {
		if parsed.Error != nil && parsed.Error.Message != "" {
			errLow := strings.ToLower(parsed.Error.Message)
			if strings.Contains(errLow, "credit") || strings.Contains(errLow, "balance") {
				return domain.TestMessageResult{
					Success: false,
					Message: "Tu cuenta de Anthropic no tiene créditos disponibles (balance demasiado bajo). Recarga saldo en console.anthropic.com/settings/plans.",
				}, nil
			}
			if strings.Contains(errLow, "not_found") || strings.Contains(errLow, "not found") || strings.Contains(errLow, "model") {
				return domain.TestMessageResult{
					Success: false,
					Message: fmt.Sprintf("El modelo '%s' no existe en Anthropic o tu cuenta no tiene acceso a él.", model),
				}, nil
			}
		}
		return domain.TestMessageResult{Success: false, Message: transport.SafeProviderHTTPError("Anthropic", resp.StatusCode)}, nil
	}

	var text string
	for _, c := range parsed.Content {
		text += c.Text
	}

	return domain.TestMessageResult{
		Success:      true,
		Message:      "El modelo respondió correctamente. Este consumo debería aparecer en el reporte de uso en unos minutos.",
		ResponseText: text,
		Model:        parsed.Model,
	}, nil
}

func fetchAnthropicCostTotal(ctx context.Context, adminKey string, startingAt string) (float64, error) {
	endpoint := "https://api.anthropic.com/v1/organizations/cost_report?starting_at=" + url.QueryEscape(startingAt)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("x-api-key", adminKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := transport.Client.Do(req)
	if err != nil {
		return 0, err
	}
	defer transport.CloseHTTPResponse(resp)

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("estado inesperado (%d)", resp.StatusCode)
	}

	var parsed anthropicCostReportResponse
	if err := transport.DecodeJSONLimited(resp.Body, &parsed, transport.DefaultMaxResponseBytes); err != nil {
		return 0, err
	}

	var total float64
	for _, bucket := range parsed.Data {
		for _, r := range bucket.Results {
			if amt, err := strconv.ParseFloat(r.Amount, 64); err == nil {
				total += amt
			}
		}
	}
	return total, nil
}
