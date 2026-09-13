package aiproviders

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"merlincode/internal/domain"
)

// DeepSeekValidator verifica claves de API contra el endpoint oficial de DeepSeek.
// Usa /user/balance en vez de /models porque además de validar la clave retorna
// el saldo disponible de la cuenta, información más útil para el usuario.
type DeepSeekValidator struct{}

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

// deepSeekChatRequest y deepSeekChatResponse reflejan la forma mínima necesaria de
// /chat/completions (API compatible con OpenAI) para enviar y leer un mensaje de prueba.
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

// fetchModels obtiene la lista de modelos disponibles para la cuenta. Se hace en una petición
// separada porque /user/balance no incluye esta información; un fallo aquí no invalida la
// verificación de la clave, que ya se confirmó contra /user/balance.
func (DeepSeekValidator) fetchModels(ctx context.Context, apiKey string) []string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.deepseek.com/models", nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := httpClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		return nil
	}
	defer resp.Body.Close()

	var parsed deepSeekModelsResponse
	_ = json.NewDecoder(resp.Body).Decode(&parsed)
	models := make([]string, 0, len(parsed.Data))
	for _, m := range parsed.Data {
		models = append(models, m.ID)
	}
	return models
}

// fetchBalance consulta /user/balance y retorna el texto de saldo junto con el status HTTP crudo
// para que tanto Validate como FetchUsage puedan interpretarlo según su propio contexto.
func (DeepSeekValidator) fetchBalance(ctx context.Context, apiKey string) (accountInfo string, statusCode int, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.deepseek.com/user/balance", nil)
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", resp.StatusCode, nil
	}

	var parsed deepSeekBalanceResponse
	_ = json.NewDecoder(resp.Body).Decode(&parsed)
	accountInfo = "Cuenta verificada correctamente"
	if len(parsed.BalanceInfo) > 0 {
		b := parsed.BalanceInfo[0]
		accountInfo = fmt.Sprintf("Saldo disponible: %s %s", b.TotalBalance, b.Currency)
	}
	return accountInfo, resp.StatusCode, nil
}

func (d DeepSeekValidator) Validate(ctx context.Context, apiKey string) (domain.ProviderValidationResult, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return domain.ProviderValidationResult{Valid: false, Message: "La clave de API no puede estar vacía."}, nil
	}

	accountInfo, statusCode, err := d.fetchBalance(ctx, apiKey)
	if err != nil {
		return domain.ProviderValidationResult{Valid: false, Message: "No se pudo conectar con DeepSeek: " + err.Error()}, nil
	}

	switch {
	case statusCode == http.StatusOK:
		return domain.ProviderValidationResult{
			Valid:       true,
			Message:     "Conexión establecida correctamente con DeepSeek.",
			AccountInfo: accountInfo,
			Models:      d.fetchModels(ctx, apiKey),
		}, nil
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return domain.ProviderValidationResult{Valid: false, Message: "Clave de API inválida o revocada."}, nil
	default:
		return domain.ProviderValidationResult{Valid: false, Message: fmt.Sprintf("DeepSeek respondió con estado inesperado (%d).", statusCode)}, nil
	}
}

// RequiresAdminKeyForUsage: DeepSeek expone su saldo con la misma clave de API estándar.
func (DeepSeekValidator) RequiresAdminKeyForUsage() bool { return false }

// FetchUsage reutiliza /user/balance para reportar el saldo prepagado real de la cuenta.
func (d DeepSeekValidator) FetchUsage(ctx context.Context, apiKey string, _ string) (domain.ProviderUsageResult, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return domain.ProviderUsageResult{Available: false, Message: "Configura y verifica una clave de API primero."}, nil
	}

	accountInfo, statusCode, err := d.fetchBalance(ctx, apiKey)
	if err != nil {
		return domain.ProviderUsageResult{Available: false, Message: "No se pudo conectar con DeepSeek: " + err.Error()}, nil
	}
	if statusCode != http.StatusOK {
		return domain.ProviderUsageResult{Available: false, Message: fmt.Sprintf("DeepSeek respondió con estado inesperado (%d).", statusCode)}, nil
	}

	return domain.ProviderUsageResult{Available: true, Message: "Saldo obtenido en vivo desde DeepSeek.", BalanceText: accountInfo}, nil
}

// SendTestMessage envía un mensaje real y mínimo a /chat/completions usando la clave de API
// estándar (DeepSeek no tiene un concepto de Admin Key separado; una única clave sirve tanto
// para consultar el saldo como para generar respuestas del modelo).
func (DeepSeekValidator) SendTestMessage(ctx context.Context, apiKey string, message string, model string) (domain.TestMessageResult, error) {
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
		model = "deepseek-chat"
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

	resp, err := httpClient.Do(req)
	if err != nil {
		return domain.TestMessageResult{Success: false, Message: "No se pudo conectar con DeepSeek: " + err.Error()}, nil
	}
	defer resp.Body.Close()

	var parsed deepSeekChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return domain.TestMessageResult{Success: false, Message: "No se pudo interpretar la respuesta de DeepSeek."}, nil
	}

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("DeepSeek respondió con estado inesperado (%d).", resp.StatusCode)
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
		Message:      "El modelo respondió correctamente.",
		ResponseText: text,
		Model:        parsed.Model,
	}, nil
}
