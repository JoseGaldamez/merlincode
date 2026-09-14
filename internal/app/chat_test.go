package app

import (
	"context"
	"errors"
	"testing"

	"merlincode/internal/domain"
)

func TestChatSlotSurvivesCancelUntilCompletion(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a := &App{ctx: ctx}
	stream, finish, err := a.beginChatStream("same-id")
	if err != nil {
		t.Fatal(err)
	}
	if !a.CancelChatStream("same-id") {
		t.Fatal("cancel missing")
	}
	if !errors.Is(stream.Err(), context.Canceled) {
		t.Fatal("stream not cancelled")
	}
	for _, id := range []string{"same-id", "different-id"} {
		if _, _, err := a.beginChatStream(id); !errors.Is(err, domain.ErrChatBusy) {
			t.Fatalf("overlapping stream allowed: %v", err)
		}
	}
	finish()
	_, nextFinish, err := a.beginChatStream("same-id")
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	nextFinish()
	a.streamsWG.Wait()
	if _, _, err := a.beginChatStream("new"); !errors.Is(err, context.Canceled) {
		t.Fatal("accepted stream after shutdown")
	}
}

func TestChatErrorSanitization(t *testing.T) {
	err := sanitizeAIProviderError(errors.New("SENTINEL https://private.invalid/ secret"))
	if err != domain.ErrAIProviderOperationFailed {
		t.Fatalf("unsafe error: %v", err)
	}
	for _, expected := range []error{domain.ErrChatTooLarge, domain.ErrInvalidModel, domain.ErrChatBusy, domain.ErrChatOutputTooLarge} {
		if sanitizeAIProviderError(expected) != expected {
			t.Fatalf("lost public error: %v", expected)
		}
	}
}
