package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"domain-list-manager/internal/domain"
)

type DomainRepository struct {
	db *sql.DB
}

func NewDomainRepository(db *sql.DB) *DomainRepository {
	return &DomainRepository{db: db}
}

func (r *DomainRepository) Create(d *domain.Domain) error {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	var sourceID sql.NullString
	if d.SourceID != nil {
		sourceID = sql.NullString{String: *d.SourceID, Valid: true}
	}
	_, err := r.db.Exec(
		"INSERT INTO domains (id, name, source_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		d.ID, d.Name, sourceID, now, now,
	)
	if err != nil {
		return fmt.Errorf("create domain: %w", err)
	}
	return nil
}

func (r *DomainRepository) Get(id string) (*domain.Domain, error) {
	var name string
	var sourceID, createdAt, updatedAt interface{}
	err := r.db.QueryRow(
		"SELECT name, source_id, created_at, updated_at FROM domains WHERE id = ?", id,
	).Scan(&name, &sourceID, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get domain: %w", err)
	}
	d := &domain.Domain{
		ID: id, Name: name,
		CreatedAt: ParseTime(createdAt), UpdatedAt: ParseTime(updatedAt),
	}
	if sourceID != nil {
		if s, ok := sourceID.(string); ok && s != "" {
			d.SourceID = &s
		}
	}
	return d, nil
}

func (r *DomainRepository) List() ([]*domain.Domain, error) {
	rows, err := r.db.Query("SELECT id, name, source_id, created_at, updated_at FROM domains")
	if err != nil {
		return nil, fmt.Errorf("list domains: %w", err)
	}
	defer rows.Close()

	var domains []*domain.Domain
	for rows.Next() {
		var d domain.Domain
		var sourceID, createdAt, updatedAt interface{}
		if err := rows.Scan(&d.ID, &d.Name, &sourceID, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan domain: %w", err)
		}
		d.CreatedAt = ParseTime(createdAt)
		d.UpdatedAt = ParseTime(updatedAt)
		if sourceID != nil {
			if s, ok := sourceID.(string); ok && s != "" {
				d.SourceID = &s
			}
		}
		domains = append(domains, &d)
	}
	return domains, nil
}

func (r *DomainRepository) Update(id string, d *domain.Domain) error {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	var sourceID sql.NullString
	if d.SourceID != nil {
		sourceID = sql.NullString{String: *d.SourceID, Valid: true}
	}
	res, err := r.db.Exec(
		"UPDATE domains SET name = ?, source_id = ?, updated_at = ? WHERE id = ?",
		d.Name, sourceID, now, id,
	)
	if err != nil {
		return fmt.Errorf("update domain: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *DomainRepository) Delete(id string) error {
	res, err := r.db.Exec("DELETE FROM domains WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete domain: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *DomainRepository) ListBySource(sourceID string) ([]*domain.Domain, error) {
	rows, err := r.db.Query("SELECT id, name, source_id, created_at, updated_at FROM domains WHERE source_id = ?", sourceID)
	if err != nil {
		return nil, fmt.Errorf("list domains by source: %w", err)
	}
	defer rows.Close()

	var domains []*domain.Domain
	for rows.Next() {
		var d domain.Domain
		var sourceID, createdAt, updatedAt interface{}
		if err := rows.Scan(&d.ID, &d.Name, &sourceID, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan domain: %w", err)
		}
		d.CreatedAt = ParseTime(createdAt)
		d.UpdatedAt = ParseTime(updatedAt)
		if sourceID != nil {
			if s, ok := sourceID.(string); ok && s != "" {
				d.SourceID = &s
			}
		}
		domains = append(domains, &d)
	}
	return domains, nil
}

func (r *DomainRepository) Search(query string) ([]*domain.Domain, error) {
	searchPattern := "%" + query + "%"
	rows, err := r.db.Query("SELECT id, name, source_id, created_at, updated_at FROM domains WHERE name LIKE ?", searchPattern)
	if err != nil {
		return nil, fmt.Errorf("search domains: %w", err)
	}
	defer rows.Close()

	var domains []*domain.Domain
	for rows.Next() {
		var d domain.Domain
		var sourceID, createdAt, updatedAt interface{}
		if err := rows.Scan(&d.ID, &d.Name, &sourceID, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan domain: %w", err)
		}
		d.CreatedAt = ParseTime(createdAt)
		d.UpdatedAt = ParseTime(updatedAt)
		if sourceID != nil {
			if s, ok := sourceID.(string); ok && s != "" {
				d.SourceID = &s
			}
		}
		domains = append(domains, &d)
	}
	return domains, nil
}

func (r *DomainRepository) FindByName(name string) ([]*domain.Domain, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return nil, fmt.Errorf("domain name cannot be empty")
	}
	rows, err := r.db.Query("SELECT id, name, source_id, created_at, updated_at FROM domains WHERE name = ?", name)
	if err != nil {
		return nil, fmt.Errorf("find by name: %w", err)
	}
	defer rows.Close()

	var domains []*domain.Domain
	for rows.Next() {
		var d domain.Domain
		var sourceID, createdAt, updatedAt interface{}
		if err := rows.Scan(&d.ID, &d.Name, &sourceID, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan domain: %w", err)
		}
		d.CreatedAt = ParseTime(createdAt)
		d.UpdatedAt = ParseTime(updatedAt)
		if sourceID != nil {
			if s, ok := sourceID.(string); ok && s != "" {
				d.SourceID = &s
			}
		}
		domains = append(domains, &d)
	}
	return domains, nil
}

func (r *DomainRepository) Upsert(d *domain.Domain) error {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	var sourceID sql.NullString
	if d.SourceID != nil {
		sourceID = sql.NullString{String: *d.SourceID, Valid: true}
	}
	_, err := r.db.Exec(
		"INSERT OR REPLACE INTO domains (id, name, source_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		d.ID, d.Name, sourceID, now, now,
	)
	if err != nil {
		return fmt.Errorf("upsert domain: %w", err)
	}
	return nil
}

func (r *DomainRepository) DeleteByNameAndSource(name, sourceID string) error {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return fmt.Errorf("domain name cannot be empty")
	}

	if sourceID != "" && sourceID != "none" {
		sourceID = strings.TrimSpace(sourceID)
		_, err := r.db.Exec("DELETE FROM domains WHERE name = ? AND source_id = ?", name, sourceID)
		if err != nil {
			return fmt.Errorf("delete domain by source: %w", err)
		}
	} else {
		_, err := r.db.Exec("DELETE FROM domains WHERE name = ?", name)
		if err != nil {
			return fmt.Errorf("delete domain: %w", err)
		}
	}

	return nil
}

func ParseTime(v interface{}) time.Time {
	if v == nil {
		return time.Time{}
	}
	switch val := v.(type) {
	case int64:
		return time.Unix(val, 0).UTC()
	case []byte:
		s := string(val)
		for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02T15:04:05Z", "2006-01-02T15:04:05"} {
			if t, err := time.Parse(layout, s); err == nil {
				return t
			}
		}
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return t
		}
	case string:
		for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02T15:04:05Z", "2006-01-02T15:04:05"} {
			if t, err := time.Parse(layout, val); err == nil {
				return t
			}
		}
		if t, err := time.Parse(time.RFC3339, val); err == nil {
			return t
		}
	case time.Time:
		return val
	}
	return time.Time{}
}
