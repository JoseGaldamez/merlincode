package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"merlincode/internal/ai/transport"
	"merlincode/internal/domain"
)

const responsesEndpoint = "https://api.openai.com/v1/responses"

type responsesRequest struct {
	Model           string            `json:"model"`
	Input           []any             `json:"input"`
	Tools           []responseTool    `json:"tools,omitempty"`
	Reasoning       responseReasoning `json:"reasoning"`
	MaxOutputTokens int               `json:"max_output_tokens"`
	Store           bool              `json:"store"`
	Stream          bool              `json:"stream,omitempty"`
}

type responseMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseTool struct {
	Type        string         `json:"type"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
	Strict      bool           `json:"strict"`
}

type responseReasoning struct {
	Effort string `json:"effort"`
}

func reasoningForModel(model string) responseReasoning {
	if model == "gpt-6-luna" {
		return responseReasoning{Effort: "none"}
	}
	return responseReasoning{Effort: "low"}
}

type responseFunctionCall struct {
	Type      string `json:"type"`
	CallID    string `json:"call_id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type responseFunctionOutput struct {
	Type   string `json:"type"`
	CallID string `json:"call_id"`
	Output string `json:"output"`
}

type responseError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type responseOutputItem struct {
	responseFunctionCall
	Status  string `json:"status"`
	Content []struct {
		Type    string `json:"type"`
		Text    string `json:"text"`
		Refusal string `json:"refusal"`
	} `json:"content"`
	Summary []struct {
		Text string `json:"text"`
	} `json:"summary"`
}

type openAIResponse struct {
	Model  string          `json:"model"`
	Status string          `json:"status"`
	Output json.RawMessage `json:"output"`
	Usage  *struct {
		InputTokens  int64 `json:"input_tokens"`
		OutputTokens int64 `json:"output_tokens"`
		InputDetails struct {
			Cached       int64 `json:"cached_tokens"`
			CacheWrites  int64 `json:"cache_write_tokens"`
			WriteDetails struct {
				Short int64 `json:"ttl_30m_tokens"`
				Long  int64 `json:"ttl_12h_tokens"`
			} `json:"cache_write_tokens_details"`
		} `json:"input_tokens_details"`
	} `json:"usage"`
	ServiceTier       string         `json:"service_tier"`
	Error             *responseError `json:"error"`
	IncompleteDetails *struct {
		Reason string `json:"reason"`
	} `json:"incomplete_details"`
}

func responsesInput(messages []domain.ChatMessage) ([]any, error) {
	var input []any
	for _, message := range messages {
		if message.Role == domain.ChatRoleAssistant && len(message.ProviderContent) > 0 {
			// Replay the original ordered output, including encrypted reasoning and call IDs.
			var items []json.RawMessage
			if err := json.Unmarshal(message.ProviderContent, &items); err != nil || len(items) == 0 {
				return nil, errors.New("no se pudo recuperar el turno anterior de OpenAI")
			}
			for _, item := range items {
				input = append(input, item)
			}
			continue
		}
		if len(message.ToolResults) > 0 {
			for _, result := range message.ToolResults {
				input = append(input, responseFunctionOutput{
					Type: "function_call_output", CallID: result.ToolCallID, Output: result.Content,
				})
			}
			continue
		}
		role := string(message.Role)
		if role == "" {
			role = "user"
		}
		if message.Content != "" || len(message.ToolCalls) == 0 {
			input = append(input, responseMessage{Role: role, Content: message.Content})
		}
		for _, call := range message.ToolCalls {
			input = append(input, responseFunctionCall{
				Type: "function_call", CallID: call.ID, Name: call.Name, Arguments: call.Arguments,
			})
		}
	}
	if len(input) == 0 {
		return nil, errors.New("no hay mensajes para enviar a OpenAI")
	}
	return input, nil
}

func newResponsesRequest(ctx context.Context, apiKey string, payload responsesRequest) (*http.Request, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.New("no se pudo codificar la solicitud a OpenAI")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, responsesEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	if payload.Stream {
		req.Header.Set("Accept", "text/event-stream")
	}
	return req, nil
}

func responseFailure(status int, details *responseError) error {
	if details != nil {
		message := strings.ToLower(details.Message)
		switch {
		case details.Code == "insufficient_quota" || strings.Contains(message, "quota") || strings.Contains(message, "billing"):
			return errors.New("tu cuenta de OpenAI no tiene saldo disponible o excedió su cuota de uso (insufficient_quota). Recarga créditos en platform.openai.com/settings/organization/billing")
		case details.Code == "model_not_found" || strings.Contains(message, "does not exist"):
			return errors.New("el modelo no está disponible en la API de OpenAI o tu cuenta no tiene acceso")
		case details.Code == "rate_limit_exceeded":
			status = http.StatusTooManyRequests
		}
	}
	if status == 0 {
		return errors.New("OpenAI no pudo completar la respuesta")
	}
	return errors.New(transport.SafeProviderHTTPError("OpenAI", status))
}

func (response *openAIResponse) completion(requestedModel string) (*domain.ChatCompletionResult, error) {
	result := &domain.ChatCompletionResult{Model: response.Model}
	if result.Model == "" {
		result.Model = requestedModel
	}
	var meteredResult *domain.ChatCompletionResult
	if usage := response.Usage; usage != nil {
		result.TokensPrompt = usage.InputTokens
		result.TokensCompletion = usage.OutputTokens
		details := usage.InputDetails
		result.Usage = &domain.ModelUsage{
			CacheRead: details.Cached, CacheWriteLong: details.WriteDetails.Long,
			CacheWriteShort: max(details.WriteDetails.Short, details.CacheWrites-details.WriteDetails.Long),
			ServiceTier:     response.ServiceTier,
		}
		meteredResult = result
	}
	if response.Status == "incomplete" {
		if response.IncompleteDetails != nil && response.IncompleteDetails.Reason == "max_output_tokens" {
			return meteredResult, errors.New("OpenAI alcanzó el límite de tokens antes de completar la respuesta")
		}
		return meteredResult, errors.New("OpenAI devolvió una respuesta incompleta")
	}
	if response.Status != "completed" {
		return meteredResult, responseFailure(0, response.Error)
	}
	var items []responseOutputItem
	if err := json.Unmarshal(response.Output, &items); err != nil {
		return meteredResult, errors.New("OpenAI devolvió un formato de respuesta inesperado")
	}

	var text, thinking strings.Builder
	var toolCalls []domain.ToolCall
	seenCalls := make(map[string]struct{})
	for _, item := range items {
		switch item.Type {
		case "message":
			for _, part := range item.Content {
				switch part.Type {
				case "output_text":
					text.WriteString(part.Text)
				case "refusal":
					text.WriteString(part.Refusal)
				}
			}
		case "reasoning":
			for _, part := range item.Summary {
				thinking.WriteString(part.Text)
			}
		case "function_call":
			args := strings.TrimSpace(item.Arguments)
			if item.CallID == "" || item.Name == "" || (item.Status != "" && item.Status != "completed") ||
				!strings.HasPrefix(args, "{") || !json.Valid([]byte(args)) {
				return meteredResult, errors.New("OpenAI devolvió una llamada de herramienta inválida")
			}
			if _, exists := seenCalls[item.CallID]; exists {
				return meteredResult, errors.New("OpenAI devolvió una llamada de herramienta duplicada")
			}
			seenCalls[item.CallID] = struct{}{}
			toolCalls = append(toolCalls, domain.ToolCall{ID: item.CallID, Name: item.Name, Arguments: item.Arguments})
		}
	}
	if text.Len() == 0 && len(toolCalls) == 0 {
		return meteredResult, errors.New("OpenAI devolvió una respuesta sin texto ni herramientas")
	}

	result.Content = text.String()
	result.Thinking = thinking.String()
	result.ToolCalls = toolCalls
	if len(toolCalls) > 0 {
		result.ProviderContent = response.Output
	}
	return result, nil
}
