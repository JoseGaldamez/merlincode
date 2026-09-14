package anthropic

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

type anthropicStreamEvent struct {
	Type  string `json:"type"`
	Index int    `json:"index,omitempty"`
	Delta *struct {
		Type     string `json:"type"`
		Text     string `json:"text,omitempty"`
		Thinking string `json:"thinking,omitempty"`
	} `json:"delta,omitempty"`
	Message *struct {
		Usage struct {
			InputTokens int64 `json:"input_tokens"`
		} `json:"usage"`
	} `json:"message,omitempty"`
	Usage *struct {
		OutputTokens int64 `json:"output_tokens"`
	} `json:"usage,omitempty"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// StreamChat realiza una llamada en streaming vía SSE a Anthropic (/v1/messages con stream: true).
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
		return nil, errors.New("la clave de API de Anthropic no puede estar vacía")
	}
	if model == "" {
		model = "claude-sonnet-5"
	}

	type anthropicMsg struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	var reqMessages []anthropicMsg
	var systemPrompt string

	for _, m := range messages {
		if m.Role == domain.ChatRoleSystem {
			if systemPrompt != "" {
				systemPrompt += "\n\n" + m.Content
			} else {
				systemPrompt = m.Content
			}
			continue
		}
		role := "user"
		if m.Role == domain.ChatRoleAssistant {
			role = "assistant"
		}
		reqMessages = append(reqMessages, anthropicMsg{
			Role:    role,
			Content: m.Content,
		})
	}

	if len(reqMessages) == 0 {
		return nil, errors.New("no hay mensajes para enviar a Anthropic")
	}

	reqBody := struct {
		Model     string         `json:"model"`
		MaxTokens int            `json:"max_tokens"`
		System    string         `json:"system,omitempty"`
		Messages  []anthropicMsg `json:"messages"`
		Stream    bool           `json:"stream"`
	}{
		Model:     model,
		MaxTokens: 4096,
		System:    systemPrompt,
		Messages:  reqMessages,
		Stream:    true,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("error codificando solicitud a Anthropic: %w", err)
	}

	endpoint := "https://api.anthropic.com/v1/messages"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := transport.StreamClient.Do(req)
	if err != nil {
		return nil, errors.New(transport.SanitizeTransportError(err, "Anthropic"))
	}
	defer transport.CloseHTTPResponse(resp)

	if resp.StatusCode != http.StatusOK {
		var errResp anthropicStreamEvent
		_ = transport.DecodeJSONLimited(resp.Body, &errResp, transport.DefaultMaxResponseBytes)
		if errResp.Error != nil && errResp.Error.Message != "" {
			errLow := strings.ToLower(errResp.Error.Message)
			if strings.Contains(errLow, "credit") || strings.Contains(errLow, "balance") {
				return nil, errors.New("tu cuenta de Anthropic no tiene créditos disponibles (balance demasiado bajo). Recarga saldo en console.anthropic.com/settings/plans.")
			}
			if strings.Contains(errLow, "not_found") || strings.Contains(errLow, "not found") || strings.Contains(errLow, "model") {
				return nil, fmt.Errorf("el modelo '%s' no existe en Anthropic o tu cuenta no tiene acceso a él", model)
			}
		}
		return nil, errors.New(transport.SafeProviderHTTPError("Anthropic", resp.StatusCode))
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
		var event anthropicStreamEvent
		if err := json.Unmarshal([]byte(dataContent), &event); err != nil {
			continue
		}

		if event.Message != nil {
			promptTokens = event.Message.Usage.InputTokens
		}
		if event.Usage != nil {
			completionTokens = event.Usage.OutputTokens
		}

		if event.Delta != nil {
			if event.Delta.Thinking != "" {
				totalThinking.WriteString(event.Delta.Thinking)
				if onChunk != nil {
					if err := onChunk(StreamChunk{
						Type:     domain.ChunkTypeThinking,
						Thinking: event.Delta.Thinking,
					}); err != nil {
						return nil, err
					}
				}
			}
			if event.Delta.Text != "" {
				totalText.WriteString(event.Delta.Text)
				if onChunk != nil {
					if err := onChunk(StreamChunk{
						Type: domain.ChunkTypeContent,
						Text: event.Delta.Text,
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
