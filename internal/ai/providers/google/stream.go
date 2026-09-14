package google

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"merlincode/internal/ai/transport"
	"merlincode/internal/domain"
)

// StreamChunk es un alias de compatibilidad hacia domain.StreamChunk.
type StreamChunk = domain.StreamChunk

type googleStreamResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text    string `json:"text"`
				Thought bool   `json:"thought,omitempty"`
			} `json:"parts"`
			Role string `json:"role"`
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

// StreamChat realiza una llamada en streaming vía SSE a Google Gemini (streamGenerateContent?alt=sse).
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
		return nil, errors.New("la clave de API de Google Gemini no puede estar vacía")
	}
	if model == "" {
		model = "gemini-3.8-flash"
	}

	// Mapear historial al formato de Google Gemini ("user" / "model")
	type geminiPart struct {
		Text string `json:"text"`
	}
	type geminiContent struct {
		Role  string       `json:"role"`
		Parts []geminiPart `json:"parts"`
	}

	var contents []geminiContent
	for _, m := range messages {
		role := "user"
		if m.Role == domain.ChatRoleAssistant {
			role = "model"
		}
		contents = append(contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: m.Content}},
		})
	}

	if len(contents) == 0 {
		return nil, errors.New("no hay mensajes para enviar a Google Gemini")
	}

	reqBody := struct {
		Contents         []geminiContent         `json:"contents"`
		GenerationConfig *googleGenerationConfig `json:"generationConfig,omitempty"`
	}{
		Contents: contents,
		GenerationConfig: &googleGenerationConfig{
			MaxOutputTokens: 4096,
		},
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("error codificando solicitud a Gemini: %w", err)
	}

	endpoint := "https://generativelanguage.googleapis.com/v1beta/models/" + url.PathEscape(model) + ":streamGenerateContent?alt=sse"
	log.Printf("[Google AI] Conectando a %s con modelo '%s'...", endpoint, model)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", apiKey)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := transport.StreamClient.Do(req)
	if err != nil {
		log.Printf("[Google AI] Error de conexión/transporte: %v", err)
		return nil, errors.New(transport.SanitizeTransportError(err, "Google AI"))
	}
	defer transport.CloseHTTPResponse(resp)

	log.Printf("[Google AI] Respuesta HTTP: %d %s", resp.StatusCode, resp.Status)

	if resp.StatusCode != http.StatusOK {
		var errResp googleStreamResponse
		_ = transport.DecodeJSONLimited(resp.Body, &errResp, transport.DefaultMaxResponseBytes)
		if errResp.Error != nil && errResp.Error.Message != "" {
			errLow := strings.ToLower(errResp.Error.Message)
			log.Printf("[Google AI] Error devuelto por API (%d): %s", resp.StatusCode, errResp.Error.Message)
			if strings.Contains(errLow, "quota") || strings.Contains(errLow, "resource_exhausted") {
				return nil, errors.New("Google Gemini reporta que excediste la cuota de peticiones (Resource Exhausted). Intenta en unos minutos.")
			}
			if strings.Contains(errLow, "not found") || strings.Contains(errLow, "model") {
				return nil, fmt.Errorf("el modelo '%s' no existe en Google Gemini o no está disponible", model)
			}
		}
		return nil, errors.New(transport.SafeProviderHTTPError("Google AI", resp.StatusCode))
	}

	scanner := bufio.NewScanner(resp.Body)
	// Permitir líneas de hasta 1 MiB para chunks grandes
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var totalText strings.Builder
	var totalThinking strings.Builder
	var promptTokens int64
	var completionTokens int64

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue // Comentario o heartbeat SSE
		}

		if !strings.HasPrefix(line, "data:") {
			continue
		}

		jsonData := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if jsonData == "" {
			continue
		}

		var chunkResp googleStreamResponse
		if err := json.Unmarshal([]byte(jsonData), &chunkResp); err != nil {
			continue // Ignorar líneas no JSON
		}

		if chunkResp.UsageMetadata != nil {
			promptTokens = chunkResp.UsageMetadata.PromptTokenCount
			completionTokens = chunkResp.UsageMetadata.CandidatesTokenCount
		}

		for _, cand := range chunkResp.Candidates {
			for _, part := range cand.Content.Parts {
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

	if err := scanner.Err(); err != nil && !errors.Is(err, context.Canceled) {
		return nil, fmt.Errorf("error leyendo stream de Gemini: %w", err)
	}

	return &domain.ChatCompletionResult{
		Content:          totalText.String(),
		Thinking:         totalThinking.String(),
		Model:            model,
		TokensPrompt:     promptTokens,
		TokensCompletion: completionTokens,
	}, nil
}
