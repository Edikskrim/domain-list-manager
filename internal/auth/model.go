package auth

import (
	"time"
)

// Session represents a user session.
type Session struct {
	ID       string
	Token    string
	Username string
	ExpiresAt time.Time
}
