package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"merlincode/internal/ai/transport"
	"merlincode/internal/domain"
)

// openAIToolCallFunction es la función invocada dentro de un tool_call de OpenAI.
type openAIToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openAIToolCall struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Function openAIToolCallFunction `json:"function"`
}

// openAIChatReqMsg es el mensaje de request extendido con soporte de tool-calling.
type openAIChatReqMsg struct {
	Role       string           `json:"role"`
	Content    string           `json:"content,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type openAIToolFunctionDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// openAITool es la traducción de domain.ToolDefinition al wire format de OpenAI.
type openAITool struct {
	Type     string                `json:"type"`
	Function openAIToolFunctionDef `json:"function"`
}

type openAIToolsStreamChunkResponse struct {
	Choices []struct {
		Delta struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
			ToolCalls        []struct {
				Index    int    `json:"index"`
				ID       string `json:"id,omitempty"`
				Function struct {
					Name      string `json:"name,omitempty"`
					Arguments string `json:"arguments,omitempty"`
				} `json:"function,omitempty"`
			} `json:"tool_calls,omitempty"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int64 `json:"prompt_tokens"`
		CompletionTokens int64 `json:"completion_tokens"`
	} `json:"usage,omitempty"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    any    `json:"code"`
	} `json:"error,omitempty"`
}

// openAIToolCallAccumulator ensambla los fragmentos de argumentos de un tool_call en curso.
// OpenAI fragmenta delta.tool_calls[].function.arguments como piezas de string por índice,
// sin que el JSON sea válido hasta que se ensamblan todas las piezas.
type openAIToolCallAccumulator struct {
	id   string
	name string
	args strings.Builder
}

// StreamChatWithTools realiza una llamada en streaming vía SSE a OpenAI ofreciendo herramientas
// al modelo, satisfaciendo la interfaz opcional ai.ToolCaller.
func (Client) StreamChatWithTools(
	ctx context.Context,
	apiKey string,
	model string,
	messages []domain.ChatMessage,
	tools []domain.ToolDefinition,
	onChunk func(chunk StreamChunk) error,
) (*domain.ChatCompletionResult, error) {
	apiKey = strings.TrimSpace(apiKey)
	model = strings.TrimSpace(model)
	if apiKey == "" {
		return nil, errors.New("la clave de API de OpenAI no puede estar vacía")
	}
	if model == "" {
		model = "gpt-5.6-terra"
	}

	var reqMessages []openAIChatReqMsg
	for _, m := range messages {
		if len(m.ToolResults) > 0 {
			for _, tr := range m.ToolResults {
				reqMessages = append(reqMessages, openAIChatReqMsg{
					Role:       "tool",
					Content:    tr.Content,
					ToolCallID: tr.ToolCallID,
				})
			}
			continue
		}

		role := string(m.Role)
		if role == "" {
			role = "user"
		}

		msg := openAIChatReqMsg{Role: role, Content: m.Content}
		for _, tc := range m.ToolCalls {
			msg.ToolCalls = append(msg.ToolCalls, openAIToolCall{
				ID:   tc.ID,
				Type: "function",
				Function: openAIToolCallFunction{
					Name:      tc.Name,
					Arguments: tc.Arguments,
				},
			})
		}
		reqMessages = append(reqMessages, msg)
	}

	if len(reqMessages) == 0 {
		return nil, errors.New("no hay mensajes para enviar a OpenAI")
	}

	var reqTools []openAITool
	for _, t := range tools {
		reqTools = append(reqTools, openAITool{
			Type: "function",
			Function: openAIToolFunctionDef{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		})
	}

	reqBody := struct {
		MaxOutputTokens int                `json:"max_completion_tokens"`
		Model           string             `json:"model"`
		Messages        []openAIChatReqMsg `json:"messages"`
		Tools           []openAITool       `json:"tools,omitempty"`
		// ReasoningEffort: /v1/chat/completions rechaza function tools en modelos de razonamiento
		// (familia gpt-5.x/gpt-6) a menos que se fije en "none" — así lo indica el propio error de
		// OpenAI ("Function tools with reasoning_effort are not supported... set reasoning_effort to 'none'").
		ReasoningEffort string `json:"reasoning_effort,omitempty"`
		Stream          bool   `json:"stream"`
		StreamOptions   *struct {
			IncludeUsage bool `json:"include_usage"`
		} `json:"stream_options,omitempty"`
	}{
		MaxOutputTokens: 4096,
		Model:           model,
		Messages:        reqMessages,
		Tools:           reqTools,
		ReasoningEffort: "none",
		Stream:          true,
		StreamOptions: &struct {
			IncludeUsage bool `json:"include_usage"`
		}{
			IncludeUsage: true,
		},
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("error codificando solicitud a OpenAI: %w", err)
	}

	endpoint := "https://api.openai.com/v1/chat/completions"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := transport.StreamClient.Do(req)
	if err != nil {
		return nil, errors.New(transport.SanitizeTransportError(err, "OpenAI"))
	}
	defer transport.CloseHTTPResponse(resp)

	if resp.StatusCode != http.StatusOK {
		var errResp openAIToolsStreamChunkResponse
		_ = transport.DecodeJSONLimited(resp.Body, &errResp, transport.DefaultMaxResponseBytes)
		if errResp.Error != nil && errResp.Error.Message != "" {
			errLow := strings.ToLower(errResp.Error.Message)
			codeStr := strings.ToLower(fmt.Sprintf("%v", errResp.Error.Code))
			if strings.Contains(errLow, "quota") || strings.Contains(errLow, "billing") || codeStr == "insufficient_quota" {
				return nil, errors.New("tu cuenta de OpenAI no tiene saldo disponible o excedió su cuota de uso (insufficient_quota). Recarga créditos en platform.openai.com/settings/organization/billing.")
			}
			if strings.Contains(errLow, "does not exist") || strings.Contains(errLow, "not found") || codeStr == "model_not_found" {
				return nil, fmt.Errorf("el modelo '%s' no existe en la API de OpenAI o tu cuenta no tiene acceso a él", model)
			}
			if resp.StatusCode == http.StatusTooManyRequests {
				return nil, errors.New("OpenAI limitó temporalmente las solicitudes por exceso de peticiones por minuto (Rate Limit). Intenta de nuevo más tarde.")
			}
			return nil, fmt.Errorf("%s (detalle OpenAI: %s)", transport.SafeProviderHTTPError("OpenAI", resp.StatusCode), errResp.Error.Message)
		}
		return nil, errors.New(transport.SafeProviderHTTPError("OpenAI", resp.StatusCode))
	}

	scanner := bufio.NewScanner(transport.LimitedStreamReader(resp.Body))
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var totalText strings.Builder
	var totalThinking strings.Builder
	var promptTokens int64
	var completionTokens int64
	accumulators := make(map[int]*openAIToolCallAccumulator)
	var toolCallOrder []int

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		if !strings.HasPrefix(line, "data:") {
			continue
		}

		dataContent := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if dataContent == "[DONE]" {
			break
		}

		var chunk openAIToolsStreamChunkResponse
		if err := json.Unmarshal([]byte(dataContent), &chunk); err != nil {
			continue
		}

		if chunk.Usage != nil {
			promptTokens = chunk.Usage.PromptTokens
			completionTokens = chunk.Usage.CompletionTokens
		}

		for _, choice := range chunk.Choices {
			if choice.Delta.ReasoningContent != "" {
				totalThinking.WriteString(choice.Delta.ReasoningContent)
				if onChunk != nil {
					if err := onChunk(StreamChunk{
						Type:     domain.ChunkTypeThinking,
						Thinking: choice.Delta.ReasoningContent,
					}); err != nil {
						return nil, err
					}
				}
			}
			if choice.Delta.Content != "" {
				totalText.WriteString(choice.Delta.Content)
				if onChunk != nil {
					if err := onChunk(StreamChunk{
						Type: domain.ChunkTypeContent,
						Text: choice.Delta.Content,
					}); err != nil {
						return nil, err
					}
				}
			}
			for _, tc := range choice.Delta.ToolCalls {
				acc, ok := accumulators[tc.Index]
				if !ok {
					acc = &openAIToolCallAccumulator{}
					accumulators[tc.Index] = acc
					toolCallOrder = append(toolCallOrder, tc.Index)
				}
				if tc.ID != "" {
					acc.id = tc.ID
				}
				if tc.Function.Name != "" {
					acc.name = tc.Function.Name
				}
				if tc.Function.Arguments != "" {
					acc.args.WriteString(tc.Function.Arguments)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, errors.New(transport.SanitizeTransportError(err, "el proveedor"))
	}

	var toolCalls []domain.ToolCall
	for _, idx := range toolCallOrder {
		acc := accumulators[idx]
		args := acc.args.String()
		if args == "" || !json.Valid([]byte(args)) {
			args = "{}"
		}
		tc := domain.ToolCall{ID: acc.id, Name: acc.name, Arguments: args}
		toolCalls = append(toolCalls, tc)
		if onChunk != nil {
			if err := onChunk(StreamChunk{Type: domain.ChunkTypeToolCall, ToolCall: &tc}); err != nil {
				return nil, err
			}
		}
	}

	return &domain.ChatCompletionResult{
		Content:          totalText.String(),
		Thinking:         totalThinking.String(),
		Model:            model,
		TokensPrompt:     promptTokens,
		TokensCompletion: completionTokens,
		ToolCalls:        toolCalls,
	}, nil
}
