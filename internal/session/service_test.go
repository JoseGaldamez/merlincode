package session_test

import (
	"context"
	"testing"
	"time"

	"merlincode/internal/domain"
	"merlincode/internal/session"
)

func TestSessionService(t *testing.T) {
	ctx := context.Background()
	// Usamos base de datos SQLite en memoria para tests rápidos y limpios
	svc, err := session.NewService("file:testmemdb?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("falló NewService: %v", err)
	}
	defer svc.Close()

	// 1. Crear sesión
	created, err := svc.CreateSession(ctx, "sess-1", "Test Session", "proj-1", "D:/test")
	if err != nil {
		t.Fatalf("error al crear sesión: %v", err)
	}
	if created.ID != "sess-1" || created.Title != "Test Session" {
		t.Errorf("datos inesperados en sesión creada: %+v", created)
	}

	// 2. Listar sesiones
	sessions, err := svc.ListSessions(ctx)
	if err != nil {
		t.Fatalf("error al listar sesiones: %v", err)
	}
	if len(sessions) != 1 || sessions[0].ID != "sess-1" {
		t.Fatalf("resultado inesperado en ListSessions: %+v", sessions)
	}

	// 3. Guardar mensaje de usuario
	userMsg := domain.ChatMessageRecord{
		ID:        "msg-1",
		SessionID: "sess-1",
		Role:      domain.ChatRoleUser,
		Content:   "Hola mundo",
		CreatedAt: time.Now(),
	}
	if err := svc.SaveMessage(ctx, userMsg); err != nil {
		t.Fatalf("error al guardar mensaje de usuario: %v", err)
	}

	// 4. Guardar mensaje de asistente con métricas
	asstMsg := domain.ChatMessageRecord{
		ID:               "msg-2",
		SessionID:        "sess-1",
		Role:             domain.ChatRoleAssistant,
		Content:          "Respuesta del modelo",
		ThoughtChain:     "Pensando...",
		ProviderID:       "openai",
		ModelID:          "gpt-4o",
		TokensPrompt:     15,
		TokensCompletion: 42,
		DurationSeconds:  2,
		Status:           "done",
		CreatedAt:        time.Now().Add(time.Second),
	}
	if err := svc.SaveMessage(ctx, asstMsg); err != nil {
		t.Fatalf("error al guardar mensaje del asistente: %v", err)
	}

	// 5. Verificar mensajes cargados
	msgs, err := svc.GetSessionMessages(ctx, "sess-1")
	if err != nil {
		t.Fatalf("error al obtener mensajes: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("se esperaban 2 mensajes, se obtuvieron: %d", len(msgs))
	}
	if msgs[1].TokensCompletion != 42 || msgs[1].DurationSeconds != 2 || msgs[1].ThoughtChain != "Pensando..." {
		t.Errorf("mensaje de asistente no conservó todas las métricas: %+v", msgs[1])
	}

	// 6. Actualizar feedback (like/dislike)
	like := "like"
	if err := svc.UpdateMessageFeedback(ctx, "msg-2", &like); err != nil {
		t.Fatalf("error al actualizar feedback a like: %v", err)
	}
	msgs, _ = svc.GetSessionMessages(ctx, "sess-1")
	if msgs[1].Feedback == nil || *msgs[1].Feedback != "like" {
		t.Errorf("feedback no persistido correctamente: %+v", msgs[1].Feedback)
	}

	// Quitar feedback
	if err := svc.UpdateMessageFeedback(ctx, "msg-2", nil); err != nil {
		t.Fatalf("error al quitar feedback: %v", err)
	}
	msgs, _ = svc.GetSessionMessages(ctx, "sess-1")
	if msgs[1].Feedback != nil {
		t.Errorf("feedback no fue eliminado correctamente: %+v", msgs[1].Feedback)
	}

	// 7. Modificar título de sesión
	if err := svc.UpdateSessionTitle(ctx, "sess-1", "Nuevo Título"); err != nil {
		t.Fatalf("error al actualizar título: %v", err)
	}
	sessions, _ = svc.ListSessions(ctx)
	if sessions[0].Title != "Nuevo Título" {
		t.Errorf("título no actualizado: %s", sessions[0].Title)
	}
	if sessions[0].MessagesCount != 2 {
		t.Errorf("conteo de mensajes incorrecto: esperado 2, obtenido %d", sessions[0].MessagesCount)
	}

	// 8. Eliminar sesión en cascada
	if err := svc.DeleteSession(ctx, "sess-1"); err != nil {
		t.Fatalf("error al eliminar sesión: %v", err)
	}
	sessions, _ = svc.ListSessions(ctx)
	if len(sessions) != 0 {
		t.Errorf("la sesión no fue eliminada")
	}
	msgs, _ = svc.GetSessionMessages(ctx, "sess-1")
	if len(msgs) != 0 {
		t.Errorf("los mensajes no se eliminaron en cascada, quedaron %d", len(msgs))
	}
}
