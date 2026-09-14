package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// DefaultMaxResponseBytes define el límite máximo del cuerpo de respuesta aceptado de un proveedor (2 MiB).
const DefaultMaxResponseBytes = 2 * 1024 * 1024

// ErrResponseTooLarge se retorna cuando un proveedor excede el límite máximo permitido de respuesta.
var ErrResponseTooLarge = errors.New("la respuesta del proveedor excedió el límite máximo permitido")

// CloseHTTPResponse libera la conexión cerrando el cuerpo de respuesta de forma segura.
func CloseHTTPResponse(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
}

// Client es el cliente HTTP seguro compartido con timeout estricto y bloqueo de fugas en redirección.
var Client = &http.Client{
	Timeout: 15 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("demasiadas redirecciones")
		}
		if len(via) > 0 {
			first := via[0]
			// Bloquear redirecciones a un host distinto o que degraden HTTPS
			if req.URL.Host != first.URL.Host || req.URL.Scheme != "https" {
				return errors.New("redirección externa o no segura bloqueada")
			}
			// Eliminar credenciales en redirección para prevenir fugas accidentales
			req.Header.Del("Authorization")
			req.Header.Del("x-api-key")
			req.Header.Del("x-goog-api-key")
		}
		return nil
	},
}

// StreamClient es el cliente HTTP seguro para streaming SSE prolongado.
// No tiene un timeout fijo global en el cliente; el ciclo de vida se gobierna estrictamente
// mediante el context.Context de cada petición (context.WithCancel / context.WithTimeout).
var StreamClient = &http.Client{
	Timeout:       0,
	CheckRedirect: Client.CheckRedirect,
}

// DecodeJSONLimited deserializa JSON desde un io.Reader limitando el tamaño máximo
// a maxBytes para evitar denegación de servicio por consumo de memoria.
func DecodeJSONLimited(r io.Reader, v any, maxBytes int64) error {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxResponseBytes
	}
	lr := io.LimitReader(r, maxBytes+1)
	data, err := io.ReadAll(lr)
	if err != nil {
		return err
	}
	if int64(len(data)) > maxBytes {
		return ErrResponseTooLarge
	}
	return json.Unmarshal(data, v)
}

// SanitizeTransportError clasifica errores de red y transporte en mensajes comprensibles
// y seguros para el usuario, sin exponer nunca URLs, parámetros ni información sensible.
func SanitizeTransportError(err error, providerName string) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return fmt.Sprintf("La conexión con %s excedió el tiempo de espera o fue cancelada.", providerName)
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return fmt.Sprintf("Tiempo de espera agotado al conectar con %s.", providerName)
	}
	errStr := strings.ToLower(err.Error())
	if strings.Contains(errStr, "no such host") || strings.Contains(errStr, "lookup") {
		return fmt.Sprintf("No se pudo resolver el nombre de host de %s. Verifica tu conexión a internet.", providerName)
	}
	if strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "connect: ") {
		return fmt.Sprintf("No se pudo establecer conexión con %s.", providerName)
	}
	if strings.Contains(errStr, "certificate") || strings.Contains(errStr, "tls") {
		return fmt.Sprintf("Fallo en la validación del certificado seguro (TLS) con %s.", providerName)
	}
	if strings.Contains(errStr, "bloqueada") || strings.Contains(errStr, "redirect") {
		return fmt.Sprintf("Redirección externa o no segura bloqueada al conectar con %s.", providerName)
	}
	if errors.Is(err, ErrResponseTooLarge) || strings.Contains(errStr, "límite máximo permitido") {
		return fmt.Sprintf("La respuesta recibida de %s es demasiado extensa.", providerName)
	}
	return fmt.Sprintf("Error de comunicación al conectar con %s. No se pudo completar la solicitud.", providerName)
}

// SafeProviderHTTPError traduce estados HTTP a mensajes públicos sin reenviar cuerpos controlados por terceros.
func SafeProviderHTTPError(providerName string, statusCode int) string {
	switch statusCode {
	case http.StatusBadRequest:
		return fmt.Sprintf("%s rechazó la solicitud de prueba.", providerName)
	case http.StatusUnauthorized, http.StatusForbidden:
		return "La clave de API no tiene autorización para realizar esta operación."
	case http.StatusTooManyRequests:
		return fmt.Sprintf("%s limitó temporalmente las solicitudes. Intenta de nuevo más tarde.", providerName)
	case http.StatusRequestTimeout, http.StatusGatewayTimeout:
		return fmt.Sprintf("%s excedió el tiempo de espera al procesar la solicitud.", providerName)
	default:
		if statusCode >= http.StatusInternalServerError {
			return fmt.Sprintf("%s no está disponible temporalmente.", providerName)
		}
		return fmt.Sprintf("%s respondió con estado inesperado (%d).", providerName, statusCode)
	}
}

// LimitedStreamReader caps the entire SSE body, including ignored events and heartbeats.
func LimitedStreamReader(r io.Reader) io.Reader {
	return &limitedStreamReader{reader: r, remaining: DefaultMaxResponseBytes}
}

type limitedStreamReader struct {
	reader    io.Reader
	remaining int
}

func (r *limitedStreamReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if r.remaining == 0 {
		var probe [1]byte
		n, err := r.reader.Read(probe[:])
		if n != 0 {
			return 0, ErrResponseTooLarge
		}
		return 0, err
	}
	if len(p) > r.remaining {
		p = p[:r.remaining]
	}
	n, err := r.reader.Read(p)
	r.remaining -= n
	return n, err
}
