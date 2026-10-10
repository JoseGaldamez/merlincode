package domain

// ToolDefinition describe una herramienta disponible para el modelo, en un formato agnóstico
// de proveedor (cada adaptador de proveedor la traduce a su propio wire format).
type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// ToolCall representa una invocación de herramienta solicitada por el modelo.
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON crudo de los argumentos

	// ThoughtSignature es un dato opaco específico de Google Gemini: cuando el modelo invoca una
	// función, Gemini adjunta esta firma a la parte functionCall y exige que se le devuelva
	// exactamente igual al reenviar esa misma llamada en el historial de un turno posterior,
	// o rechaza la solicitud. Otros proveedores la dejan vacía.
	ThoughtSignature string `json:"thoughtSignature,omitempty"`
}

// ToolResult representa el resultado de ejecutar una ToolCall, listo para devolver al modelo.
type ToolResult struct {
	ToolCallID string `json:"toolCallId"`
	Name       string `json:"name"`
	Content    string `json:"content"`
	IsError    bool   `json:"isError,omitempty"`
}

// ToolApprovalRequest describe una escritura pendiente de aprobación explícita del usuario.
type ToolApprovalRequest struct {
	RequestID  string `json:"requestId"`
	ToolName   string `json:"toolName"`
	Path       string `json:"path"`
	NewContent string `json:"newContent"`
	OldContent string `json:"oldContent,omitempty"`
	ExpiresAt  int64  `json:"expiresAt"` // epoch ms — tras este momento el backend la rechaza automáticamente
}

// ToolTraceEntry es la forma persistida (independiente del wire format del proveedor) de una
// ejecución de herramienta dentro de un turno de agente, guardada como parte de un mensaje.
type ToolTraceEntry struct {
	ToolName  string `json:"toolName"`
	Arguments string `json:"arguments"`
	Result    string `json:"result"`
	IsError   bool   `json:"isError,omitempty"`
	Approved  *bool  `json:"approved,omitempty"`
}
