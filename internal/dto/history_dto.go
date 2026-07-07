package dto

// SnapshotSummary represents a brief snapshot entry for list view.
type SnapshotSummary struct {
	ID           string `json:"id"`
	BuildTime    string `json:"build_time"`
	TotalDomains int    `json:"total_domains"`
	TotalSources int    `json:"total_sources"`
	TotalFetched int    `json:"total_fetched"`
	CreatedAt    string `json:"created_at"`
}

// SnapshotDetail represents a full snapshot entry with domains.
type SnapshotDetail struct {
	ID           string   `json:"id"`
	BuildTime    string   `json:"build_time"`
	TotalDomains int      `json:"total_domains"`
	TotalSources int      `json:"total_sources"`
	TotalFetched int      `json:"total_fetched"`
	TotalParsed  int      `json:"total_parsed"`
	Duplicates   int      `json:"duplicates"`
	Errors       []string `json:"errors,omitempty"`
	BuildTimeMs  int64    `json:"build_time_ms"`
	Domains      []string `json:"domains"`
	CreatedAt    string   `json:"created_at"`
}

// DiffResponse represents the diff between two snapshots.
type DiffResponse struct {
	Snapshot1     string   `json:"snapshot_1"`
	Snapshot2     string   `json:"snapshot_2"`
	BuildTime1    string   `json:"build_time_1"`
	BuildTime2    string   `json:"build_time_2"`
	TotalDomains1 int      `json:"total_domains_1"`
	TotalDomains2 int      `json:"total_domains_2"`
	Added         []string `json:"added"`
	Removed       []string `json:"removed"`
	AddedCount    int      `json:"added_count"`
	RemovedCount  int      `json:"removed_count"`
}
