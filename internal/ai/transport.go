package ai

import "merlincode/internal/ai/transport"

// Constantes y funciones de transporte re-exportadas para uso del paquete ai
const DefaultMaxResponseBytes = transport.DefaultMaxResponseBytes

var (
	ErrResponseTooLarge    = transport.ErrResponseTooLarge
	DecodeJSONLimited      = transport.DecodeJSONLimited
	SanitizeTransportError = transport.SanitizeTransportError
	SafeProviderHTTPError  = transport.SafeProviderHTTPError
	CloseHTTPResponse      = transport.CloseHTTPResponse
)
