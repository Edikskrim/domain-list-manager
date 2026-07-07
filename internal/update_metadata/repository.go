package update_metadata

import (
	"database/sql"
	"fmt"

	"domain-list-manager/internal/repository"
)

type UpdateMetadataRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *UpdateMetadataRepository {
	return &UpdateMetadataRepository{db: db}
}

func (r *UpdateMetadataRepository) Create(m *UpdateMetadata) error {
	lastFetched := m.LastFetched.Format("2006-01-02 15:04:05")
	_, err := r.db.Exec(
		"INSERT INTO update_metadata (source_id, etag, last_modified, last_fetched, content_hash) VALUES (?, ?, ?, ?, ?)",
		m.SourceID, m.ETag, m.LastModified, lastFetched, m.ContentHash,
	)
	if err != nil {
		return fmt.Errorf("create update metadata: %w", err)
	}
	return nil
}

func (r *UpdateMetadataRepository) Get(sourceID string) (*UpdateMetadata, error) {
	var m UpdateMetadata
	var lastFetched interface{}
	err := r.db.QueryRow(
		"SELECT etag, last_modified, last_fetched, content_hash FROM update_metadata WHERE source_id = ?",
		sourceID,
	).Scan(&m.ETag, &m.LastModified, &lastFetched, &m.ContentHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("update metadata not found for source: %s", sourceID)
		}
		return nil, fmt.Errorf("get update metadata: %w", err)
	}
	m.SourceID = sourceID
	m.LastFetched = repository.ParseTime(lastFetched)
	return &m, nil
}

func (r *UpdateMetadataRepository) Update(sourceID string, m *UpdateMetadata) error {
	lastFetched := m.LastFetched.Format("2006-01-02 15:04:05")
	res, err := r.db.Exec(
		"UPDATE update_metadata SET etag = ?, last_modified = ?, last_fetched = ?, content_hash = ? WHERE source_id = ?",
		m.ETag, m.LastModified, lastFetched, m.ContentHash, sourceID,
	)
	if err != nil {
		return fmt.Errorf("update update metadata: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("update metadata not found for source: %s", sourceID)
	}
	return nil
}

func (r *UpdateMetadataRepository) Delete(sourceID string) error {
	res, err := r.db.Exec("DELETE FROM update_metadata WHERE source_id = ?", sourceID)
	if err != nil {
		return fmt.Errorf("delete update metadata: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("update metadata not found for source: %s", sourceID)
	}
	return nil
}
