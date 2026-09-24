package ai

import (
	"context"
	"strings"
	"sync"
	"time"

	"merlincode/internal/domain"
)

// Constantes de seguridad para limitar abusos y consumo accidental en mensajes de prueba
const (
	MaxTestMessageLength    = 4096 // 4 KiB
	MaxResponseTextLength   = 4096 // 4 KiB
	TestMessageCooldownTime = 2 * time.Second
)

type providerTestState struct {
	mu       sync.Mutex
	lastTest time.Time
}

func (s *Service) getProviderTestState(providerID string) *providerTestState {
	s.testStatesMu.Lock()
	defer s.testStatesMu.Unlock()
	state, exists := s.testStates[providerID]
	if !exists {
		state = &providerTestState{}
		s.testStates[providerID] = state
	}
	return state
}

// SendTestMessage envía un mensaje real de prueba validando modelo, tamaño, concurrencia y cooldown.
func (s *Service) SendTestMessage(ctx context.Context, providerID string, message string, model string) (domain.TestMessageResult, error) {
	if err := s.InitializationError(); err != nil {
		return domain.TestMessageResult{}, err
	}
	validator, ok := s.validators[providerID]
	if !ok {
		return domain.TestMessageResult{}, domain.ErrUnknownAIProvider
	}

	mt, ok := validator.(MessageTester)
	if !ok {
		return domain.TestMessageResult{
			Success: false,
			Message: "Este proveedor no admite el envío de mensajes de prueba desde la interfaz.",
		}, nil
	}

	// 1. Limitar tamaño del mensaje de prueba (máximo 4 KiB)
	if len(message) > MaxTestMessageLength {
		return domain.TestMessageResult{
			Success: false,
			Message: "El mensaje de prueba excede el límite máximo permitido de 4 KiB.",
		}, domain.ErrMessageTooLarge
	}

	s.mu.RLock()
	cred, hasCred := s.credentials[providerID]
	s.mu.RUnlock()

	if !hasCred || cred.APIKey == "" {
		return domain.TestMessageResult{Success: false, Message: "Configura y verifica una clave de API primero."}, nil
	}

	model = strings.TrimSpace(model)
	if model == "" {
		model = GetOrchestratorModel(providerID)
	}

	// 2. Validar que el modelo solicitado pertenece a los permitidos para este proveedor
	if !IsModelAllowedForProvider(providerID, model, cred.Models) {
		return domain.TestMessageResult{
			Success: false,
			Message: "El modelo seleccionado no está permitido o no es compatible con este proveedor.",
		}, domain.ErrInvalidModel
	}

	// 3. Control de concurrencia y cooldown por proveedor
	state := s.getProviderTestState(providerID)
	if !state.mu.TryLock() {
		return domain.TestMessageResult{
			Success: false,
			Message: "Ya hay una solicitud de prueba en curso para este proveedor. Espera a que finalice.",
		}, domain.ErrConcurrentRequestBlocked
	}
	defer state.mu.Unlock()

	if time.Since(state.lastTest) < TestMessageCooldownTime {
		return domain.TestMessageResult{
			Success: false,
			Message: "Espera unos segundos antes de enviar otro mensaje de prueba.",
		}, domain.ErrRateLimited
	}

	result, err := mt.SendTestMessage(ctx, cred.APIKey, message, model)
	state.lastTest = time.Now()

	// 4. Limitar tamaño de respuesta devuelta al frontend (máximo 4 KiB)
	if len(result.ResponseText) > MaxResponseTextLength {
		result.ResponseText = result.ResponseText[:MaxResponseTextLength] + "..."
	}

	return result, err
}
