package auth

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SessionRepository handles session persistence.
type SessionRepository struct {
	db *sql.DB
}

// NewSessionRepository creates a new SessionRepository.
func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// Create stores a new session in the database.
func (r *SessionRepository) Create(session *Session) error {
	query := `INSERT INTO sessions (id, token, username, expires_at) VALUES (?, ?, ?, ?)`
	_, err := r.db.Exec(query, session.ID, session.Token, session.Username, session.ExpiresAt)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

// GetByToken retrieves a session by its token.
func (r *SessionRepository) GetByToken(token string) (*Session, error) {
	var session Session
	query := `SELECT id, token, username, expires_at FROM sessions WHERE token = ? AND expires_at > datetime('now')`
	err := r.db.QueryRow(query, token).Scan(&session.ID, &session.Token, &session.Username, &session.ExpiresAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found or expired")
		}
		return nil, fmt.Errorf("get session: %w", err)
	}
	return &session, nil
}

// DeleteByToken removes a session by its token.
func (r *SessionRepository) DeleteByToken(token string) error {
	query := `DELETE FROM sessions WHERE token = ?`
	_, err := r.db.Exec(query, token)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// CleanupExpired removes all expired sessions.
func (r *SessionRepository) CleanupExpired() error {
	query := `DELETE FROM sessions WHERE expires_at <= datetime('now')`
	_, err := r.db.Exec(query)
	if err != nil {
		return fmt.Errorf("cleanup expired sessions: %w", err)
	}
	return nil
}

// NewSession creates a new session with a generated token.
func NewSession(username string, duration time.Duration) *Session {
	now := time.Now()
	return &Session{
		ID:       uuid.New().String(),
		Token:    uuid.New().String(),
		Username: username,
		ExpiresAt: now.Add(duration),
	}
}
