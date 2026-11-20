package openai

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"merlincode/internal/ai/transport"
	"merlincode/internal/domain"
)

// StreamChunk es un alias de compatibilidad hacia domain.StreamChunk.
type StreamChunk = domain.StreamChunk

type responseStreamEvent struct {
	Type     string          `json:"type"`
	Delta    string          `json:"delta"`
	Code     string          `json:"code"`
	Message  string          `json:"message"`
	Response *openAIResponse `json:"response"`
}

func (c Client) StreamChat(ctx context.Context, apiKey, model string, messages []domain.ChatMessage,
	onChunk func(StreamChunk) error) (*domain.ChatCompletionResult, error) {
	return c.StreamChatWithTools(ctx, apiKey, model, messages, nil, onChunk)
}

// StreamChatWithTools streams a stateless Responses API request using only Merlin's local tools.
func (Client) StreamChatWithTools(ctx context.Context, apiKey, model string, messages []domain.ChatMessage,
	tools []domain.ToolDefinition, onChunk func(StreamChunk) error) (*domain.ChatCompletionResult, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, errors.New("la clave de API de OpenAI no puede estar vacía")
	}
	model = strings.TrimSpace(model)
	if model == "" {
		model = defaultModel
	}
	input, err := responsesInput(messages)
	if err != nil {
		return nil, err
	}
	requestTools := make([]responseTool, 0, len(tools))
	for _, tool := range tools {
		requestTools = append(requestTools, responseTool{
			Type: "function", Name: tool.Name, Description: tool.Description, Parameters: tool.InputSchema,
			// Preserve the optional fields accepted by Merlin's existing tools.
			Strict: false,
		})
	}
	req, err := newResponsesRequest(ctx, apiKey, responsesRequest{
		Model: model, Input: input, Tools: requestTools, Reasoning: reasoningForModel(model),
		MaxOutputTokens: 4096, Store: false, Stream: true,
	})
	if err != nil {
		return nil, err
	}
	resp, err := transport.StreamClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, errors.New(transport.SanitizeTransportError(err, "OpenAI"))
	}
	defer transport.CloseHTTPResponse(resp)

	if resp.StatusCode != http.StatusOK {
		var rejected openAIResponse
		_ = transport.DecodeJSONLimited(resp.Body, &rejected, transport.DefaultMaxResponseBytes)
		return nil, responseFailure(resp.StatusCode, rejected.Error)
	}

	return readResponsesStream(ctx, resp, model, onChunk)
}

func readResponsesStream(ctx context.Context, resp *http.Response, model string,
	onChunk func(StreamChunk) error) (*domain.ChatCompletionResult, error) {
	scanner := bufio.NewScanner(transport.LimitedStreamReader(resp.Body))
	scanner.Buffer(make([]byte, 64*1024), transport.DefaultMaxResponseBytes)
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		var event responseStreamEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			return nil, errors.New("OpenAI devolvió un evento de streaming inválido")
		}
		switch event.Type {
		case "response.output_text.delta", "response.refusal.delta":
			if onChunk != nil && event.Delta != "" {
				if err := onChunk(StreamChunk{Type: domain.ChunkTypeContent, Text: event.Delta}); err != nil {
					return nil, err
				}
			}
		case "response.reasoning_summary_text.delta":
			if onChunk != nil && event.Delta != "" {
				if err := onChunk(StreamChunk{Type: domain.ChunkTypeThinking, Thinking: event.Delta}); err != nil {
					return nil, err
				}
			}
		case "response.completed", "response.incomplete", "response.failed":
			if event.Response == nil {
				return nil, errors.New("OpenAI devolvió un evento final sin respuesta")
			}
			if event.Type != "response.completed" && event.Response.Status == "completed" {
				return nil, errors.New("OpenAI devolvió un estado de respuesta incoherente")
			}
			result, err := event.Response.completion(model)
			if err != nil {
				return result, err
			}
			// The completed output has the full arguments. Never execute a partial call.
			for _, call := range result.ToolCalls {
				if onChunk != nil {
					if err := onChunk(StreamChunk{Type: domain.ChunkTypeToolCall, ToolCall: &call}); err != nil {
						return result, err
					}
				}
			}
			return result, nil
		case "error":
			return nil, responseFailure(0, &responseError{Code: event.Code, Message: event.Message})
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := scanner.Err(); err != nil {
		return nil, errors.New(transport.SanitizeTransportError(err, "OpenAI"))
	}
	return nil, errors.New("el streaming de OpenAI se interrumpió antes de completar la respuesta")
}
