package domain

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// InMemoryRepository is a simple in-memory implementation of the Domain repository.
type InMemoryRepository struct {
	mu    sync.RWMutex
	domains map[string]*Domain
}

// NewInMemoryRepository creates and returns a new InMemoryRepository instance.
func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		domains: make(map[string]*Domain),
	}
}

// Create adds a new domain to the repository.
func (r *InMemoryRepository) Create(domain *Domain) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if domain.ID == "" {
		return ErrInvalidID
	}

	if _, exists := r.domains[domain.ID]; exists {
		return ErrDuplicateID
	}

	domain.CreatedAt = time.Now().UTC()
	domain.UpdatedAt = time.Now().UTC()
	r.domains[domain.ID] = domain

	return nil
}

// Get retrieves a domain by its ID.
func (r *InMemoryRepository) Get(id string) (*Domain, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	domain, exists := r.domains[id]
	if !exists {
		return nil, ErrNotFound
	}

	return domain, nil
}

// List returns all domains in the repository.
func (r *InMemoryRepository) List() ([]*Domain, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	domains := make([]*Domain, 0, len(r.domains))
	for _, domain := range r.domains {
		domains = append(domains, domain)
	}

	return domains, nil
}

// Update modifies an existing domain.
func (r *InMemoryRepository) Update(id string, domain *Domain) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.domains[id]; !exists {
		return ErrNotFound
	}

	domain.ID = id
	domain.UpdatedAt = time.Now().UTC()
	r.domains[id] = domain

	return nil
}

// Delete removes a domain from the repository.
func (r *InMemoryRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.domains[id]; !exists {
		return ErrNotFound
	}

	delete(r.domains, id)
	return nil
}

func (r *InMemoryRepository) FindByName(name string) ([]*Domain, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return nil, fmt.Errorf("domain name cannot be empty")
	}
	var domains []*Domain
	for _, d := range r.domains {
		if strings.ToLower(d.Name) == name {
			domains = append(domains, d)
		}
	}
	return domains, nil
}

func (r *InMemoryRepository) Upsert(domain *Domain) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if domain.ID == "" {
		return ErrInvalidID
	}

	now := time.Now().UTC()
	if _, exists := r.domains[domain.ID]; exists {
		domain.UpdatedAt = now
		r.domains[domain.ID] = domain
	} else {
		domain.CreatedAt = now
		domain.UpdatedAt = now
		r.domains[domain.ID] = domain
	}

	return nil
}

func (r *InMemoryRepository) DeleteByNameAndSource(name, sourceID string) error {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return fmt.Errorf("domain name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if sourceID != "" && sourceID != "none" {
		sourceID = strings.TrimSpace(sourceID)
		var found bool
		for id, d := range r.domains {
			if strings.ToLower(d.Name) == name {
				if d.SourceID != nil && *d.SourceID == sourceID {
					delete(r.domains, id)
					found = true
					break
				}
			}
		}
		if !found {
			return fmt.Errorf("domain not found")
		}
	} else {
		var ids []string
		for id, d := range r.domains {
			if strings.ToLower(d.Name) == name {
				ids = append(ids, id)
			}
		}
		for _, id := range ids {
			delete(r.domains, id)
		}
	}

	return nil
}

