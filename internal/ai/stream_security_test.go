package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"merlincode/internal/ai/transport"
	"merlincode/internal/domain"
)

type controlledStreamer struct {
	fakeValidator
	run func(context.Context, func(domain.StreamChunk) error) (*domain.ChatCompletionResult, error)
}

func (s controlledStreamer) StreamChat(ctx context.Context, key, model string, messages []domain.ChatMessage, cb func(domain.StreamChunk) error) (*domain.ChatCompletionResult, error) {
	return s.run(ctx, cb)
}

func TestStreamInputOutputConcurrencyAndCancellation(t *testing.T) {
	messages := []domain.ChatMessage{{Role: domain.ChatRoleUser, Content: "hello"}}
	model := GetOrchestratorModel("google")
	for _, tc := range []struct {
		name, model string
		messages    []domain.ChatMessage
		want        error
	}{
		{"model", "not-allowed", messages, domain.ErrInvalidModel},
		{"size", model, []domain.ChatMessage{{Role: domain.ChatRoleUser, Content: strings.Repeat("x", MaxChatInputBytes+1)}}, domain.ErrChatTooLarge},
		{"count", model, make([]domain.ChatMessage, MaxChatMessages+1), domain.ErrChatTooLarge},
		{"role", model, []domain.ChatMessage{{Role: "invalid"}}, domain.ErrInvalidChat},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateChatRequest("google", tc.model, tc.messages); !errors.Is(err, tc.want) {
				t.Fatalf("got %v", err)
			}
		})
	}
	started := make(chan struct{})
	streamer := controlledStreamer{run: func(ctx context.Context, cb func(domain.StreamChunk) error) (*domain.ChatCompletionResult, error) {
		if _, ok := ctx.Deadline(); !ok {
			t.Error("missing deadline")
		}
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	}}
	svc, _, _ := newTestService(t, map[string]Provider{"google": streamer})
	if err := svc.setSecretVerified("google", "sentinel"); err != nil {
		t.Fatal(err)
	}
	if err := svc.hydrateFromKeyring(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := svc.StreamChat(ctx, "google", model, messages, nil); done <- err }()
	<-started
	if _, err := svc.StreamChat(ctx, "google", model, messages, nil); !errors.Is(err, domain.ErrChatBusy) {
		t.Fatalf("parallel stream accepted: %v", err)
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancellation hung")
	}
	// Verify the released slot and the combined text/thinking budget, not just per-chunk size.
	svc.validators["google"] = controlledStreamer{run: func(ctx context.Context, cb func(domain.StreamChunk) error) (*domain.ChatCompletionResult, error) {
		if err := cb(domain.StreamChunk{Text: strings.Repeat("x", MaxChatOutputBytes)}); err != nil {
			return nil, err
		}
		return nil, cb(domain.StreamChunk{Thinking: "overflow"})
	}}
	if _, err := svc.StreamChat(context.Background(), "google", model, messages, nil); !errors.Is(err, domain.ErrChatOutputTooLarge) {
		t.Fatalf("unbounded output: %v", err)
	}
	deadlineCtx, stop := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer stop()
	if _, err := svc.StreamChat(deadlineCtx, "google", model, messages, nil); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
}

type failingStreamBody struct{}

func (failingStreamBody) Read([]byte) (int, error) {
	return 0, errors.New("SENTINEL https://private.invalid/key")
}
func (failingStreamBody) Close() error { return nil }

func TestAllStreamProvidersBoundBodiesAndSanitizeErrors(t *testing.T) {
	oldClient := transport.StreamClient
	defer func() { transport.StreamClient = oldClient }()
	var logs bytes.Buffer
	oldLog := log.Writer()
	log.SetOutput(&logs)
	defer log.SetOutput(oldLog)
	for _, id := range KnownProviderIDs {
		for _, scenario := range []string{"http error", "read error", "oversize"} {
			t.Run(id+"/"+scenario, func(t *testing.T) {
				transport.StreamClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					if strings.Contains(req.URL.String(), "SENTINEL") {
						t.Fatal("key in URL")
					}
					if id == "google" && req.Header.Get("x-goog-api-key") != "SENTINEL" {
						t.Fatal("missing Google header")
					}
					var payload map[string]any
					if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
						t.Fatal(err)
					}
					field := "max_tokens"
					if id == "openai" {
						field = "max_completion_tokens"
					}
					if id == "google" {
						if payload["generationConfig"].(map[string]any)["maxOutputTokens"] != float64(4096) {
							t.Fatal("unbounded tokens")
						}
					} else if payload[field] != float64(4096) {
						t.Fatal("unbounded tokens")
					}
					resp := &http.Response{StatusCode: 200, Header: make(http.Header)}
					switch scenario {
					case "http error":
						resp.StatusCode = 400
						resp.Body = io.NopCloser(strings.NewReader(`{"error":{"message":"SENTINEL https://private.invalid"}}`))
					case "read error":
						resp.Body = failingStreamBody{}
					case "oversize":
						resp.Body = io.NopCloser(strings.NewReader(strings.Repeat(": heartbeat\n", transport.DefaultMaxResponseBytes/12+100)))
					}
					return resp, nil
				})}
				p := defaultRegistry[id].(Streamer)
				_, err := p.StreamChat(context.Background(), "SENTINEL", GetOrchestratorModel(id), []domain.ChatMessage{{Role: domain.ChatRoleUser, Content: "hi"}}, nil)
				if err == nil {
					t.Fatal("expected controlled failure")
				}
				if strings.Contains(err.Error(), "SENTINEL") || strings.Contains(err.Error(), "https://") {
					t.Fatalf("unsafe public error: %v", err)
				}
			})
		}
	}
	if strings.Contains(logs.String(), "SENTINEL") || strings.Contains(logs.String(), "private.invalid") {
		t.Fatal("secret in logs")
	}
}

func TestStreamPropagatesKeyringReadFailure(t *testing.T) {
	svc, _, store := newTestService(t, map[string]Provider{"google": &fakeStreamer{}})
	store.getError = errors.New("sentinel keyring failure")
	_, err := svc.StreamChat(context.Background(), "google", "", []domain.ChatMessage{{Role: domain.ChatRoleUser, Content: "hi"}}, nil)
	if !errors.Is(err, domain.ErrCredentialStoreUnavailable) {
		t.Fatalf("read failure hidden: %v", err)
	}
}

type gatedStore struct {
	*memorySecretStore
	once                      sync.Once
	entered, release, deleted chan struct{}
	gate                      bool
}

func (s *gatedStore) Get(account string) (string, error) {
	if s.gate {
		s.once.Do(func() { close(s.entered); <-s.release })
	}
	return s.memorySecretStore.Get(account)
}
func (s *gatedStore) Delete(account string) error {
	select {
	case s.deleted <- struct{}{}:
	default:
	}
	return s.memorySecretStore.Delete(account)
}

func TestCredentialSaveAndDeleteAreSerialized(t *testing.T) {
	store := &gatedStore{memorySecretStore: newMemorySecretStore(), entered: make(chan struct{}), release: make(chan struct{}), deleted: make(chan struct{}, 2)}
	svc := NewServiceWithStore(t.TempDir()+"/metadata.json", map[string]Provider{"google": fakeValidator{result: domain.ProviderValidationResult{Valid: true}}}, store)
	if err := svc.InitializationError(); err != nil {
		t.Fatal(err)
	}
	store.gate = true
	saved, cleared := make(chan error, 1), make(chan error, 1)
	go func() { _, err := svc.ValidateAndSaveKey(context.Background(), "google", "new-secret"); saved <- err }()
	<-store.entered
	go func() { cleared <- svc.ClearKey("google") }()
	select {
	case <-store.deleted:
		t.Error("delete interleaved with save")
	case <-time.After(50 * time.Millisecond):
	}
	close(store.release)
	if err := <-saved; err != nil {
		t.Fatal(err)
	}
	if err := <-cleared; err != nil {
		t.Fatal(err)
	}
	if svc.GetStatus("google").Configured {
		t.Fatal("memory retained deleted key")
	}
	if _, err := store.Get("google"); !errors.Is(err, ErrSecretNotFound) {
		t.Fatal("keyring retained deleted key")
	}
}
