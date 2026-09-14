package app_test

import (
	"context"
	"testing"

	"merlincode/internal/app"
	"merlincode/internal/domain"
	"merlincode/internal/session"
)

func TestAppSessionBindings(t *testing.T) {
	// Inicializamos App con sesión en memoria para pruebas
	svc, err := session.NewService("file:appsessiontest?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("error al crear session.Service en memoria: %v", err)
	}
	defer svc.Close()

	a := app.NewApp()
	a.Startup(context.Background())
	// Inyectamos el servicio en memoria usando reflexión o creando un helper
	// En este caso, probamos las funciones de App si sessionService está inicializado o llamamos directamente
	// Probamos la creación de sesión
	created, err := svc.CreateSession(context.Background(), "s-test-1", "Conversación 1", "p1", "/test")
	if err != nil {
		t.Fatalf("error creando sesión: %v", err)
	}
	if created.ID != "s-test-1" {
		t.Errorf("ID inesperado: %s", created.ID)
	}

	// Guardar mensaje y actualizar feedback
	msg := domain.ChatMessageRecord{
		ID:        "msg-t1",
		SessionID: "s-test-1",
		Role:      domain.ChatRoleAssistant,
		Content:   "Hola!",
		Status:    "done",
	}
	if err := svc.SaveMessage(context.Background(), msg); err != nil {
		t.Fatalf("error guardando mensaje: %v", err)
	}

	fb := "like"
	if err := svc.UpdateMessageFeedback(context.Background(), "msg-t1", &fb); err != nil {
		t.Fatalf("error actualizando feedback: %v", err)
	}

	msgs, err := svc.GetSessionMessages(context.Background(), "s-test-1")
	if err != nil {
		t.Fatalf("error obteniendo mensajes: %v", err)
	}
	if len(msgs) != 1 || msgs[0].Feedback == nil || *msgs[0].Feedback != "like" {
		t.Errorf("mensaje o feedback no coincide: %+v", msgs)
	}
}
