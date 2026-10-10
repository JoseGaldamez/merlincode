package google

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"merlincode/internal/ai/transport"
	"merlincode/internal/domain"
)

// geminiFunctionCall es la invocación de función que Gemini incluye dentro de una parte de
// contenido. A diferencia de OpenAI, Gemini manda los argumentos completos (Args) en una sola
// parte, sin fragmentar el JSON entre varios eventos del stream.
type geminiFunctionCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

// geminiFunctionResponse es el resultado de una función que se reenvía a Gemini en el turno siguiente.
type geminiFunctionResponse struct {
	Name     string         `json:"name"`
	Response map[string]any `json:"response"`
}

type geminiToolPart struct {
	Text             string                  `json:"text,omitempty"`
	Thought          bool                    `json:"thought,omitempty"`
	ThoughtSignature string                  `json:"thoughtSignature,omitempty"`
	FunctionCall     *geminiFunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *geminiFunctionResponse `json:"functionResponse,omitempty"`
}

type geminiToolContent struct {
	Role  string           `json:"role"`
	Parts []geminiToolPart `json:"parts"`
}

type geminiFunctionDeclaration struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// geminiTool es la traducción de domain.ToolDefinition al wire format de Gemini.
type geminiTool struct {
	FunctionDeclarations []geminiFunctionDeclaration `json:"functionDeclarations"`
}

type googleToolsStreamResponse struct {
	Candidates []struct {
		Content struct {
			Parts []geminiToolPart `json:"parts"`
			Role  string           `json:"role"`
		} `json:"content"`
	} `json:"candidates"`
	UsageMetadata *struct {
		PromptTokenCount     int64 `json:"promptTokenCount"`
		CandidatesTokenCount int64 `json:"candidatesTokenCount"`
		TotalTokenCount      int64 `json:"totalTokenCount"`
	} `json:"usageMetadata,omitempty"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error,omitempty"`
}

// StreamChatWithTools realiza una llamada en streaming vía SSE a Google Gemini ofreciendo
// herramientas al modelo, satisfaciendo la interfaz opcional ai.ToolCaller. Gemini no asigna un
// ID nativo a cada llamada de función, así que se sintetiza uno local (gemini-tool-N) únicamente
// para la correlación interna del bucle de agente; el wire format de vuelta hacia Gemini solo
// necesita el nombre de la función, no el ID.
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
		return nil, errors.New("la clave de API de Google Gemini no puede estar vacía")
	}
	if model == "" {
		model = "gemini-3.8-flash"
	}

	var contents []geminiToolContent
	for _, m := range messages {
		if len(m.ToolResults) > 0 {
			parts := make([]geminiToolPart, 0, len(m.ToolResults))
			for _, tr := range m.ToolResults {
				responseObj := map[string]any{"result": tr.Content}
				if tr.IsError {
					responseObj = map[string]any{"error": tr.Content}
				}
				parts = append(parts, geminiToolPart{
					FunctionResponse: &geminiFunctionResponse{
						Name:     tr.Name,
						Response: responseObj,
					},
				})
			}
			// La API de Gemini espera el rol "user" para el turno que lleva el functionResponse
			// de vuelta (no "function" pese a que ese fue el nombre histórico de la parte).
			contents = append(contents, geminiToolContent{Role: "user", Parts: parts})
			continue
		}

		role := "user"
		if m.Role == domain.ChatRoleAssistant {
			role = "model"
		}

		var parts []geminiToolPart
		if m.Content != "" {
			parts = append(parts, geminiToolPart{Text: m.Content})
		}
		for _, tc := range m.ToolCalls {
			var args map[string]any
			if tc.Arguments != "" {
				_ = json.Unmarshal([]byte(tc.Arguments), &args)
			}
			if args == nil {
				args = map[string]any{}
			}
			parts = append(parts, geminiToolPart{
				FunctionCall:     &geminiFunctionCall{Name: tc.Name, Args: args},
				ThoughtSignature: tc.ThoughtSignature,
			})
		}
		if len(parts) == 0 {
			continue
		}
		contents = append(contents, geminiToolContent{Role: role, Parts: parts})
	}

	if len(contents) == 0 {
		return nil, errors.New("no hay mensajes para enviar a Google Gemini")
	}

	var reqTools []geminiTool
	if len(tools) > 0 {
		decls := make([]geminiFunctionDeclaration, 0, len(tools))
		for _, t := range tools {
			decls = append(decls, geminiFunctionDeclaration{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
			})
		}
		reqTools = []geminiTool{{FunctionDeclarations: decls}}
	}

	reqBody := struct {
		Contents         []geminiToolContent     `json:"contents"`
		Tools            []geminiTool            `json:"tools,omitempty"`
		GenerationConfig *googleGenerationConfig `json:"generationConfig,omitempty"`
	}{
		Contents: contents,
		Tools:    reqTools,
		GenerationConfig: &googleGenerationConfig{
			MaxOutputTokens: 4096,
		},
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("error codificando solicitud a Gemini: %w", err)
	}

	endpoint := "https://generativelanguage.googleapis.com/v1beta/models/" + url.PathEscape(model) + ":streamGenerateContent?alt=sse"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", apiKey)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := transport.StreamClient.Do(req)
	if err != nil {
		return nil, errors.New(transport.SanitizeTransportError(err, "Google AI"))
	}
	defer transport.CloseHTTPResponse(resp)

	if resp.StatusCode != http.StatusOK {
		var errResp googleToolsStreamResponse
		_ = transport.DecodeJSONLimited(resp.Body, &errResp, transport.DefaultMaxResponseBytes)
		if errResp.Error != nil && errResp.Error.Message != "" {
			errLow := strings.ToLower(errResp.Error.Message)
			if strings.Contains(errLow, "quota") || strings.Contains(errLow, "resource_exhausted") {
				return nil, errors.New("Google Gemini reporta que excediste la cuota de peticiones (Resource Exhausted). Intenta en unos minutos.")
			}
			if strings.Contains(errLow, "not found") && strings.Contains(errLow, "model") {
				return nil, fmt.Errorf("el modelo '%s' no existe en Google Gemini o no está disponible (detalle Gemini: %s)", model, errResp.Error.Message)
			}
			return nil, fmt.Errorf("%s (detalle Gemini: %s)", transport.SafeProviderHTTPError("Google AI", resp.StatusCode), errResp.Error.Message)
		}
		return nil, errors.New(transport.SafeProviderHTTPError("Google AI", resp.StatusCode))
	}

	scanner := bufio.NewScanner(transport.LimitedStreamReader(resp.Body))
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var totalText strings.Builder
	var totalThinking strings.Builder
	var promptTokens int64
	var completionTokens int64
	var toolCalls []domain.ToolCall
	toolCallCounter := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		if !strings.HasPrefix(line, "data:") {
			continue
		}

		jsonData := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if jsonData == "" {
			continue
		}

		var chunkResp googleToolsStreamResponse
		if err := json.Unmarshal([]byte(jsonData), &chunkResp); err != nil {
			continue
		}

		if chunkResp.UsageMetadata != nil {
			promptTokens = chunkResp.UsageMetadata.PromptTokenCount
			completionTokens = chunkResp.UsageMetadata.CandidatesTokenCount
		}

		for _, cand := range chunkResp.Candidates {
			for _, part := range cand.Content.Parts {
				if part.FunctionCall != nil {
					argsJSON, err := json.Marshal(part.FunctionCall.Args)
					if err != nil || len(argsJSON) == 0 {
						argsJSON = []byte("{}")
					}
					tc := domain.ToolCall{
						ID:               fmt.Sprintf("gemini-tool-%d", toolCallCounter),
						Name:             part.FunctionCall.Name,
						Arguments:        string(argsJSON),
						ThoughtSignature: part.ThoughtSignature,
					}
					toolCallCounter++
					toolCalls = append(toolCalls, tc)
					if onChunk != nil {
						if err := onChunk(StreamChunk{Type: domain.ChunkTypeToolCall, ToolCall: &tc}); err != nil {
							return nil, err
						}
					}
					continue
				}
				if part.Thought {
					totalThinking.WriteString(part.Text)
					if onChunk != nil && part.Text != "" {
						if err := onChunk(StreamChunk{
							Type:     domain.ChunkTypeThinking,
							Thinking: part.Text,
						}); err != nil {
							return nil, err
						}
					}
				} else if part.Text != "" {
					totalText.WriteString(part.Text)
					if onChunk != nil {
						if err := onChunk(StreamChunk{
							Type: domain.ChunkTypeContent,
							Text: part.Text,
						}); err != nil {
							return nil, err
						}
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
		ToolCalls:        toolCalls,
	}, nil
}
