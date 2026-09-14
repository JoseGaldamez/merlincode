package deepseek

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"merlincode/internal/ai/transport"
	"merlincode/internal/domain"
)

// Client verifica credenciales, consulta saldo y envía mensajes a DeepSeek.
type Client struct{}

type deepSeekBalanceResponse struct {
	IsAvailable bool `json:"is_available"`
	BalanceInfo []struct {
		Currency     string `json:"currency"`
		TotalBalance string `json:"total_balance"`
	} `json:"balance_infos"`
}

type deepSeekModelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

type deepSeekChatRequest struct {
	Model     string `json:"model"`
	MaxTokens int    `json:"max_tokens"`
	Messages  []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

type deepSeekChatResponse struct {
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

func (Client) fetchModels(ctx context.Context, apiKey string) []string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.deepseek.com/models", nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := transport.Client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		transport.CloseHTTPResponse(resp)
		return nil
	}
	defer transport.CloseHTTPResponse(resp)

	var parsed deepSeekModelsResponse
	if err := transport.DecodeJSONLimited(resp.Body, &parsed, transport.DefaultMaxResponseBytes); err != nil {
		return nil
	}
	models := make([]string, 0, len(parsed.Data))
	for _, m := range parsed.Data {
		models = append(models, m.ID)
	}
	return models
}

func (Client) fetchBalance(ctx context.Context, apiKey string) (accountInfo string, statusCode int, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.deepseek.com/user/balance", nil)
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := transport.Client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer transport.CloseHTTPResponse(resp)

	if resp.StatusCode != http.StatusOK {
		return "", resp.StatusCode, nil
	}

	var parsed deepSeekBalanceResponse
	if err := transport.DecodeJSONLimited(resp.Body, &parsed, transport.DefaultMaxResponseBytes); err != nil {
		return "", resp.StatusCode, err
	}
	accountInfo = "Cuenta verificada correctamente"
	if len(parsed.BalanceInfo) > 0 {
		b := parsed.BalanceInfo[0]
		accountInfo = fmt.Sprintf("Saldo disponible: %s %s", b.TotalBalance, b.Currency)
	}
	return accountInfo, resp.StatusCode, nil
}

// Validate realiza una comprobación real contra /user/balance para confirmar la clave y obtener el saldo.
func (c Client) Validate(ctx context.Context, apiKey string) (domain.ProviderValidationResult, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return domain.ProviderValidationResult{Valid: false, Message: "La clave de API no puede estar vacía."}, nil
	}

	accountInfo, statusCode, err := c.fetchBalance(ctx, apiKey)
	if err != nil {
		return domain.ProviderValidationResult{Valid: false, Message: transport.SanitizeTransportError(err, "DeepSeek")}, nil
	}

	switch statusCode {
	case http.StatusOK:
		return domain.ProviderValidationResult{
			Valid:       true,
			Message:     "Conexión establecida correctamente con DeepSeek.",
			AccountInfo: accountInfo,
		}, nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return domain.ProviderValidationResult{Valid: false, Message: "Clave de API inválida o revocada."}, nil
	default:
		return domain.ProviderValidationResult{Valid: false, Message: fmt.Sprintf("DeepSeek respondió con estado inesperado (%d).", statusCode)}, nil
	}
}

// RequiresAdminKeyForUsage indica que DeepSeek no requiere Admin Key para consultar saldo.
func (Client) RequiresAdminKeyForUsage() bool { return false }

// FetchUsage reutiliza /user/balance para reportar el saldo prepagado disponible.
func (c Client) FetchUsage(ctx context.Context, apiKey string, _ string) (domain.ProviderUsageResult, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return domain.ProviderUsageResult{Available: false, Message: "Configura y verifica una clave de API primero."}, nil
	}

	accountInfo, statusCode, err := c.fetchBalance(ctx, apiKey)
	if err != nil {
		return domain.ProviderUsageResult{Available: false, Message: transport.SanitizeTransportError(err, "DeepSeek")}, nil
	}
	if statusCode != http.StatusOK {
		return domain.ProviderUsageResult{Available: false, Message: fmt.Sprintf("DeepSeek respondió con estado inesperado (%d).", statusCode)}, nil
	}

	return domain.ProviderUsageResult{Available: true, Message: "Saldo obtenido en vivo desde DeepSeek.", BalanceText: accountInfo}, nil
}

// SendTestMessage envía un mensaje de prueba a /chat/completions usando la clave de API estándar.
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
		model = "deepseek-v4-pro"
	}

	reqBody := deepSeekChatRequest{Model: model, MaxTokens: 256}
	reqBody.Messages = []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}{{Role: "user", Content: message}}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return domain.TestMessageResult{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.deepseek.com/chat/completions", strings.NewReader(string(payload)))
	if err != nil {
		return domain.TestMessageResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := transport.Client.Do(req)
	if err != nil {
		return domain.TestMessageResult{Success: false, Message: transport.SanitizeTransportError(err, "DeepSeek")}, nil
	}
	defer transport.CloseHTTPResponse(resp)

	var parsed deepSeekChatResponse
	if err := transport.DecodeJSONLimited(resp.Body, &parsed, transport.DefaultMaxResponseBytes); err != nil {
		return domain.TestMessageResult{Success: false, Message: transport.SanitizeTransportError(err, "DeepSeek")}, nil
	}

	if resp.StatusCode != http.StatusOK {
		if parsed.Error != nil && parsed.Error.Message != "" {
			errLow := strings.ToLower(parsed.Error.Message)
			if strings.Contains(errLow, "balance") || strings.Contains(errLow, "insufficient") {
				return domain.TestMessageResult{
					Success: false,
					Message: "Tu cuenta de DeepSeek no tiene saldo suficiente. Recarga tu balance en platform.deepseek.com.",
				}, nil
			}
			if strings.Contains(errLow, "not found") || strings.Contains(errLow, "model") {
				return domain.TestMessageResult{
					Success: false,
					Message: fmt.Sprintf("El modelo '%s' no existe en DeepSeek o no está disponible.", model),
				}, nil
			}
		}
		return domain.TestMessageResult{Success: false, Message: transport.SafeProviderHTTPError("DeepSeek", resp.StatusCode)}, nil
	}

	var text string
	if len(parsed.Choices) > 0 {
		text = parsed.Choices[0].Message.Content
	}

	return domain.TestMessageResult{
		Success:      true,
		Message:      "El modelo respondió correctamente.",
		ResponseText: text,
		Model:        parsed.Model,
	}, nil
}
