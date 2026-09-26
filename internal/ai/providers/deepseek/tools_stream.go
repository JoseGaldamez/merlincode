package deepseek

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

// deepSeekToolCallFunction es la función invocada dentro de un tool_call de DeepSeek.
type deepSeekToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type deepSeekToolCall struct {
	ID       string                   `json:"id"`
	Type     string                   `json:"type"`
	Function deepSeekToolCallFunction `json:"function"`
}

// deepSeekChatReqMsg es el mensaje de request extendido con soporte de tool-calling.
type deepSeekChatReqMsg struct {
	Role       string             `json:"role"`
	Content    string             `json:"content,omitempty"`
	ToolCalls  []deepSeekToolCall `json:"tool_calls,omitempty"`
	ToolCallID string             `json:"tool_call_id,omitempty"`
}

type deepSeekToolFunctionDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// deepSeekTool es la traducción de domain.ToolDefinition al wire format de DeepSeek (compatible con OpenAI).
type deepSeekTool struct {
	Type     string                  `json:"type"`
	Function deepSeekToolFunctionDef `json:"function"`
}

type deepSeekToolsStreamChunkResponse struct {
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

// deepSeekToolCallAccumulator ensambla los fragmentos de argumentos de un tool_call en curso.
// Igual que OpenAI, DeepSeek fragmenta delta.tool_calls[].function.arguments como piezas de
// string por índice, sin que el JSON sea válido hasta que se ensamblan todas las piezas.
type deepSeekToolCallAccumulator struct {
	id   string
	name string
	args strings.Builder
}

// StreamChatWithTools realiza una llamada en streaming vía SSE a DeepSeek ofreciendo herramientas
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
		return nil, errors.New("la clave de API de DeepSeek no puede estar vacía")
	}
	if model == "" {
		model = "deepseek-v4-pro"
	}

	var reqMessages []deepSeekChatReqMsg
	for _, m := range messages {
		if len(m.ToolResults) > 0 {
			for _, tr := range m.ToolResults {
				reqMessages = append(reqMessages, deepSeekChatReqMsg{
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

		msg := deepSeekChatReqMsg{Role: role, Content: m.Content}
		for _, tc := range m.ToolCalls {
			msg.ToolCalls = append(msg.ToolCalls, deepSeekToolCall{
				ID:   tc.ID,
				Type: "function",
				Function: deepSeekToolCallFunction{
					Name:      tc.Name,
					Arguments: tc.Arguments,
				},
			})
		}
		reqMessages = append(reqMessages, msg)
	}

	if len(reqMessages) == 0 {
		return nil, errors.New("no hay mensajes para enviar a DeepSeek")
	}

	var reqTools []deepSeekTool
	for _, t := range tools {
		reqTools = append(reqTools, deepSeekTool{
			Type: "function",
			Function: deepSeekToolFunctionDef{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		})
	}

	reqBody := struct {
		MaxOutputTokens int                  `json:"max_tokens"`
		Model           string               `json:"model"`
		Messages        []deepSeekChatReqMsg `json:"messages"`
		Tools           []deepSeekTool       `json:"tools,omitempty"`
		Stream          bool                 `json:"stream"`
	}{
		MaxOutputTokens: 4096,
		Model:           model,
		Messages:        reqMessages,
		Tools:           reqTools,
		Stream:          true,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("error codificando solicitud a DeepSeek: %w", err)
	}

	endpoint := "https://api.deepseek.com/chat/completions"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := transport.StreamClient.Do(req)
	if err != nil {
		return nil, errors.New(transport.SanitizeTransportError(err, "DeepSeek"))
	}
	defer transport.CloseHTTPResponse(resp)

	if resp.StatusCode != http.StatusOK {
		var errResp deepSeekToolsStreamChunkResponse
		_ = transport.DecodeJSONLimited(resp.Body, &errResp, transport.DefaultMaxResponseBytes)
		if errResp.Error != nil && errResp.Error.Message != "" {
			errLow := strings.ToLower(errResp.Error.Message)
			if strings.Contains(errLow, "balance") || strings.Contains(errLow, "insufficient") {
				return nil, errors.New("tu cuenta de DeepSeek no tiene saldo suficiente. Recarga tu balance en platform.deepseek.com.")
			}
			if strings.Contains(errLow, "not found") || strings.Contains(errLow, "model") {
				return nil, fmt.Errorf("el modelo '%s' no existe en DeepSeek o no está disponible", model)
			}
		}
		return nil, errors.New(transport.SafeProviderHTTPError("DeepSeek", resp.StatusCode))
	}

	scanner := bufio.NewScanner(transport.LimitedStreamReader(resp.Body))
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var totalText strings.Builder
	var totalThinking strings.Builder
	var promptTokens int64
	var completionTokens int64
	accumulators := make(map[int]*deepSeekToolCallAccumulator)
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

		var chunk deepSeekToolsStreamChunkResponse
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
					acc = &deepSeekToolCallAccumulator{}
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
