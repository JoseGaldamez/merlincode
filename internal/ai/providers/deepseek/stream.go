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

// StreamChunk es un alias de compatibilidad hacia domain.StreamChunk.
type StreamChunk = domain.StreamChunk

type deepSeekStreamChunkResponse struct {
	Choices []struct {
		Delta struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
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

// StreamChat realiza una llamada en streaming vía SSE a DeepSeek (/chat/completions con stream: true).
func (Client) StreamChat(
	ctx context.Context,
	apiKey string,
	model string,
	messages []domain.ChatMessage,
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

	type chatReqMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	var reqMessages []chatReqMsg
	for _, m := range messages {
		role := string(m.Role)
		if role == "" {
			role = "user"
		}
		reqMessages = append(reqMessages, chatReqMsg{
			Role:    role,
			Content: m.Content,
		})
	}

	if len(reqMessages) == 0 {
		return nil, errors.New("no hay mensajes para enviar a DeepSeek")
	}

	reqBody := struct {
		MaxOutputTokens int          `json:"max_tokens"`
		Model           string       `json:"model"`
		Messages        []chatReqMsg `json:"messages"`
		Stream          bool         `json:"stream"`
	}{
		MaxOutputTokens: 4096,
		Model:           model,
		Messages:        reqMessages,
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
		var errResp deepSeekStreamChunkResponse
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

		var chunk deepSeekStreamChunkResponse
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
		}
	}

	if err := scanner.Err(); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, errors.New(transport.SanitizeTransportError(err, "el proveedor"))
	}

	return &domain.ChatCompletionResult{
		Content:          totalText.String(),
		Thinking:         totalThinking.String(),
		Model:            model,
		TokensPrompt:     promptTokens,
		TokensCompletion: completionTokens,
	}, nil
}
