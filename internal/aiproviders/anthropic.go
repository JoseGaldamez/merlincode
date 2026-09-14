package aiproviders

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"merlincode/internal/domain"
)

// AnthropicValidator verifica claves de API contra el endpoint oficial de Anthropic
type AnthropicValidator struct{}

type anthropicModelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

// anthropicUsageReportResponse refleja la forma de la respuesta de la Admin API de Anthropic
// para el reporte de uso de mensajes (https://api.anthropic.com/v1/organizations/usage_report/messages)
type anthropicUsageReportResponse struct {
	Data []struct {
		Results []struct {
			UncachedInputTokens  int64 `json:"uncached_input_tokens"`
			CacheReadInputTokens int64 `json:"cache_read_input_tokens"`
			OutputTokens         int64 `json:"output_tokens"`
		} `json:"results"`
	} `json:"data"`
}

// anthropicCostReportResponse refleja la forma de la respuesta del reporte de costos
// (https://api.anthropic.com/v1/organizations/cost_report)
type anthropicCostReportResponse struct {
	Data []struct {
		Results []struct {
			Amount   string `json:"amount"`
			Currency string `json:"currency"`
		} `json:"results"`
	} `json:"data"`
}

func (AnthropicValidator) Validate(ctx context.Context, apiKey string) (domain.ProviderValidationResult, error) {
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

	resp, err := httpClient.Do(req)
	if err != nil {
		return domain.ProviderValidationResult{Valid: false, Message: "No se pudo conectar con Anthropic: " + err.Error()}, nil
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusOK:
		var parsed anthropicModelsResponse
		_ = json.NewDecoder(resp.Body).Decode(&parsed)
		models := make([]string, 0, len(parsed.Data))
		for _, m := range parsed.Data {
			models = append(models, m.ID)
		}
		return domain.ProviderValidationResult{
			Valid:       true,
			Message:     "Conexión establecida correctamente con Anthropic.",
			AccountInfo: fmt.Sprintf("%d modelos disponibles para esta cuenta", len(models)),
			Models:      models,
		}, nil
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return domain.ProviderValidationResult{Valid: false, Message: "Clave de API inválida o revocada."}, nil
	default:
		return domain.ProviderValidationResult{Valid: false, Message: fmt.Sprintf("Anthropic respondió con estado inesperado (%d).", resp.StatusCode)}, nil
	}
}

// RequiresAdminKeyForUsage: el reporte de uso/costo de Anthropic es a nivel de organización y
// solo es accesible con una Admin API Key, distinta de la clave de API normal de proyecto.
func (AnthropicValidator) RequiresAdminKeyForUsage() bool { return true }

// FetchUsage consulta la Admin API de Anthropic (Usage & Cost Report) para obtener el consumo
// real de tokens y el costo del día en curso de toda la organización.
func (AnthropicValidator) FetchUsage(ctx context.Context, _ string, adminKey string) (domain.ProviderUsageResult, error) {
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
		return domain.ProviderUsageResult{Available: false, Message: "No se pudo obtener el uso: " + err.Error()}, nil
	}

	// El costo es informativo adicional: si falla, igual reportamos los tokens obtenidos.
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

	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("Anthropic respondió con estado inesperado (%d)", resp.StatusCode)
	}

	var parsed anthropicUsageReportResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
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

// anthropicMessagesRequest y anthropicMessagesResponse reflejan la forma mínima necesaria de
// /v1/messages (https://api.anthropic.com/v1/messages) para enviar y leer un mensaje de prueba.
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

// SendTestMessage envía un mensaje real y mínimo a /v1/messages usando la clave de API estándar
// (la Admin Key no tiene permiso para generar respuestas, solo para consultar datos de organización).
// Si model viene vacío, usa un modelo económico por defecto.
func (AnthropicValidator) SendTestMessage(ctx context.Context, apiKey string, message string, model string) (domain.TestMessageResult, error) {
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
		model = "claude-3-5-haiku-20241022"
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

	resp, err := httpClient.Do(req)
	if err != nil {
		return domain.TestMessageResult{Success: false, Message: "No se pudo conectar con Anthropic: " + err.Error()}, nil
	}
	defer resp.Body.Close()

	var parsed anthropicMessagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return domain.TestMessageResult{Success: false, Message: "No se pudo interpretar la respuesta de Anthropic."}, nil
	}

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Anthropic respondió con estado inesperado (%d).", resp.StatusCode)
		if parsed.Error != nil && parsed.Error.Message != "" {
			errMsg = parsed.Error.Message
		}
		return domain.TestMessageResult{Success: false, Message: errMsg}, nil
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

	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("estado inesperado (%d)", resp.StatusCode)
	}

	var parsed anthropicCostReportResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
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
