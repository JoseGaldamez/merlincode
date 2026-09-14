package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"merlincode/internal/domain"
)

// ListSessions retorna la lista de sesiones guardadas en SQLite
func (a *App) ListSessions() ([]domain.Session, error) {
	if a.sessionService == nil {
		return []domain.Session{}, nil
	}
	ctx, cancel, err := a.getAppContext(5 * time.Second)
	if err != nil {
		ctx = context.Background()
	} else {
		defer cancel()
	}
	return a.sessionService.ListSessions(ctx)
}

// CreateSession crea una nueva sesión en SQLite
func (a *App) CreateSession(id, title, projectID, projectPath string) (domain.Session, error) {
	if a.sessionService == nil {
		return domain.Session{}, errors.New("servicio de sesiones no disponible")
	}
	ctx, cancel, err := a.getAppContext(5 * time.Second)
	if err != nil {
		ctx = context.Background()
	} else {
		defer cancel()
	}
	return a.sessionService.CreateSession(ctx, id, title, projectID, projectPath)
}

// UpdateSessionTitle actualiza el título de una sesión
func (a *App) UpdateSessionTitle(id, title string) error {
	if a.sessionService == nil {
		return errors.New("servicio de sesiones no disponible")
	}
	ctx, cancel, err := a.getAppContext(5 * time.Second)
	if err != nil {
		ctx = context.Background()
	} else {
		defer cancel()
	}
	return a.sessionService.UpdateSessionTitle(ctx, id, title)
}

// DeleteSession elimina una sesión y sus mensajes asociados en cascada
func (a *App) DeleteSession(id string) error {
	if a.sessionService == nil {
		return errors.New("servicio de sesiones no disponible")
	}
	ctx, cancel, err := a.getAppContext(5 * time.Second)
	if err != nil {
		ctx = context.Background()
	} else {
		defer cancel()
	}
	return a.sessionService.DeleteSession(ctx, id)
}

// GetSessionMessages retorna los mensajes almacenados de una sesión
func (a *App) GetSessionMessages(sessionID string) ([]domain.ChatMessageRecord, error) {
	if a.sessionService == nil {
		return []domain.ChatMessageRecord{}, nil
	}
	ctx, cancel, err := a.getAppContext(5 * time.Second)
	if err != nil {
		ctx = context.Background()
	} else {
		defer cancel()
	}
	return a.sessionService.GetSessionMessages(ctx, sessionID)
}

// SaveChatMessage permite persistir o actualizar un mensaje directamente
func (a *App) SaveChatMessage(msg domain.ChatMessageRecord) error {
	if a.sessionService == nil {
		return errors.New("servicio de sesiones no disponible")
	}
	ctx, cancel, err := a.getAppContext(5 * time.Second)
	if err != nil {
		ctx = context.Background()
	} else {
		defer cancel()
	}
	return a.sessionService.SaveMessage(ctx, msg)
}

// UpdateMessageFeedback guarda o quita el feedback ("like", "dislike", o ""/vacío para eliminarlo)
func (a *App) UpdateMessageFeedback(messageID string, feedback string) error {
	if a.sessionService == nil {
		return errors.New("servicio de sesiones no disponible")
	}
	ctx, cancel, err := a.getAppContext(5 * time.Second)
	if err != nil {
		ctx = context.Background()
	} else {
		defer cancel()
	}
	var fb *string
	trimmed := strings.TrimSpace(feedback)
	if trimmed == "like" || trimmed == "dislike" {
		fb = &trimmed
	}
	return a.sessionService.UpdateMessageFeedback(ctx, messageID, fb)
}
