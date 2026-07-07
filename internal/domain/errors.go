package domain

import "errors"

// Error constants for domain operations
var (
	ErrNotFound     = errors.New("domain not found")
	ErrInvalidID    = errors.New("invalid domain ID")
	ErrDuplicateID  = errors.New("duplicate domain ID")
)