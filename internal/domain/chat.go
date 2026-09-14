package domain

// ChatRole identifica el emisor del mensaje en la conversación.
type ChatRole string

const (
	ChatRoleUser      ChatRole = "user"
	ChatRoleAssistant ChatRole = "assistant"
	ChatRoleSystem    ChatRole = "system"
)

// ChatMessage representa un mensaje dentro del historial de la conversación.
type ChatMessage struct {
	Role    ChatRole `json:"role"`
	Content string   `json:"content"`
}

// ChatStreamRequest contiene los parámetros para iniciar una respuesta en streaming desde el frontend.
type ChatStreamRequest struct {
	SessionID     string        `json:"sessionId"`
	MessageID     string        `json:"messageId"`
	UserMessageID string        `json:"userMessageId,omitempty"`
	ProviderID    string        `json:"providerId"`
	ModelID       string        `json:"modelId,omitempty"`
	Prompt        string        `json:"prompt"`
	History       []ChatMessage `json:"history,omitempty"`
}

// StreamChunkType clasifica el tipo de contenido recibido en tiempo real.
type StreamChunkType string

const (
	ChunkTypeStatus   StreamChunkType = "status"
	ChunkTypeThinking StreamChunkType = "thinking"
	ChunkTypeContent  StreamChunkType = "content"
	ChunkTypeDone     StreamChunkType = "done"
	ChunkTypeError    StreamChunkType = "error"
)

// StreamChunk representa un fragmento de contenido o pensamiento recibido en tiempo real.
type StreamChunk struct {
	Type     StreamChunkType `json:"type"`
	Text     string          `json:"text,omitempty"`
	Thinking string          `json:"thinking,omitempty"`
}

// ChatStreamEvent es el evento estructurado que se transmite en tiempo real vía Wails hacia el frontend.
type ChatStreamEvent struct {
	SessionID        string          `json:"sessionId"`
	MessageID        string          `json:"messageId"`
	Type             StreamChunkType `json:"type"`
	StatusText       string          `json:"statusText,omitempty"`
	Content          string          `json:"content,omitempty"`
	Thinking         string          `json:"thinking,omitempty"`
	ProviderID       string          `json:"providerId,omitempty"`
	ModelID          string          `json:"modelId,omitempty"`
	TokensPrompt     int64           `json:"tokensPrompt,omitempty"`
	TokensCompletion int64           `json:"tokensCompletion,omitempty"`
	Error            string          `json:"error,omitempty"`
}

// ChatCompletionResult almacena el resultado final de la generación del chat.
type ChatCompletionResult struct {
	Content          string `json:"content"`
	Thinking         string `json:"thinking,omitempty"`
	Model            string `json:"model"`
	TokensPrompt     int64  `json:"tokensPrompt"`
	TokensCompletion int64  `json:"tokensCompletion"`
}
