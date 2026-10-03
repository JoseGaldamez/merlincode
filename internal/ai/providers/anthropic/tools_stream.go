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

// anthropicContentBlock representa un bloque de contenido dentro de un mensaje: texto plano,
// una invocación de herramienta (tool_use) o el resultado de una herramienta (tool_result).
type anthropicContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   string          `json:"content,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
}

type anthropicToolsMsg struct {
	Role    string                  `json:"role"`
	Content []anthropicContentBlock `json:"content"`
}

// anthropicTool es la traducción de domain.ToolDefinition al wire format de Anthropic.
type anthropicTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

type anthropicToolsStreamEvent struct {
	Type         string `json:"type"`
	Index        int    `json:"index"`
	ContentBlock *struct {
		Type string `json:"type"`
		ID   string `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"content_block,omitempty"`
	Delta *struct {
		Type        string `json:"type"`
		Text        string `json:"text,omitempty"`
		Thinking    string `json:"thinking,omitempty"`
		PartialJSON string `json:"partial_json,omitempty"`
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

// pendingToolUse acumula los fragmentos input_json_delta de un bloque tool_use en curso.
type pendingToolUse struct {
	id      string
	name    string
	builder strings.Builder
}

// StreamChatWithTools realiza una llamada en streaming vía SSE a Anthropic ofreciendo herramientas
// al modelo (tool_use / tool_result), satisfaciendo la interfaz opcional ai.ToolCaller.
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
		return nil, errors.New("la clave de API de Anthropic no puede estar vacía")
	}
	if model == "" {
		model = "claude-sonnet-5"
	}

	var reqMessages []anthropicToolsMsg
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

		var blocks []anthropicContentBlock
		if len(m.ToolResults) > 0 {
			for _, tr := range m.ToolResults {
				blocks = append(blocks, anthropicContentBlock{
					Type:      "tool_result",
					ToolUseID: tr.ToolCallID,
					Content:   tr.Content,
					IsError:   tr.IsError,
				})
			}
		} else {
			if m.Content != "" {
				blocks = append(blocks, anthropicContentBlock{Type: "text", Text: m.Content})
			}
			for _, tc := range m.ToolCalls {
				blocks = append(blocks, anthropicContentBlock{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Name,
					Input: json.RawMessage(tc.Arguments),
				})
			}
		}

		if len(blocks) == 0 {
			continue
		}
		reqMessages = append(reqMessages, anthropicToolsMsg{Role: role, Content: blocks})
	}

	if len(reqMessages) == 0 {
		return nil, errors.New("no hay mensajes para enviar a Anthropic")
	}

	var reqTools []anthropicTool
	for _, t := range tools {
		reqTools = append(reqTools, anthropicTool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		})
	}

	reqBody := struct {
		Model     string              `json:"model"`
		MaxTokens int                 `json:"max_tokens"`
		System    string              `json:"system,omitempty"`
		Messages  []anthropicToolsMsg `json:"messages"`
		Tools     []anthropicTool     `json:"tools,omitempty"`
		Stream    bool                `json:"stream"`
	}{
		Model:     model,
		MaxTokens: 4096,
		System:    systemPrompt,
		Messages:  reqMessages,
		Tools:     reqTools,
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
		var errResp anthropicToolsStreamEvent
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
	pendingTools := make(map[int]*pendingToolUse)
	var toolCalls []domain.ToolCall

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		dataContent := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		var event anthropicToolsStreamEvent
		if err := json.Unmarshal([]byte(dataContent), &event); err != nil {
			continue
		}

		if event.Message != nil {
			promptTokens = event.Message.Usage.InputTokens
		}
		if event.Usage != nil {
			completionTokens = event.Usage.OutputTokens
		}

		switch event.Type {
		case "content_block_start":
			if event.ContentBlock != nil && event.ContentBlock.Type == "tool_use" {
				pendingTools[event.Index] = &pendingToolUse{id: event.ContentBlock.ID, name: event.ContentBlock.Name}
			}
		case "content_block_delta":
			if event.Delta == nil {
				continue
			}
			if event.Delta.Thinking != "" {
				totalThinking.WriteString(event.Delta.Thinking)
				if onChunk != nil {
					if err := onChunk(StreamChunk{Type: domain.ChunkTypeThinking, Thinking: event.Delta.Thinking}); err != nil {
						return nil, err
					}
				}
			}
			if event.Delta.Text != "" {
				totalText.WriteString(event.Delta.Text)
				if onChunk != nil {
					if err := onChunk(StreamChunk{Type: domain.ChunkTypeContent, Text: event.Delta.Text}); err != nil {
						return nil, err
					}
				}
			}
			if event.Delta.PartialJSON != "" {
				if p, ok := pendingTools[event.Index]; ok {
					p.builder.WriteString(event.Delta.PartialJSON)
				}
			}
		case "content_block_stop":
			if p, ok := pendingTools[event.Index]; ok {
				args := p.builder.String()
				if args == "" {
					args = "{}"
				}
				tc := domain.ToolCall{ID: p.id, Name: p.name, Arguments: args}
				toolCalls = append(toolCalls, tc)
				if onChunk != nil {
					if err := onChunk(StreamChunk{Type: domain.ChunkTypeToolCall, ToolCall: &tc}); err != nil {
						return nil, err
					}
				}
				delete(pendingTools, event.Index)
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
		ToolCalls:        toolCalls,
	}, nil
}
