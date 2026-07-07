package setting

import (
	"database/sql"
	"fmt"
	"time"

	"domain-list-manager/internal/repository"
)

type SettingRepository struct {
	db *sql.DB
}

func NewSettingRepository(db *sql.DB) *SettingRepository {
	return &SettingRepository{db: db}
}

func (r *SettingRepository) Create(s *Setting) error {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	_, err := r.db.Exec(
		"INSERT INTO settings (key, value, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		s.Key, s.Value, s.Description, now, now,
	)
	if err != nil {
		return fmt.Errorf("create setting: %w", err)
	}
	return nil
}

func (r *SettingRepository) Get(key string) (*Setting, error) {
	var s Setting
	var createdAt, updatedAt interface{}
	err := r.db.QueryRow(
		"SELECT value, description, created_at, updated_at FROM settings WHERE key = ?", key,
	).Scan(&s.Value, &s.Description, &createdAt, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("setting not found: %s", key)
		}
		return nil, fmt.Errorf("get setting: %w", err)
	}
	s.Key = key
	s.CreatedAt = repository.ParseTime(createdAt)
	s.UpdatedAt = repository.ParseTime(updatedAt)
	return &s, nil
}

func (r *SettingRepository) List() ([]*Setting, error) {
	rows, err := r.db.Query("SELECT key, value, description, created_at, updated_at FROM settings ORDER BY key")
	if err != nil {
		return nil, fmt.Errorf("list settings: %w", err)
	}
	defer rows.Close()

	var settings []*Setting
	for rows.Next() {
		var s Setting
		var createdAt, updatedAt interface{}
		if err := rows.Scan(&s.Key, &s.Value, &s.Description, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan setting: %w", err)
		}
		s.CreatedAt = repository.ParseTime(createdAt)
		s.UpdatedAt = repository.ParseTime(updatedAt)
		settings = append(settings, &s)
	}
	return settings, nil
}

func (r *SettingRepository) Update(key string, s *Setting) error {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	res, err := r.db.Exec(
		"UPDATE settings SET value = ?, description = ?, updated_at = ? WHERE key = ?",
		s.Value, s.Description, now, key,
	)
	if err != nil {
		return fmt.Errorf("update setting: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("setting not found: %s", key)
	}
	return nil
}

func (r *SettingRepository) Delete(key string) error {
	res, err := r.db.Exec("DELETE FROM settings WHERE key = ?", key)
	if err != nil {
		return fmt.Errorf("delete setting: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("setting not found: %s", key)
	}
	return nil
}
