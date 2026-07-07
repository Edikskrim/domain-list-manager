package dto

type DashboardResponse struct {
	TotalSources      int               `json:"total_sources"`
	EnabledSources    int               `json:"enabled_sources"`
	DisabledSources   int               `json:"disabled_sources"`
	TotalDomains      int               `json:"total_domains"`
	LastBuild         *LastBuildInfo    `json:"last_build"`
	BuildStats        BuildStatistics   `json:"build_stats"`
	RecentErrors      []string          `json:"recent_errors"`
	SourceStatus      []SourceStatusDTO `json:"source_status"`
	StorageUsage      StorageUsageDTO   `json:"storage_usage"`
	UpdatedAt         string            `json:"updated_at"`
}

type LastBuildInfo struct {
	ID           string `json:"id"`
	BuildTime    string `json:"build_time"`
	TotalDomains int    `json:"total_domains"`
	TotalSources int    `json:"total_sources"`
	TotalFetched int    `json:"total_fetched"`
	TotalParsed  int    `json:"total_parsed"`
	Duplicates   int    `json:"duplicates"`
	Errors       string `json:"errors"`
	BuildTimeMs  int64  `json:"build_time_ms"`
	CreatedAt    string `json:"created_at"`
}

type BuildStatistics struct {
	TotalBuilds    int     `json:"total_builds"`
	AvgDomains     float64 `json:"avg_domains"`
	AvgFetched     float64 `json:"avg_fetched"`
	AvgDuplicates  float64 `json:"avg_duplicates"`
	LastBuildDate  string  `json:"last_build_date"`
	FirstBuildDate string  `json:"first_build_date"`
	AvgBuildTimeMs float64 `json:"avg_build_time_ms"`
}

type SourceStatusDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Enabled     bool   `json:"enabled"`
	DomainCount int    `json:"domain_count"`
	LastUpdate  string `json:"last_update"`
	LastError   string `json:"last_error"`
	UpdatedAt   string `json:"updated_at"`
}

type StorageUsageDTO struct {
	DatabaseSizeBytes int64 `json:"database_size_bytes"`
	SnapshotCount     int   `json:"snapshot_count"`
	TotalDomainsCount int   `json:"total_domains_count"`
}
