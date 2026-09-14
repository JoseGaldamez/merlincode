package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"merlincode/internal/domain"
)

func TestSanitizeAIProviderErrorNeverExposesInternalDetails(t *testing.T) {
	internal := fmt.Errorf("%w: C:\\private\\vault SENTINEL_INTERNAL_TOKEN", domain.ErrCredentialStoreUnavailable)
	public := sanitizeAIProviderError(internal)
	if public == nil {
		t.Fatal("se esperaba un error público")
	}
	if !errors.Is(public, domain.ErrCredentialStoreUnavailable) {
		t.Fatalf("categoría pública inesperada: %v", public)
	}
	if strings.Contains(public.Error(), "SENTINEL") || strings.Contains(public.Error(), "private") {
		t.Fatalf("el error público expuso detalles internos: %v", public)
	}
}

func TestAppContextUsesLifecycleCancellation(t *testing.T) {
	parent, cancelParent := context.WithCancel(context.Background())
	app := &App{ctx: parent}
	requestCtx, cancelRequest, err := app.getAppContext(time.Minute)
	if err != nil {
		t.Fatalf("getAppContext retornó error: %v", err)
	}
	defer cancelRequest()

	cancelParent()
	select {
	case <-requestCtx.Done():
		if !errors.Is(requestCtx.Err(), context.Canceled) {
			t.Fatalf("cancelación inesperada: %v", requestCtx.Err())
		}
	case <-time.After(time.Second):
		t.Fatal("la solicitud no fue cancelada con el ciclo de vida de la aplicación")
	}
}

func TestAppContextRejectsCallsBeforeStartup(t *testing.T) {
	app := &App{}
	if _, _, err := app.getAppContext(time.Second); !errors.Is(err, domain.ErrRuntimeNotInitialized) {
		t.Fatalf("se esperaba ErrRuntimeNotInitialized, obtenido %v", err)
	}
}
