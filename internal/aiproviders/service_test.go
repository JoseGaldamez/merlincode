package aiproviders

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/zalando/go-keyring"

	"merlincode/internal/domain"
)

// TestMain reemplaza el llavero nativo del sistema operativo por uno en memoria durante toda la
// suite, para que las pruebas sean deterministas y no dependan de (ni contaminen) el Credential
// Manager/Keychain/Secret Service real de la máquina donde corren.
func TestMain(m *testing.M) {
	keyring.MockInit()
	os.Exit(m.Run())
}

// fakeValidator simula la respuesta de un proveedor real sin tocar la red,
// permitiendo probar el Service de forma determinista y aislada.
type fakeValidator struct {
	result domain.ProviderValidationResult
	err    error
}

func (f fakeValidator) Validate(ctx context.Context, apiKey string) (domain.ProviderValidationResult, error) {
	return f.result, f.err
}

// fakeUsageValidator simula un proveedor que además reporta uso/costo real de la cuenta,
// requiriendo una Admin API Key separada (como Anthropic), sin tocar la red.
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

func newTestService(t *testing.T, validators map[string]Validator) (*Service, string) {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "merlin_ai_providers_test_*")
	if err != nil {
		t.Fatalf("Error creando tempDir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tempDir) })

	filePath := filepath.Join(tempDir, "ai_providers.json")
	return NewServiceWithOptions(filePath, validators), filePath
}

func TestValidateAndSaveKey_ValidPersists(t *testing.T) {
	validators := map[string]Validator{
		"anthropic": fakeValidator{result: domain.ProviderValidationResult{
			Valid:       true,
			Message:     "ok",
			AccountInfo: "3 modelos disponibles",
		}},
	}
	svc, filePath := newTestService(t, validators)

	result, err := svc.ValidateAndSaveKey(context.Background(), "anthropic", "sk-ant-test-key-123456")
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if !result.Valid {
		t.Fatalf("esperado válido, obtenido inválido: %+v", result)
	}

	status := svc.GetStatus("anthropic")
	if !status.Configured || !status.Verified {
		t.Fatalf("esperado proveedor configurado y verificado, obtenido: %+v", status)
	}
	if status.MaskedKey == "sk-ant-test-key-123456" {
		t.Fatalf("la key completa no debe exponerse en el status: %+v", status)
	}
	if status.AccountInfo != "3 modelos disponibles" {
		t.Fatalf("accountInfo no persistido correctamente: %+v", status)
	}

	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("se esperaba que el archivo de credenciales existiera: %v", err)
	}

	// Recargar desde disco en una instancia nueva debe conservar el estado guardado
	reloaded := NewServiceWithOptions(filePath, validators)
	reloadedStatus := reloaded.GetStatus("anthropic")
	if !reloadedStatus.Verified {
		t.Fatalf("estado no persistido tras recarga: %+v", reloadedStatus)
	}
}

func TestValidateAndSaveKey_InvalidDoesNotPersist(t *testing.T) {
	validators := map[string]Validator{
		"openai": fakeValidator{result: domain.ProviderValidationResult{
			Valid:   false,
			Message: "Clave de API inválida o revocada.",
		}},
	}
	svc, _ := newTestService(t, validators)

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
}

func TestValidateAndSaveKey_UnknownProvider(t *testing.T) {
	svc, _ := newTestService(t, map[string]Validator{})

	_, err := svc.ValidateAndSaveKey(context.Background(), "unknown-provider", "some-key")
	if err != domain.ErrUnknownAIProvider {
		t.Fatalf("esperado ErrUnknownAIProvider, obtenido: %v", err)
	}
}

func TestClearKeyRemovesCredential(t *testing.T) {
	validators := map[string]Validator{
		"google": fakeValidator{result: domain.ProviderValidationResult{Valid: true, AccountInfo: "5 modelos"}},
	}
	svc, _ := newTestService(t, validators)

	if _, err := svc.ValidateAndSaveKey(context.Background(), "google", "AIzaSyTestKey"); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if !svc.GetStatus("google").Configured {
		t.Fatalf("se esperaba que la clave quedara configurada antes de limpiar")
	}

	svc.ClearKey("google")

	if svc.GetStatus("google").Configured {
		t.Fatalf("se esperaba que la clave quedara eliminada tras ClearKey")
	}
}

func TestSaveAdminKey_RejectedWhenProviderDoesNotSupportIt(t *testing.T) {
	validators := map[string]Validator{
		"deepseek": fakeValidator{result: domain.ProviderValidationResult{Valid: true}},
	}
	svc, _ := newTestService(t, validators)

	if err := svc.SaveAdminKey("deepseek", "some-admin-key"); err != domain.ErrAdminKeyNotSupported {
		t.Fatalf("esperado ErrAdminKeyNotSupported, obtenido: %v", err)
	}
}

func TestSaveAdminKeyAndFetchUsage_PassesKeysToValidator(t *testing.T) {
	usageValidator := &fakeUsageValidator{
		fakeValidator: fakeValidator{result: domain.ProviderValidationResult{Valid: true}},
		usageResult:   domain.ProviderUsageResult{Available: true, TokensPrompt: 100, TokensCompletion: 50, CostUsd: 1.23},
	}
	validators := map[string]Validator{"anthropic": usageValidator}
	svc, _ := newTestService(t, validators)

	if _, err := svc.ValidateAndSaveKey(context.Background(), "anthropic", "sk-ant-real-key"); err != nil {
		t.Fatalf("error inesperado al guardar la key: %v", err)
	}

	status := svc.GetStatus("anthropic")
	if !status.SupportsAdminKey {
		t.Fatalf("se esperaba SupportsAdminKey=true para anthropic")
	}
	if status.HasAdminKey {
		t.Fatalf("no se esperaba una admin key guardada todavía")
	}

	if err := svc.SaveAdminKey("anthropic", "admin-key-123"); err != nil {
		t.Fatalf("error inesperado guardando admin key: %v", err)
	}

	status = svc.GetStatus("anthropic")
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
	if usageValidator.gotAPIKey != "sk-ant-real-key" || usageValidator.gotAdminKey != "admin-key-123" {
		t.Fatalf("las keys no se propagaron correctamente al validador: apiKey=%q adminKey=%q", usageValidator.gotAPIKey, usageValidator.gotAdminKey)
	}

	svc.ClearAdminKey("anthropic")
	if svc.GetStatus("anthropic").HasAdminKey {
		t.Fatalf("se esperaba que la admin key quedara eliminada tras ClearAdminKey")
	}
}

func TestGetProviderUsage_UnsupportedProviderReturnsUnavailable(t *testing.T) {
	validators := map[string]Validator{
		"openai": fakeValidator{result: domain.ProviderValidationResult{Valid: true}},
	}
	svc, _ := newTestService(t, validators)

	if _, err := svc.ValidateAndSaveKey(context.Background(), "openai", "sk-openai-key"); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	result, err := svc.GetProviderUsage(context.Background(), "openai")
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if result.Available {
		t.Fatalf("se esperaba Available=false para un proveedor sin UsageFetcher")
	}
}

func TestListStatusesReturnsAllKnownProvidersInOrder(t *testing.T) {
	svc, _ := newTestService(t, map[string]Validator{})

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
