package ai

import (
	"context"
	"errors"
	"time"

	"merlincode/internal/domain"
)

// ValidateChatRequest rejects untrusted model IDs and oversized history before persistence or HTTP.
const MaxChatInputBytes = 256 * 1024
const MaxChatMessages = 200
const MaxChatOutputBytes = 2 * 1024 * 1024
const ChatTimeout = 5 * time.Minute

func ValidateChatRequest(providerID, model string, messages []domain.ChatMessage) error {
	if _, ok := GetProviderConfig(providerID); !ok {
		return domain.ErrUnknownAIProvider
	}
	if model == "" {
		model = GetOrchestratorModel(providerID)
	}
	if !IsModelAllowedForProvider(providerID, model, nil) {
		return domain.ErrInvalidModel
	}
	if len(messages) == 0 {
		return domain.ErrInvalidChat
	}
	if len(messages) > MaxChatMessages {
		return domain.ErrChatTooLarge
	}
	size := 0
	for _, m := range messages {
		if m.Role != domain.ChatRoleUser && m.Role != domain.ChatRoleAssistant && m.Role != domain.ChatRoleSystem {
			return domain.ErrInvalidChat
		}
		if len(m.Content) > MaxChatInputBytes-size {
			return domain.ErrChatTooLarge
		}
		size += len(m.Content)
	}
	return nil
}

// StreamChat bounds input, output, duration and concurrency independently of the WebView.
func (s *Service) StreamChat(ctx context.Context, providerID, model string, messages []domain.ChatMessage,
	onChunk func(domain.StreamChunk) error) (*domain.ChatCompletionResult, error) {
	if err := s.InitializationError(); err != nil {
		return nil, err
	}
	providerID = NormalizeProviderID(providerID)
	if model == "" {
		model = GetOrchestratorModel(providerID)
	}
	if err := ValidateChatRequest(providerID, model, messages); err != nil {
		return nil, err
	}
	if !s.streamMu.TryLock() {
		return nil, domain.ErrChatBusy
	}
	defer s.streamMu.Unlock()
	streamer, ok := s.validators[providerID].(Streamer)
	if !ok {
		return nil, domain.ErrAIProviderOperationFailed
	}
	s.mu.RLock()
	apiKey := s.credentials[providerID].APIKey
	if apiKey == "" {
		secret, _, err := s.readSecret(providerID)
		if err != nil {
			s.mu.RUnlock()
			return nil, err
		}
		apiKey = secret
	}
	s.mu.RUnlock()
	if apiKey == "" {
		return nil, errors.New("no hay una clave de API configurada para el proveedor")
	}
	ctx, cancel := context.WithTimeout(ctx, ChatTimeout)
	defer cancel()
	size := 0
	result, err := streamer.StreamChat(ctx, apiKey, model, messages, func(chunk domain.StreamChunk) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if len(chunk.Text) > MaxChatOutputBytes-size {
			return domain.ErrChatOutputTooLarge
		}
		size += len(chunk.Text)
		if len(chunk.Thinking) > MaxChatOutputBytes-size {
			return domain.ErrChatOutputTooLarge
		}
		size += len(chunk.Thinking)
		if onChunk != nil {
			return onChunk(chunk)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if result == nil {
		return nil, domain.ErrAIProviderOperationFailed
	}
	if len(result.Content) > MaxChatOutputBytes || len(result.Thinking) > MaxChatOutputBytes-len(result.Content) {
		return nil, domain.ErrChatOutputTooLarge
	}
	return result, nil
}

// StreamChatWithTools es idéntico a StreamChat en validación, límites y contabilidad de tamaño,
// pero además ofrece herramientas al modelo cuando el proveedor implementa la interfaz opcional
// ToolCaller. Si el proveedor no la implementa (p. ej. Google/DeepSeek en el Incremento 1), cae
// automáticamente a Streamer.StreamChat sin herramientas, sin romper esos proveedores.
func (s *Service) StreamChatWithTools(ctx context.Context, providerID, model string, messages []domain.ChatMessage,
	tools []domain.ToolDefinition, onChunk func(domain.StreamChunk) error) (*domain.ChatCompletionResult, error) {
	if err := s.InitializationError(); err != nil {
		return nil, err
	}
	providerID = NormalizeProviderID(providerID)
	if model == "" {
		model = GetOrchestratorModel(providerID)
	}
	if err := ValidateChatRequest(providerID, model, messages); err != nil {
		return nil, err
	}
	if !s.streamMu.TryLock() {
		return nil, domain.ErrChatBusy
	}
	defer s.streamMu.Unlock()

	validator, ok := s.validators[providerID]
	if !ok {
		return nil, domain.ErrAIProviderOperationFailed
	}

	s.mu.RLock()
	apiKey := s.credentials[providerID].APIKey
	if apiKey == "" {
		secret, _, err := s.readSecret(providerID)
		if err != nil {
			s.mu.RUnlock()
			return nil, err
		}
		apiKey = secret
	}
	s.mu.RUnlock()
	if apiKey == "" {
		return nil, errors.New("no hay una clave de API configurada para el proveedor")
	}

	ctx, cancel := context.WithTimeout(ctx, ChatTimeout)
	defer cancel()
	size := 0
	wrappedOnChunk := func(chunk domain.StreamChunk) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if len(chunk.Text) > MaxChatOutputBytes-size {
			return domain.ErrChatOutputTooLarge
		}
		size += len(chunk.Text)
		if len(chunk.Thinking) > MaxChatOutputBytes-size {
			return domain.ErrChatOutputTooLarge
		}
		size += len(chunk.Thinking)
		if onChunk != nil {
			return onChunk(chunk)
		}
		return nil
	}

	var result *domain.ChatCompletionResult
	var err error
	if toolCaller, ok := validator.(ToolCaller); ok {
		result, err = toolCaller.StreamChatWithTools(ctx, apiKey, model, messages, tools, wrappedOnChunk)
	} else {
		streamer, ok := validator.(Streamer)
		if !ok {
			return nil, domain.ErrAIProviderOperationFailed
		}
		result, err = streamer.StreamChat(ctx, apiKey, model, messages, wrappedOnChunk)
	}
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if result == nil {
		return nil, domain.ErrAIProviderOperationFailed
	}
	if len(result.Content) > MaxChatOutputBytes || len(result.Thinking) > MaxChatOutputBytes-len(result.Content) {
		return nil, domain.ErrChatOutputTooLarge
	}
	return result, nil
}
