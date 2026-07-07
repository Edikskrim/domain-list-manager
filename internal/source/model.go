package source

import "time"

type Source struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	URL             string    `json:"url"`
	ParserType      string    `json:"parser_type"`
	Enabled         bool      `json:"enabled"`
	UpdateInterval  int       `json:"update_interval"`
	LastUpdate      time.Time `json:"last_update"`
	LastError       string    `json:"last_error"`
	DomainCount     int       `json:"domain_count"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Repository interface {
	Create(s *Source) error
	Get(id string) (*Source, error)
	List() ([]*Source, error)
	Update(id string, s *Source) error
	Delete(id string) error
	Enable(id string) error
	Disable(id string) error
}
