package update_metadata

import "time"

type UpdateMetadata struct {
	SourceID     string    `json:"source_id"`
	ETag         string    `json:"etag"`
	LastModified string    `json:"last_modified"`
	LastFetched  time.Time `json:"last_fetched"`
	ContentHash  string    `json:"content_hash"`
}

type Repository interface {
	Create(m *UpdateMetadata) error
	Get(sourceID string) (*UpdateMetadata, error)
	Update(sourceID string, m *UpdateMetadata) error
	Delete(sourceID string) error
}
