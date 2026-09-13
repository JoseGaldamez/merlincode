package aiproviders

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"merlincode/internal/domain"
)

// OpenAIValidator verifica claves de API contra el endpoint oficial de OpenAI
type OpenAIValidator struct{}

type openAIModelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

// openAIUsageResponse refleja la forma de la respuesta de la Admin API de OpenAI
// para el uso de completions (https://api.openai.com/v1/organization/usage/completions)
type openAIUsageResponse struct {
	Data []struct {
		Results []struct {
			InputTokens       int64 `json:"input_tokens"`
			OutputTokens      int64 `json:"output_tokens"`
			InputCachedTokens int64 `json:"input_cached_tokens"`
		} `json:"results"`
	} `json:"data"`
}

// openAICostsResponse refleja la forma de la respuesta del reporte de costos
// (https://api.openai.com/v1/organization/costs)
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

func (OpenAIValidator) Validate(ctx context.Context, apiKey string) (domain.ProviderValidationResult, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return domain.ProviderValidationResult{Valid: false, Message: "La clave de API no puede estar vacía."}, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.openai.com/v1/models", nil)
	if err != nil {
		return domain.ProviderValidationResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return domain.ProviderValidationResult{Valid: false, Message: "No se pudo conectar con OpenAI: " + err.Error()}, nil
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusOK:
		var parsed openAIModelsResponse
		_ = json.NewDecoder(resp.Body).Decode(&parsed)
		models := make([]string, 0, len(parsed.Data))
		for _, m := range parsed.Data {
			models = append(models, m.ID)
		}
		return domain.ProviderValidationResult{
			Valid:       true,
			Message:     "Conexión establecida correctamente con OpenAI.",
			AccountInfo: fmt.Sprintf("%d modelos disponibles para esta cuenta", len(models)),
			Models:      models,
		}, nil
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return domain.ProviderValidationResult{Valid: false, Message: "Clave de API inválida o revocada."}, nil
	default:
		return domain.ProviderValidationResult{Valid: false, Message: fmt.Sprintf("OpenAI respondió con estado inesperado (%d).", resp.StatusCode)}, nil
	}
}

// RequiresAdminKeyForUsage: el uso/costo de OpenAI es a nivel de organización y solo es
// accesible con una Admin API Key, distinta de la clave de API normal de proyecto.
func (OpenAIValidator) RequiresAdminKeyForUsage() bool { return true }

// FetchUsage consulta la Admin API de OpenAI (Usage & Costs) para obtener el consumo real
// de tokens y el costo del día en curso de toda la organización.
func (OpenAIValidator) FetchUsage(ctx context.Context, _ string, adminKey string) (domain.ProviderUsageResult, error) {
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
		return domain.ProviderUsageResult{Available: false, Message: "No se pudo obtener el uso: " + err.Error()}, nil
	}

	// El costo es informativo adicional: si falla, igual reportamos los tokens obtenidos.
	costUsd, _ := fetchOpenAICostTotal(ctx, adminKey, startTime)

	return domain.ProviderUsageResult{
		Available:        true,
		Message:          "Consumo obtenido en vivo desde la Admin API de OpenAI.",
		TokensPrompt:     prompt,
		TokensCompletion: completion,
		CostUsd:          costUsd,
	}, nil
}

// openAIChatRequest y openAIChatResponse reflejan la forma mínima necesaria de
// /v1/chat/completions para enviar y leer un mensaje de prueba.
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
	} `json:"error"`
}

// SendTestMessage envía un mensaje real y mínimo a /v1/chat/completions usando la clave de API
// estándar (la Admin Key de OpenAI no tiene permiso para generar respuestas, solo para
// consultar datos de organización).
func (OpenAIValidator) SendTestMessage(ctx context.Context, apiKey string, message string, model string) (domain.TestMessageResult, error) {
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
		model = "gpt-4o-mini"
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

	resp, err := httpClient.Do(req)
	if err != nil {
		return domain.TestMessageResult{Success: false, Message: "No se pudo conectar con OpenAI: " + err.Error()}, nil
	}
	defer resp.Body.Close()

	var parsed openAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return domain.TestMessageResult{Success: false, Message: "No se pudo interpretar la respuesta de OpenAI."}, nil
	}

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("OpenAI respondió con estado inesperado (%d).", resp.StatusCode)
		if parsed.Error != nil && parsed.Error.Message != "" {
			errMsg = parsed.Error.Message
		}
		return domain.TestMessageResult{Success: false, Message: errMsg}, nil
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

	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("OpenAI respondió con estado inesperado (%d)", resp.StatusCode)
	}

	var parsed openAIUsageResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
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

	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("estado inesperado (%d)", resp.StatusCode)
	}

	var parsed openAICostsResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
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
