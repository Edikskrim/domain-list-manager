package domain

import (
	"time"
)

// Domain represents a single domain entry in our system.
type Domain struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	SourceID  *string   `json:"source_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Repository defines the interface for domain storage operations.
type Repository interface {
	Create(domain *Domain) error
	Get(id string) (*Domain, error)
	List() ([]*Domain, error)
	Update(id string, domain *Domain) error
	Delete(id string) error
	ListBySource(sourceID string) ([]*Domain, error)
	Search(query string) ([]*Domain, error)
	Upsert(domain *Domain) error
	FindByName(name string) ([]*Domain, error)
	DeleteByNameAndSource(name, sourceID string) error
}