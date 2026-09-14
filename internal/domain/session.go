package domain

import (
	"time"
)

// Session representa una conversación o sesión de trabajo en el historial de Merlin Code.
type Session struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	ProjectID     string    `json:"projectId,omitempty"`
	ProjectPath   string    `json:"projectPath,omitempty"`
	MessagesCount int       `json:"messagesCount"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// ChatMessageRecord representa un mensaje almacenado persistentemente en la base de datos local SQLite.
type ChatMessageRecord struct {
	ID               string    `json:"id"`
	SessionID        string    `json:"sessionId"`
	Role             ChatRole  `json:"role"`
	Content          string    `json:"content"`
	ThoughtChain     string    `json:"thoughtChain,omitempty"`
	ProviderID       string    `json:"providerId,omitempty"`
	ModelID          string    `json:"modelId,omitempty"`
	TokensPrompt     int64     `json:"tokensPrompt,omitempty"`
	TokensCompletion int64     `json:"tokensCompletion,omitempty"`
	DurationSeconds  int       `json:"durationSeconds,omitempty"`
	Feedback         *string   `json:"feedback,omitempty"` // "like", "dislike" o nil
	Status           string    `json:"status,omitempty"`   // "done", "error", etc.
	CreatedAt        time.Time `json:"createdAt"`
}
