package auth

import (
	"fmt"
	"time"
)

// Service handles authentication logic.
type Service struct {
	sessionRepo *SessionRepository
	username    string
	password    string
	sessionDuration time.Duration
}

// NewService creates a new auth service.
func NewService(sessionRepo *SessionRepository, username, password string) *Service {
	return &Service{
		sessionRepo: sessionRepo,
		username:    username,
		password:    password,
		sessionDuration: 24 * time.Hour,
	}
}

// Login validates credentials and creates a session.
func (s *Service) Login(username, password string) (*Session, error) {
	if username != s.username || password != s.password {
		return nil, fmt.Errorf("invalid credentials")
	}

	session := NewSession(username, s.sessionDuration)
	if err := s.sessionRepo.Create(session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	return session, nil
}

// Logout removes a session by token.
func (s *Service) Logout(token string) error {
	return s.sessionRepo.DeleteByToken(token)
}

// GetSession retrieves a session by token.
func (s *Service) GetSession(token string) (*Session, error) {
	return s.sessionRepo.GetByToken(token)
}

// CleanupExpired removes all expired sessions.
func (s *Service) CleanupExpired() error {
	return s.sessionRepo.CleanupExpired()
}

// ValidateSession checks if a session is valid (exists and not expired).
func (s *Service) ValidateSession(token string) bool {
	session, err := s.GetSession(token)
	if err != nil {
		return false
	}
	return !session.ExpiresAt.Before(time.Now())
}
