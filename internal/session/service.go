package session

import (
	"context"
	"database/sql"
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

	CREATE INDEX IF NOT EXISTS idx_messages_session_id ON messages(session_id, created_at ASC);
	CREATE INDEX IF NOT EXISTS idx_sessions_updated_at ON sessions(updated_at DESC);
	`
	_, err := s.db.Exec(schema)
	return err
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
	SELECT id, session_id, role, content, thought_chain, provider_id, model_id,
	       tokens_prompt, tokens_completion, duration_seconds, feedback, status, created_at
	FROM messages
	WHERE session_id = ?
	ORDER BY created_at ASC;
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
		); err != nil {
			return nil, fmt.Errorf("error al leer mensaje de base de datos: %w", err)
		}

		m.Role = domain.ChatRole(roleStr)
		if fb.Valid {
			m.Feedback = &fb.String
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

	// Asegurar que la sesión exista
	now := time.Now().UTC()
	_, _ = s.db.ExecContext(ctx, `
		INSERT INTO sessions (id, title, created_at, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET updated_at = excluded.updated_at;
	`, msg.SessionID, "Conversación", now, now)

	var fb sql.NullString
	if msg.Feedback != nil && *msg.Feedback != "" {
		fb = sql.NullString{String: *msg.Feedback, Valid: true}
	}

	query := `
	INSERT INTO messages (
		id, session_id, role, content, thought_chain, provider_id, model_id,
		tokens_prompt, tokens_completion, duration_seconds, feedback, status, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		content = excluded.content,
		thought_chain = excluded.thought_chain,
		provider_id = excluded.provider_id,
		model_id = excluded.model_id,
		tokens_prompt = excluded.tokens_prompt,
		tokens_completion = excluded.tokens_completion,
		duration_seconds = excluded.duration_seconds,
		feedback = COALESCE(excluded.feedback, messages.feedback),
		status = excluded.status;
	`
	_, err := s.db.ExecContext(ctx, query,
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
	)
	if err != nil {
		return fmt.Errorf("error al guardar mensaje en base de datos: %w", err)
	}

	// Actualizar updated_at en la sesión
	_, _ = s.db.ExecContext(ctx, `UPDATE sessions SET updated_at = ? WHERE id = ?;`, now, msg.SessionID)

	return nil
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
