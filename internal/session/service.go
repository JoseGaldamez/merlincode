package session

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"merlincode/internal/domain"
)

// Service gestiona la persistencia de sesiones y mensajes de chat en una base de datos local SQLite.
type Service struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewService inicializa la base de datos SQLite en la ruta provista, configurando pragmas y el esquema.
func NewService(dbPath string) (*Service, error) {
	if strings.TrimSpace(dbPath) == "" {
		return nil, errors.New("la ruta de la base de datos no puede estar vacía")
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("error al abrir la base de datos SQLite: %w", err)
	}

	// Pragmas de rendimiento, integridad y concurrencia segura
	pragmas := []string{
		"PRAGMA journal_mode = WAL;",
		"PRAGMA foreign_keys = ON;",
		"PRAGMA busy_timeout = 5000;",
		"PRAGMA synchronous = NORMAL;",
	}
	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("error al ejecutar pragma '%s': %w", pragma, err)
		}
	}

	s := &Service{db: db}
	if err := s.initSchema(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("error al inicializar el esquema de la base de datos: %w", err)
	}

	return s, nil
}

// Close cierra la conexión subyacente con la base de datos SQLite.
func (s *Service) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *Service) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		project_id TEXT NOT NULL DEFAULT '',
		project_path TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS messages (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
		role TEXT NOT NULL CHECK(role IN ('user', 'assistant', 'system')),
		content TEXT NOT NULL DEFAULT '',
		thought_chain TEXT NOT NULL DEFAULT '',
		provider_id TEXT NOT NULL DEFAULT '',
		model_id TEXT NOT NULL DEFAULT '',
		tokens_prompt INTEGER NOT NULL DEFAULT 0,
		tokens_completion INTEGER NOT NULL DEFAULT 0,
		duration_seconds INTEGER NOT NULL DEFAULT 0,
		feedback TEXT DEFAULT NULL CHECK(feedback IN ('like', 'dislike') OR feedback IS NULL),
		status TEXT NOT NULL DEFAULT 'done',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS message_usage (
		message_id TEXT PRIMARY KEY,
		provider_id TEXT NOT NULL,
		completed_at INTEGER NOT NULL,
		tokens_prompt INTEGER NOT NULL,
		tokens_completion INTEGER NOT NULL,
		cost_nanos_usd INTEGER,
		cost_partial INTEGER NOT NULL,
		requests INTEGER NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_messages_session_id ON messages(session_id, created_at ASC);
	CREATE INDEX IF NOT EXISTS idx_sessions_updated_at ON sessions(updated_at DESC);
	CREATE INDEX IF NOT EXISTS idx_message_usage_day ON message_usage(provider_id, completed_at);
	`
	if _, err := s.db.Exec(schema); err != nil {
		return err
	}
	return s.ensureToolTraceColumn()
}

// ensureToolTraceColumn agrega la columna tool_trace a bases de datos creadas antes de que existiera,
// sin tocar el resto del esquema ni ampliar el CHECK de role.
func (s *Service) ensureToolTraceColumn() error {
	rows, err := s.db.Query(`PRAGMA table_info(messages);`)
	if err != nil {
		return fmt.Errorf("error al inspeccionar el esquema de messages: %w", err)
	}
	defer rows.Close()

	hasColumn := false
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dflt, &pk); err != nil {
			return fmt.Errorf("error al leer columna de messages: %w", err)
		}
		if name == "tool_trace" {
			hasColumn = true
			break
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterando columnas de messages: %w", err)
	}
	if hasColumn {
		return nil
	}

	_, err = s.db.Exec(`ALTER TABLE messages ADD COLUMN tool_trace TEXT NOT NULL DEFAULT '';`)
	if err != nil {
		return fmt.Errorf("error al agregar columna tool_trace: %w", err)
	}
	return nil
}

// ListSessions retorna todas las sesiones ordenadas por última actualización descendente.
func (s *Service) ListSessions(ctx context.Context) ([]domain.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
	SELECT s.id, s.title, s.project_id, s.project_path,
	       COUNT(m.id) as messages_count, s.created_at, s.updated_at
	FROM sessions s
	LEFT JOIN messages m ON s.id = m.session_id
	GROUP BY s.id
	ORDER BY s.updated_at DESC;
	`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error al listar sesiones: %w", err)
	}
	defer rows.Close()

	var sessions []domain.Session
	for rows.Next() {
		var sess domain.Session
		if err := rows.Scan(
			&sess.ID,
			&sess.Title,
			&sess.ProjectID,
			&sess.ProjectPath,
			&sess.MessagesCount,
			&sess.CreatedAt,
			&sess.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("error al leer fila de sesión: %w", err)
		}
		sessions = append(sessions, sess)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterando sesiones: %w", err)
	}

	if sessions == nil {
		sessions = []domain.Session{}
	}

	return sessions, nil
}

// CreateSession inserta una nueva sesión en la base de datos.
func (s *Service) CreateSession(ctx context.Context, id, title, projectID, projectPath string) (domain.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id = strings.TrimSpace(id)
	if id == "" {
		id = fmt.Sprintf("s-%d", time.Now().UnixMilli())
	}
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Nueva sesión"
	}

	now := time.Now().UTC()
	query := `
	INSERT INTO sessions (id, title, project_id, project_path, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?);
	`
	_, err := s.db.ExecContext(ctx, query, id, title, projectID, projectPath, now, now)
	if err != nil {
		return domain.Session{}, fmt.Errorf("error al crear sesión: %w", err)
	}

	return domain.Session{
		ID:            id,
		Title:         title,
		ProjectID:     projectID,
		ProjectPath:   projectPath,
		MessagesCount: 0,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

// EnsureSessionExists crea una sesión básica si aún no existe, o actualiza su timestamp.
func (s *Service) EnsureSessionExists(ctx context.Context, id, title, projectID, projectPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("el ID de sesión no puede estar vacío")
	}
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Conversación"
	}

	now := time.Now().UTC()
	query := `
	INSERT INTO sessions (id, title, project_id, project_path, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET updated_at = excluded.updated_at;
	`
	_, err := s.db.ExecContext(ctx, query, id, title, projectID, projectPath, now, now)
	return err
}

// UpdateSessionTitle actualiza el título de una sesión existente.
func (s *Service) UpdateSessionTitle(ctx context.Context, id, title string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id = strings.TrimSpace(id)
	title = strings.TrimSpace(title)
	if id == "" {
		return errors.New("el ID de sesión no puede estar vacío")
	}
	if title == "" {
		return errors.New("el título de la sesión no puede estar vacío")
	}

	query := `UPDATE sessions SET title = ?, updated_at = ? WHERE id = ?;`
	res, err := s.db.ExecContext(ctx, query, title, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("error al actualizar título de sesión: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return errors.New("sesión no encontrada")
	}
	return nil
}

// DeleteSession elimina una sesión y en cascada todos sus mensajes asociados.
func (s *Service) DeleteSession(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("el ID de sesión no puede estar vacío")
	}

	query := `DELETE FROM sessions WHERE id = ?;`
	_, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("error al eliminar sesión: %w", err)
	}
	return nil
}

// GetSessionMessages retorna todos los mensajes de una sesión en orden cronológico ascendente.
func (s *Service) GetSessionMessages(ctx context.Context, sessionID string) ([]domain.ChatMessageRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return []domain.ChatMessageRecord{}, nil
	}

	query := `
	SELECT m.id, m.session_id, m.role, m.content, m.thought_chain, m.provider_id, m.model_id,
	       m.tokens_prompt, m.tokens_completion, m.duration_seconds, m.feedback, m.status, m.created_at, m.tool_trace,
	       u.cost_nanos_usd, u.cost_partial, u.requests
	FROM messages m LEFT JOIN message_usage u ON u.message_id = m.id
	WHERE m.session_id = ?
	ORDER BY m.created_at ASC;
	`
	rows, err := s.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("error al obtener mensajes de sesión: %w", err)
	}
	defer rows.Close()

	var messages []domain.ChatMessageRecord
	for rows.Next() {
		var m domain.ChatMessageRecord
		var roleStr string
		var fb sql.NullString
		var toolTraceJSON string
		var amount, requests sql.NullInt64
		var partial sql.NullBool

		if err := rows.Scan(
			&m.ID,
			&m.SessionID,
			&roleStr,
			&m.Content,
			&m.ThoughtChain,
			&m.ProviderID,
			&m.ModelID,
			&m.TokensPrompt,
			&m.TokensCompletion,
			&m.DurationSeconds,
			&fb,
			&m.Status,
			&m.CreatedAt,
			&toolTraceJSON,
			&amount,
			&partial,
			&requests,
		); err != nil {
			return nil, fmt.Errorf("error al leer mensaje de base de datos: %w", err)
		}

		m.Role = domain.ChatRole(roleStr)
		if requests.Valid {
			m.Cost = &domain.TurnCost{Requests: int(requests.Int64), Partial: partial.Bool}
			if amount.Valid {
				m.Cost.NanosUSD = &amount.Int64
			}
		}
		if fb.Valid {
			m.Feedback = &fb.String
		}
		if toolTraceJSON != "" {
			if err := json.Unmarshal([]byte(toolTraceJSON), &m.ToolTrace); err != nil {
				return nil, fmt.Errorf("error al decodificar tool_trace del mensaje %s: %w", m.ID, err)
			}
		}

		messages = append(messages, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterando mensajes: %w", err)
	}

	if messages == nil {
		messages = []domain.ChatMessageRecord{}
	}

	return messages, nil
}

// SaveMessage inserta o actualiza un mensaje de chat y refresca el timestamp de la sesión correspondiente.
func (s *Service) SaveMessage(ctx context.Context, msg domain.ChatMessageRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	msg.ID = strings.TrimSpace(msg.ID)
	msg.SessionID = strings.TrimSpace(msg.SessionID)
	if msg.ID == "" {
		return errors.New("el ID del mensaje no puede estar vacío")
	}
	if msg.SessionID == "" {
		return errors.New("el ID de sesión no puede estar vacío")
	}
	if msg.Role == "" {
		msg.Role = domain.ChatRoleUser
	}
	if msg.Status == "" {
		msg.Status = "done"
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now().UTC()
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Asegurar que la sesión exista
	now := time.Now().UTC()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO sessions (id, title, created_at, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET updated_at = excluded.updated_at;
	`, msg.SessionID, "Conversación", now, now)
	if err != nil {
		return fmt.Errorf("error al actualizar la sesión: %w", err)
	}

	var fb sql.NullString
	if msg.Feedback != nil && *msg.Feedback != "" {
		fb = sql.NullString{String: *msg.Feedback, Valid: true}
	}

	toolTraceJSON := ""
	if len(msg.ToolTrace) > 0 {
		data, err := json.Marshal(msg.ToolTrace)
		if err != nil {
			return fmt.Errorf("error al codificar tool_trace del mensaje: %w", err)
		}
		toolTraceJSON = string(data)
	}

	query := `
	INSERT INTO messages (
		id, session_id, role, content, thought_chain, provider_id, model_id,
		tokens_prompt, tokens_completion, duration_seconds, feedback, status, created_at, tool_trace
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		content = excluded.content,
		thought_chain = excluded.thought_chain,
		provider_id = excluded.provider_id,
		model_id = excluded.model_id,
		tokens_prompt = excluded.tokens_prompt,
		tokens_completion = excluded.tokens_completion,
		duration_seconds = excluded.duration_seconds,
		feedback = COALESCE(excluded.feedback, messages.feedback),
		status = excluded.status,
		tool_trace = excluded.tool_trace;
	`
	_, err = tx.ExecContext(ctx, query,
		msg.ID,
		msg.SessionID,
		string(msg.Role),
		msg.Content,
		msg.ThoughtChain,
		msg.ProviderID,
		msg.ModelID,
		msg.TokensPrompt,
		msg.TokensCompletion,
		msg.DurationSeconds,
		fb,
		msg.Status,
		msg.CreatedAt,
		toolTraceJSON,
	)
	if err != nil {
		return fmt.Errorf("error al guardar mensaje en base de datos: %w", err)
	}

	if msg.Role == domain.ChatRoleAssistant && msg.Cost != nil {
		// Cost totals survive deletion of a conversation. This entry contains no message content.
		_, err = tx.ExecContext(ctx, `
			INSERT INTO message_usage (
				message_id, provider_id, completed_at, tokens_prompt, tokens_completion, cost_nanos_usd, cost_partial, requests
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(message_id) DO NOTHING;
		`, msg.ID, msg.ProviderID, msg.CreatedAt.Unix(), msg.TokensPrompt, msg.TokensCompletion,
			msg.Cost.NanosUSD, msg.Cost.Partial, msg.Cost.Requests)
		if err != nil {
			return fmt.Errorf("error al guardar el consumo del turno: %w", err)
		}
	}
	return tx.Commit()
}

func (s *Service) GetTodayAIUsage(ctx context.Context, providerID string, now time.Time) (domain.DailyProviderUsage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	end := start.AddDate(0, 0, 1)
	var result domain.DailyProviderUsage
	var amount sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(tokens_prompt), 0), COALESCE(SUM(tokens_completion), 0),
		       SUM(cost_nanos_usd), COALESCE(MAX(cost_partial), 0), COALESCE(SUM(requests), 0)
		FROM message_usage
		WHERE provider_id = ? AND completed_at >= ? AND completed_at < ?;
	`, providerID, start.Unix(), end.Unix()).Scan(
		&result.TokensPrompt, &result.TokensCompletion, &amount, &result.Cost.Partial, &result.Cost.Requests)
	if err != nil {
		return domain.DailyProviderUsage{}, fmt.Errorf("error al consultar el consumo local de hoy: %w", err)
	}
	if amount.Valid {
		result.Cost.NanosUSD = &amount.Int64
	}
	return result, nil
}

// UpdateMessageFeedback guarda o elimina la calificación ("like" o "dislike") de un mensaje específico.
func (s *Service) UpdateMessageFeedback(ctx context.Context, messageID string, feedback *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	messageID = strings.TrimSpace(messageID)
	if messageID == "" {
		return errors.New("el ID del mensaje no puede estar vacío")
	}

	var fb sql.NullString
	if feedback != nil && (*feedback == "like" || *feedback == "dislike") {
		fb = sql.NullString{String: *feedback, Valid: true}
	}

	query := `UPDATE messages SET feedback = ? WHERE id = ?;`
	res, err := s.db.ExecContext(ctx, query, fb, messageID)
	if err != nil {
		return fmt.Errorf("error al actualizar calificación del mensaje: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return errors.New("mensaje no encontrado")
	}
	return nil
}
