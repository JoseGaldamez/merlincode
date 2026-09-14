package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/zalando/go-keyring"

	"merlincode/internal/ai/providers/google"
	"merlincode/internal/ai/transport"
	"merlincode/internal/domain"
)

func TestMain(m *testing.M) {
	keyring.MockInit()
	os.Exit(m.Run())
}

// memorySecretStore provee un almacén en memoria totalmente aislado por cada prueba
type memorySecretStore struct {
	mu           sync.RWMutex
	secrets      map[string]string
	deleteError  error
	deleteErrors map[string]error
	getError     error
	setError     error
}

func newMemorySecretStore() *memorySecretStore {
	return &memorySecretStore{
		secrets:      make(map[string]string),
		deleteErrors: make(map[string]error),
	}
}

func (m *memorySecretStore) Get(account string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.getError != nil {
		return "", m.getError
	}
	val, ok := m.secrets[account]
	if !ok {
		return "", ErrSecretNotFound
	}
	return val, nil
}

func (m *memorySecretStore) Set(account string, secret string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.setError != nil {
		return m.setError
	}
	m.secrets[account] = secret
	return nil
}

func (m *memorySecretStore) Delete(account string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.deleteError != nil {
		return m.deleteError
	}
	if err := m.deleteErrors[account]; err != nil {
		return err
	}
	if _, ok := m.secrets[account]; !ok {
		return ErrSecretNotFound
	}
	delete(m.secrets, account)
	return nil
}

type fakeValidator struct {
	result domain.ProviderValidationResult
	err    error
}

func (f fakeValidator) Validate(ctx context.Context, apiKey string) (domain.ProviderValidationResult, error) {
	return f.result, f.err
}

type fakeUsageValidator struct {
	fakeValidator
	usageResult domain.ProviderUsageResult
	usageErr    error
	gotAPIKey   string
	gotAdminKey string
}

func (f *fakeUsageValidator) RequiresAdminKeyForUsage() bool { return true }

func (f *fakeUsageValidator) FetchUsage(ctx context.Context, apiKey string, adminKey string) (domain.ProviderUsageResult, error) {
	f.gotAPIKey = apiKey
	f.gotAdminKey = adminKey
	return f.usageResult, f.usageErr
}

type fakeMessageTester struct {
	fakeValidator
	testResult domain.TestMessageResult
	testErr    error
	gotModel   string
	gotMsg     string
	started    chan struct{}
	release    chan struct{}
}

func (f *fakeMessageTester) SendTestMessage(ctx context.Context, apiKey string, message string, model string) (domain.TestMessageResult, error) {
	f.gotMsg = message
	f.gotModel = model
	if f.started != nil {
		select {
		case f.started <- struct{}{}:
		default:
		}
	}
	if f.release != nil {
		select {
		case <-f.release:
		case <-ctx.Done():
			return domain.TestMessageResult{}, ctx.Err()
		}
	}
	return f.testResult, f.testErr
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestService(t *testing.T, validators map[string]Provider) (*Service, string, *memorySecretStore) {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "merlin_ai_providers_test_*")
	if err != nil {
		t.Fatalf("Error creando tempDir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("no se pudo limpiar el directorio temporal: %v", err)
		}
	})

	filePath := filepath.Join(tempDir, "ai_providers.json")
	store := newMemorySecretStore()
	return NewServiceWithStore(filePath, validators, store), filePath, store
}

func TestValidateAndSaveKey_ValidPersists(t *testing.T) {
	validators := map[string]Provider{
		"anthropic": fakeValidator{result: domain.ProviderValidationResult{
			Valid:       true,
			AccountInfo: "5 modelos disponibles",
			Models:      []string{"claude-sonnet-5"},
		}},
	}
	svc, filePath, store := newTestService(t, validators)

	result, err := svc.ValidateAndSaveKey(context.Background(), "anthropic", "sk-ant-test-key")
	if err != nil {
		t.Fatalf("ValidateAndSaveKey falló: %v", err)
	}
	if !result.Valid {
		t.Fatalf("se esperaba resultado válido")
	}

	secret, err := store.Get("anthropic")
	if err != nil || secret != "sk-ant-test-key" {
		t.Fatalf("clave no guardada en el almacén de secretos: %v", err)
	}

	status := svc.GetStatus("anthropic")
	if !status.Configured || !status.Verified || status.MaskedKey != "sk-a••••••••-key" {
		t.Fatalf("estado incorrecto en memoria: %+v", status)
	}

	// Comprobar persistencia recargando el servicio desde disco
	reloaded := NewServiceWithStore(filePath, validators, store)
	reloadedStatus := reloaded.GetStatus("anthropic")
	if !reloadedStatus.Verified {
		t.Fatalf("estado no persistido tras recarga: %+v", reloadedStatus)
	}
}

func TestValidateAndSaveKey_InvalidDoesNotPersist(t *testing.T) {
	validators := map[string]Provider{
		"openai": fakeValidator{result: domain.ProviderValidationResult{
			Valid:   false,
			Message: "Clave de API inválida o revocada.",
		}},
	}
	svc, _, store := newTestService(t, validators)

	result, err := svc.ValidateAndSaveKey(context.Background(), "openai", "sk-invalid")
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if result.Valid {
		t.Fatalf("se esperaba resultado inválido")
	}

	status := svc.GetStatus("openai")
	if status.Configured {
		t.Fatalf("una clave inválida no debe quedar guardada como configurada: %+v", status)
	}

	if _, err := store.Get("openai"); err == nil {
		t.Fatalf("no se debía guardar clave inválida en el llavero")
	}
}

func TestValidateAndSaveKey_UnknownProvider(t *testing.T) {
	svc, _, _ := newTestService(t, map[string]Provider{})

	_, err := svc.ValidateAndSaveKey(context.Background(), "unknown-provider", "some-key")
	if err != domain.ErrUnknownAIProvider {
		t.Fatalf("esperado ErrUnknownAIProvider, obtenido: %v", err)
	}
}

func TestClearKeyRemovesCredential(t *testing.T) {
	validators := map[string]Provider{
		"google": fakeValidator{result: domain.ProviderValidationResult{Valid: true, AccountInfo: "5 modelos"}},
	}
	svc, _, store := newTestService(t, validators)

	if _, err := svc.ValidateAndSaveKey(context.Background(), "google", "AIzaSyTestKey"); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if !svc.GetStatus("google").Configured {
		t.Fatalf("se esperaba que la clave quedara configurada antes de limpiar")
	}

	if err := svc.ClearKey("google"); err != nil {
		t.Fatalf("error inesperado en ClearKey: %v", err)
	}

	if svc.GetStatus("google").Configured {
		t.Fatalf("se esperaba que la clave quedara eliminada tras ClearKey")
	}

	if _, err := store.Get("google"); err == nil {
		t.Fatalf("la clave aún existe en el almacén de secretos tras ClearKey")
	}
}

func TestClearKey_FailsGracefullyWhenKeyringFails(t *testing.T) {
	validators := map[string]Provider{
		"google": fakeValidator{result: domain.ProviderValidationResult{Valid: true}},
	}
	svc, _, store := newTestService(t, validators)

	if _, err := svc.ValidateAndSaveKey(context.Background(), "google", "AIzaSyTestKey"); err != nil {
		t.Fatalf("error: %v", err)
	}

	store.deleteError = errors.New("keyring locked or inaccessible")
	err := svc.ClearKey("google")
	if err == nil {
		t.Fatalf("se esperaba error cuando el llavero falla al eliminar")
	}

	// La metadata no debe borrarse si el secreto no pudo eliminarse
	if !svc.GetStatus("google").Configured {
		t.Fatalf("la metadata no debió eliminarse si el borrado del llavero falló")
	}
}

func TestSaveAdminKeyAndFetchUsage_PassesKeysToValidator(t *testing.T) {
	usageValidator := &fakeUsageValidator{
		fakeValidator: fakeValidator{result: domain.ProviderValidationResult{Valid: true}},
		usageResult:   domain.ProviderUsageResult{Available: true, TokensPrompt: 100, TokensCompletion: 50, CostUsd: 1.23},
	}
	validators := map[string]Provider{"anthropic": usageValidator}
	svc, _, _ := newTestService(t, validators)

	if _, err := svc.ValidateAndSaveKey(context.Background(), "anthropic", "sk-ant-real-key"); err != nil {
		t.Fatalf("error inesperado al guardar la key: %v", err)
	}

	if err := svc.SaveAdminKey("anthropic", "admin-key-123"); err != nil {
		t.Fatalf("error inesperado guardando admin key: %v", err)
	}

	status := svc.GetStatus("anthropic")
	if !status.HasAdminKey {
		t.Fatalf("se esperaba HasAdminKey=true tras guardar la admin key")
	}

	result, err := svc.GetProviderUsage(context.Background(), "anthropic")
	if err != nil {
		t.Fatalf("error inesperado obteniendo uso: %v", err)
	}
	if !result.Available || result.TokensPrompt != 100 || result.TokensCompletion != 50 {
		t.Fatalf("resultado de uso inesperado: %+v", result)
	}

	if err := svc.ClearAdminKey("anthropic"); err != nil {
		t.Fatalf("error en ClearAdminKey: %v", err)
	}
	if svc.GetStatus("anthropic").HasAdminKey {
		t.Fatalf("se esperaba que la admin key quedara eliminada")
	}
}

func TestListStatusesReturnsAllKnownProvidersInOrder(t *testing.T) {
	svc, _, _ := newTestService(t, map[string]Provider{})

	statuses := svc.ListStatuses()
	if len(statuses) != len(KnownProviderIDs) {
		t.Fatalf("esperados %d proveedores, obtenidos %d", len(KnownProviderIDs), len(statuses))
	}
	for i, id := range KnownProviderIDs {
		if statuses[i].ProviderID != id {
			t.Fatalf("orden inesperado en posición %d: esperado %s, obtenido %s", i, id, statuses[i].ProviderID)
		}
		if statuses[i].Configured {
			t.Fatalf("proveedor %s no debería estar configurado sin credencial guardada", id)
		}
	}
}

// Pruebas de seguridad específicas (Fases 1, 3 y 5)

func TestSecurity_ProviderCredentialJSONOmitsSecrets(t *testing.T) {
	cred := domain.ProviderCredential{
		ProviderID:  "anthropic",
		APIKey:      "SENTINEL_SECRET_API_KEY_12345",
		AdminAPIKey: "SENTINEL_SECRET_ADMIN_KEY_67890",
		Verified:    true,
	}

	data, err := json.Marshal(cred)
	if err != nil {
		t.Fatalf("error serializando credencial: %v", err)
	}

	output := string(data)
	if strings.Contains(output, "SENTINEL_SECRET_API_KEY") || strings.Contains(output, "SENTINEL_SECRET_ADMIN_KEY") {
		t.Fatalf("¡json.Marshal serializó credenciales secretas!: %s", output)
	}
}

func TestSecurity_DecodeJSONLimitedEnforcesSizeLimit(t *testing.T) {
	// Generar payload de 3 MiB
	largePayload := bytes.Repeat([]byte("a"), 3*1024*1024)
	var dummy map[string]any
	err := DecodeJSONLimited(bytes.NewReader(largePayload), &dummy, 2*1024*1024)
	if !errors.Is(err, ErrResponseTooLarge) {
		t.Fatalf("esperado ErrResponseTooLarge, obtenido: %v", err)
	}
}

func TestSecurity_CheckRedirectBlocksHostMismatch(t *testing.T) {
	// Crear servidor destino que simula redirección externa
	targetHit := make(chan struct{}, 1)
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetHit <- struct{}{}
		w.WriteHeader(http.StatusOK)
	}))
	defer targetServer.Close()

	initialServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, targetServer.URL, http.StatusFound)
	}))
	defer initialServer.Close()

	req, err := http.NewRequest(http.MethodGet, initialServer.URL, nil)
	if err != nil {
		t.Fatalf("error creando request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer SENTINEL_AUTH_TOKEN")
	req.Header.Set("x-api-key", "SENTINEL_API_KEY")

	_, err = transport.Client.Do(req)
	if err == nil {
		t.Fatalf("se esperaba que la redirección a otro host fuera bloqueada por CheckRedirect")
	}
	select {
	case <-targetHit:
		t.Fatal("la solicitud fue reenviada a un host externo no autorizado")
	default:
	}
}

func TestSecurity_SanitizeTransportErrorNeverLeakedSecret(t *testing.T) {
	cases := []struct {
		err  error
		desc string
	}{
		{err: errors.New("lookup api.anthropic.com: no such host"), desc: "DNS"},
		{err: errors.New("dial tcp: connect: connection refused"), desc: "Conexión rechazada"},
		{err: errors.New("tls: certificate signed by unknown authority"), desc: "TLS"},
		{err: context.DeadlineExceeded, desc: "Timeout"},
	}

	for _, tc := range cases {
		sanitized := SanitizeTransportError(tc.err, "Anthropic")
		if sanitized == "" {
			t.Errorf("SanitizeTransportError retornó cadena vacía para caso %s", tc.desc)
		}
		if strings.Contains(sanitized, "api.anthropic.com") || strings.Contains(sanitized, "tcp:") {
			t.Errorf("SanitizeTransportError expuso detalles del transporte en caso %s: %s", tc.desc, sanitized)
		}
	}
}

func TestSecurity_SendTestMessageLimitsAndValidation(t *testing.T) {
	mockTester := &fakeMessageTester{
		fakeValidator: fakeValidator{result: domain.ProviderValidationResult{Valid: true}},
		testResult:    domain.TestMessageResult{Success: true, ResponseText: "pong"},
	}
	validators := map[string]Provider{"anthropic": mockTester}
	svc, _, _ := newTestService(t, validators)

	// Guardar clave
	if _, err := svc.ValidateAndSaveKey(context.Background(), "anthropic", "sk-ant-test"); err != nil {
		t.Fatalf("error: %v", err)
	}

	// 1. Rechazo de modelo no autorizado
	_, err := svc.SendTestMessage(context.Background(), "anthropic", "hola", "unauthorized-model-xyz")
	if !errors.Is(err, domain.ErrInvalidModel) {
		t.Fatalf("esperado ErrInvalidModel, obtenido: %v", err)
	}

	// 2. Rechazo de mensaje excesivo (> 4 KiB)
	hugeMsg := strings.Repeat("x", 5000)
	_, err = svc.SendTestMessage(context.Background(), "anthropic", hugeMsg, "claude-sonnet-5")
	if !errors.Is(err, domain.ErrMessageTooLarge) {
		t.Fatalf("esperado ErrMessageTooLarge, obtenido: %v", err)
	}

	// 3. Envío exitoso con modelo permitido
	res, err := svc.SendTestMessage(context.Background(), "anthropic", "hola", "claude-sonnet-5")
	if err != nil || !res.Success {
		t.Fatalf("error inesperado en envío válido: err=%v, res=%+v", err, res)
	}

	// 4. Cooldown (enviar inmediatamente debe ser bloqueado por RateLimit)
	_, err = svc.SendTestMessage(context.Background(), "anthropic", "hola de nuevo", "claude-sonnet-5")
	if !errors.Is(err, domain.ErrRateLimited) {
		t.Fatalf("esperado ErrRateLimited dentro del periodo de cooldown, obtenido: %v", err)
	}
}

func TestSecurity_OrphanedSecretDetection(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "merlin_orphan_test_*")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(tempDir)
	})

	filePath := filepath.Join(tempDir, "ai_providers.json")
	store := newMemorySecretStore()
	_ = store.Set("anthropic", "sk-orphan-key")

	svc := NewServiceWithStore(filePath, map[string]Provider{}, store)
	status := svc.GetStatus("anthropic")
	if !status.Configured || !status.Verified {
		t.Fatalf("un secreto huérfano en el llavero debe detectarse y sincronizarse: %+v", status)
	}
}

func TestSecurity_GoogleCredentialsOnlyUseHeaders(t *testing.T) {
	sentinel := "AIzaSySENTINEL_HEADER_ONLY_12345"
	type capturedRequest struct {
		url    string
		header http.Header
	}
	var captured []capturedRequest
	originalClient := transport.Client
	transport.Client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		captured = append(captured, capturedRequest{url: req.URL.String(), header: req.Header.Clone()})
		body := `{"models":[]}`
		if req.Method == http.MethodPost {
			body = `{"candidates":[{"content":{"parts":[{"text":"ok"}]}}]}`
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil
	})}
	t.Cleanup(func() { transport.Client = originalClient })

	validator := google.Client{}
	if _, err := validator.Validate(context.Background(), sentinel); err != nil {
		t.Fatalf("Validate retornó error: %v", err)
	}
	if _, err := validator.SendTestMessage(context.Background(), sentinel, "hola", "gemini-1.5-flash"); err != nil {
		t.Fatalf("SendTestMessage retornó error: %v", err)
	}

	if len(captured) != 2 {
		t.Fatalf("se esperaban 2 peticiones, se capturaron %d", len(captured))
	}
	for _, request := range captured {
		if strings.Contains(request.url, sentinel) || strings.Contains(request.url, "key=") {
			t.Fatalf("la URL contiene credenciales: %s", request.url)
		}
		if got := request.header.Get("x-goog-api-key"); got != sentinel {
			t.Fatalf("encabezado x-goog-api-key inesperado: %q", got)
		}
	}
}

func TestSecurity_ProviderErrorBodyIsNeverForwarded(t *testing.T) {
	sentinel := "SENTINEL_UPSTREAM_ERROR https://private.example.invalid/token"
	originalClient := transport.Client
	transport.Client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := fmt.Sprintf(`{"error":{"message":%q}}`, sentinel)
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil
	})}
	t.Cleanup(func() { transport.Client = originalClient })

	result, err := (google.Client{}).SendTestMessage(context.Background(), "AIzaSy-safe", "hola", "gemini-1.5-flash")
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if strings.Contains(result.Message, "SENTINEL") || strings.Contains(result.Message, "private.example") {
		t.Fatalf("el mensaje público expuso el cuerpo del proveedor: %q", result.Message)
	}
}

func TestSecurity_LegacyMigrationIsVerifiedAndSanitized(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "ai_providers.json")
	apiSentinel := "SENTINEL_LEGACY_API_KEY"
	adminSentinel := "SENTINEL_LEGACY_ADMIN_KEY"
	legacy := fmt.Sprintf(`{"anthropic":{"providerId":"anthropic","apiKey":%q,"adminApiKey":%q,"verified":true,"accountInfo":"Saldo: 99 USD","models":["claude-3-5-sonnet-latest"]}}`, apiSentinel, adminSentinel)
	if err := os.WriteFile(filePath, []byte(legacy), 0644); err != nil {
		t.Fatalf("no se pudo preparar metadata heredada: %v", err)
	}

	store := newMemorySecretStore()
	svc := NewServiceWithStore(filePath, map[string]Provider{}, store)
	if err := svc.InitializationError(); err != nil {
		t.Fatalf("error en inicialización/migración: %v", err)
	}

	// 1. Los secretos deben estar en el llavero
	if got, err := store.Get("anthropic"); err != nil || got != apiSentinel {
		t.Fatalf("el secreto principal no se migró al llavero: got=%q, err=%v", got, err)
	}
	if got, err := store.Get(adminKeyringAccount("anthropic")); err != nil || got != adminSentinel {
		t.Fatalf("el secreto admin no se migró al llavero: got=%q, err=%v", got, err)
	}

	// 2. El archivo en disco debe estar sanitizado (sin claves ni información financiera)
	migratedData, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("error leyendo metadata migrada: %v", err)
	}
	content := string(migratedData)
	if strings.Contains(content, apiSentinel) || strings.Contains(content, adminSentinel) {
		t.Fatalf("el archivo en disco aún contiene secretos tras la migración: %s", content)
	}
	if strings.Contains(content, "Saldo") || strings.Contains(content, "accountInfo") {
		t.Fatalf("el archivo en disco conservó información financiera o accountInfo: %s", content)
	}

	// 3. Permisos en disco restringidos (0600 en Linux/macOS)
	if runtime.GOOS != "windows" {
		info, err := os.Stat(filePath)
		if err != nil {
			t.Fatalf("error inspeccionando archivo: %v", err)
		}
		if info.Mode().Perm() != 0600 {
			t.Fatalf("permisos inseguros tras migración: %v", info.Mode().Perm())
		}
	}
}

func TestSecurity_LegacyMigrationFailureKeepsOnlyCopy(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "ai_providers.json")
	apiSentinel := "SENTINEL_UNMIGRATED_KEY"
	legacy := fmt.Sprintf(`{"openai":{"providerId":"openai","apiKey":%q,"verified":true}}`, apiSentinel)
	if err := os.WriteFile(filePath, []byte(legacy), 0644); err != nil {
		t.Fatalf("error preparando archivo: %v", err)
	}

	store := newMemorySecretStore()
	store.setError = errors.New("keyring write blocked")

	svc := NewServiceWithStore(filePath, map[string]Provider{}, store)
	if svc.InitializationError() == nil {
		t.Fatal("se esperaba que la falla en la migración reportara error de inicialización")
	}

	// La clave no debió quedar guardada en el llavero
	if _, err := store.Get("openai"); !errors.Is(err, ErrSecretNotFound) {
		t.Fatalf("el secreto no debió quedar en el llavero tras fallo: %v", err)
	}
}

func TestSecurity_KeyringReadFailureIsPropagated(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "ai_providers.json")
	if err := os.WriteFile(filePath, []byte(`{"anthropic":{"providerId":"anthropic","verified":true}}`), 0600); err != nil {
		t.Fatalf("error preparando archivo: %v", err)
	}

	store := newMemorySecretStore()
	store.getError = errors.New("keyring unavailable")

	svc := NewServiceWithStore(filePath, map[string]Provider{}, store)
	if svc.InitializationError() == nil {
		t.Fatal("se esperaba que el fallo de lectura del llavero se propagara como error de inicialización")
	}
}

func TestSecurity_MetadataFailureRollsBackSecretAndState(t *testing.T) {
	svc, _, store := newTestService(t, map[string]Provider{"anthropic": fakeValidator{result: domain.ProviderValidationResult{Valid: true}}})
	svc.filePath = "" // Provocar fallo de guardado en disco

	_, err := svc.ValidateAndSaveKey(context.Background(), "anthropic", "sk-ant-roll-back")
	if err == nil {
		t.Fatal("se esperaba error de persistencia de metadata")
	}

	// El secreto debe haber sido revertido
	if _, err := store.Get("anthropic"); !errors.Is(err, ErrSecretNotFound) {
		t.Fatalf("el secreto debía haber sido eliminado tras rollback: %v", err)
	}

	// La memoria no debe contener la credencial
	if svc.GetStatus("anthropic").Configured {
		t.Fatal("el estado en memoria debió revertirse")
	}
}

func TestSecurity_OrphanedAdminSecretCanBeDetectedAndDeleted(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "ai_providers.json")
	store := newMemorySecretStore()
	account := adminKeyringAccount("anthropic")
	_ = store.Set(account, "sk-ant-admin-orphan")

	svc := NewServiceWithStore(filePath, map[string]Provider{}, store)
	status := svc.GetStatus("anthropic")
	if !status.HasAdminKey {
		t.Fatalf("se esperaba HasAdminKey=true para el secreto admin huérfano: %+v", status)
	}

	if err := svc.ClearAdminKey("anthropic"); err != nil {
		t.Fatalf("ClearAdminKey falló: %v", err)
	}
	if _, err := store.Get(account); !errors.Is(err, ErrSecretNotFound) {
		t.Fatalf("el secreto admin debía eliminarse del llavero: %v", err)
	}
}

func TestSecurity_ClearKeyTreatsNotFoundAsIdempotent(t *testing.T) {
	svc, _, _ := newTestService(t, map[string]Provider{})
	// Intentar borrar clave no existente en llavero debe ser idempotente y no fallar
	if err := svc.ClearKey("anthropic"); err != nil {
		t.Fatalf("ClearKey debe ser idempotente ante cuenta inexistente: %v", err)
	}
}

func TestSecurity_PartialDeleteReflectsActualKeyringState(t *testing.T) {
	svc, _, store := newTestService(t, map[string]Provider{"anthropic": fakeValidator{result: domain.ProviderValidationResult{Valid: true}}})
	if _, err := svc.ValidateAndSaveKey(context.Background(), "anthropic", "sk-ant-main"); err != nil {
		t.Fatalf("ValidateAndSaveKey falló: %v", err)
	}
	_ = store.Set(adminKeyringAccount("anthropic"), "sk-ant-admin")
	_ = svc.hydrateFromKeyring()

	// Simular fallo solo al borrar la admin key
	store.deleteErrors[adminKeyringAccount("anthropic")] = errors.New("admin delete blocked")

	err := svc.ClearKey("anthropic")
	if err == nil {
		t.Fatal("se esperaba error en ClearKey ante fallo parcial del llavero")
	}

	// La clave principal sí se borró del llavero
	if _, err := store.Get("anthropic"); !errors.Is(err, ErrSecretNotFound) {
		t.Fatalf("la clave principal debía estar eliminada: %v", err)
	}
	// La admin key no se borró
	if _, err := store.Get(adminKeyringAccount("anthropic")); err != nil {
		t.Fatalf("la admin key debía seguir en el llavero: %v", err)
	}

	// El estado en memoria debe reflejar fielmente lo que quedó en el llavero
	status := svc.GetStatus("anthropic")
	if status.Configured {
		t.Fatal("Configured debe ser false ya que la clave principal fue eliminada")
	}
	if !status.HasAdminKey {
		t.Fatal("HasAdminKey debe seguir siendo true porque la admin key no pudo eliminarse")
	}
}

func TestSecurity_MetadataNeverPersistsFinancialAccountInfo(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "ai_providers.json")
	store := newMemorySecretStore()
	svc := NewServiceWithStore(filePath, map[string]Provider{
		"deepseek": fakeValidator{result: domain.ProviderValidationResult{
			Valid:       true,
			AccountInfo: "Saldo disponible: 15.50 USD",
			Models:      []string{"deepseek-chat"},
		}},
	}, store)

	_, err := svc.ValidateAndSaveKey(context.Background(), "deepseek", "sk-ds-financial")
	if err != nil {
		t.Fatalf("ValidateAndSaveKey falló: %v", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("error leyendo archivo: %v", err)
	}
	content := string(data)
	if strings.Contains(content, "Saldo") || strings.Contains(content, "15.50") || strings.Contains(content, "accountInfo") {
		t.Fatalf("¡Se detectó información financiera persistida en disco!: %s", content)
	}
}

func TestSecurity_TestMessageConcurrencyOutputAndCancellation(t *testing.T) {
	started := make(chan struct{}, 1)
	release := make(chan struct{})

	hugeOutput := strings.Repeat("a", 10000)
	slowTester := &fakeMessageTester{
		fakeValidator: fakeValidator{result: domain.ProviderValidationResult{Valid: true}},
		testResult:    domain.TestMessageResult{Success: true, ResponseText: hugeOutput, Model: "claude-sonnet-5"},
		started:       started,
		release:       release,
	}

	svc, _, _ := newTestService(t, map[string]Provider{"anthropic": slowTester})
	if _, err := svc.ValidateAndSaveKey(context.Background(), "anthropic", "sk-ant-concurrency"); err != nil {
		t.Fatalf("ValidateAndSaveKey falló: %v", err)
	}

	t.Run("concurrent_request_is_rejected", func(t *testing.T) {
		errCh := make(chan error, 1)
		go func() {
			_, err := svc.SendTestMessage(context.Background(), "anthropic", "mensaje 1", "claude-sonnet-5")
			errCh <- err
		}()

		// Esperar a que la primera llamada esté activa
		<-started

		// Segunda llamada concurrente debe ser rechazada de inmediato
		_, err := svc.SendTestMessage(context.Background(), "anthropic", "mensaje 2", "claude-sonnet-5")
		if !errors.Is(err, domain.ErrConcurrentRequestBlocked) {
			t.Fatalf("se esperaba ErrConcurrentRequestBlocked, obtenido: %v", err)
		}

		// Liberar la primera llamada
		close(release)
		if firstErr := <-errCh; firstErr != nil {
			t.Fatalf("primera llamada falló inesperadamente: %v", firstErr)
		}
	})

	t.Run("output_is_truncated", func(t *testing.T) {
		// Esperar que termine el cooldown
		time.Sleep(TestMessageCooldownTime + 10*time.Millisecond)

		fastTester := &fakeMessageTester{
			fakeValidator: fakeValidator{result: domain.ProviderValidationResult{Valid: true}},
			testResult:    domain.TestMessageResult{Success: true, ResponseText: hugeOutput, Model: "claude-sonnet-5"},
		}
		fastSvc, _, _ := newTestService(t, map[string]Provider{"anthropic": fastTester})
		_ = fastSvc.setSecretVerified("anthropic", "sk-ant-fast")
		_ = fastSvc.hydrateFromKeyring()

		result, err := fastSvc.SendTestMessage(context.Background(), "anthropic", "test", "claude-sonnet-5")
		if err != nil {
			t.Fatalf("SendTestMessage falló: %v", err)
		}
		if len(result.ResponseText) > MaxResponseTextLength+3 {
			t.Fatalf("la respuesta devuelta excede el límite truncado: len=%d", len(result.ResponseText))
		}
		if !strings.HasSuffix(result.ResponseText, "...") {
			t.Fatalf("se esperaba el sufijo '...' indicando truncamiento")
		}
	})

	t.Run("context_cancellation_reaches_provider", func(t *testing.T) {
		time.Sleep(TestMessageCooldownTime + 10*time.Millisecond)

		cancelCtx, cancel := context.WithCancel(context.Background())
		blockingTester := &fakeMessageTester{
			fakeValidator: fakeValidator{result: domain.ProviderValidationResult{Valid: true}},
			release:       make(chan struct{}), // nunca liberado para probar cancelación
		}
		cancellingSvc, _, _ := newTestService(t, map[string]Provider{"anthropic": blockingTester})
		_ = cancellingSvc.setSecretVerified("anthropic", "sk-ant-cancel")
		_ = cancellingSvc.hydrateFromKeyring()

		errCh := make(chan error, 1)
		go func() {
			_, err := cancellingSvc.SendTestMessage(cancelCtx, "anthropic", "cancel-me", "claude-sonnet-5")
			errCh <- err
		}()

		time.Sleep(10 * time.Millisecond)
		cancel()

		err := <-errCh
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("esperado context.Canceled, obtenido: %v", err)
		}
	})
}
