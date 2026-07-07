package parser

import "errors"

// Parser error constants.
var (
	ErrUnknownFormat = errors.New("unknown parser format")
	ErrInvalidInput  = errors.New("invalid input")
	ErrEmptyContent  = errors.New("empty content")
)
