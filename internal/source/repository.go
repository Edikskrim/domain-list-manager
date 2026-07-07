package source

import (
	"database/sql"
	"fmt"
	"time"

	"domain-list-manager/internal/repository"
)

type SourceRepository struct {
	db *sql.DB
}

func NewSourceRepository(db *sql.DB) *SourceRepository {
	return &SourceRepository{db: db}
}

func (r *SourceRepository) Create(s *Source) error {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	_, err := r.db.Exec(
		"INSERT INTO sources (id, name, description, url, parser_type, enabled, update_interval, last_update, last_error, domain_count, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		s.ID, s.Name, s.Description, s.URL, s.ParserType, s.Enabled, s.UpdateInterval, interface{}(s.LastUpdate.Format("2006-01-02 15:04:05")), s.LastError, s.DomainCount, now, now,
	)
	if err != nil {
		return fmt.Errorf("create source: %w", err)
	}
	return nil
}

func (r *SourceRepository) Get(id string) (*Source, error) {
	var s Source
	var lastUpdate interface{}
	err := r.db.QueryRow(
		"SELECT name, description, url, parser_type, enabled, update_interval, last_update, last_error, domain_count, created_at, updated_at FROM sources WHERE id = ?", id,
	).Scan(&s.Name, &s.Description, &s.URL, &s.ParserType, &s.Enabled, &s.UpdateInterval, &lastUpdate, &s.LastError, &s.DomainCount, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("source not found: %s", id)
		}
		return nil, fmt.Errorf("get source: %w", err)
	}
	s.ID = id
	s.LastUpdate = repository.ParseTime(lastUpdate)
	return &s, nil
}

func (r *SourceRepository) List() ([]*Source, error) {
	rows, err := r.db.Query("SELECT id, name, description, url, parser_type, enabled, update_interval, last_update, last_error, domain_count, created_at, updated_at FROM sources ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("list sources: %w", err)
	}
	defer rows.Close()

	var sources []*Source
	for rows.Next() {
		var s Source
		var lastUpdate interface{}
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.URL, &s.ParserType, &s.Enabled, &s.UpdateInterval, &lastUpdate, &s.LastError, &s.DomainCount, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan source: %w", err)
		}
		s.LastUpdate = repository.ParseTime(lastUpdate)
		sources = append(sources, &s)
	}
	return sources, nil
}

func (r *SourceRepository) Update(id string, s *Source) error {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	res, err := r.db.Exec(
		"UPDATE sources SET name = ?, description = ?, url = ?, parser_type = ?, enabled = ?, update_interval = ?, last_update = ?, last_error = ?, domain_count = ?, updated_at = ? WHERE id = ?",
		s.Name, s.Description, s.URL, s.ParserType, s.Enabled, s.UpdateInterval, interface{}(s.LastUpdate.Format("2006-01-02 15:04:05")), s.LastError, s.DomainCount, now, id,
	)
	if err != nil {
		return fmt.Errorf("update source: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("source not found: %s", id)
	}
	return nil
}

func (r *SourceRepository) Delete(id string) error {
	res, err := r.db.Exec("DELETE FROM sources WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete source: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("source not found: %s", id)
	}
	return nil
}

func (r *SourceRepository) Enable(id string) error {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	res, err := r.db.Exec("UPDATE sources SET enabled = 1, updated_at = ? WHERE id = ?", now, id)
	if err != nil {
		return fmt.Errorf("enable source: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("source not found: %s", id)
	}
	return nil
}

func (r *SourceRepository) Disable(id string) error {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	res, err := r.db.Exec("UPDATE sources SET enabled = 0, updated_at = ? WHERE id = ?", now, id)
	if err != nil {
		return fmt.Errorf("disable source: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("source not found: %s", id)
	}
	return nil
}
