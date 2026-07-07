package dashboard

import (
	"database/sql"
	"fmt"
	"time"
)

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

type DashboardData struct {
	TotalSources      int           `json:"total_sources"`
	EnabledSources    int           `json:"enabled_sources"`
	DisabledSources   int           `json:"disabled_sources"`
	TotalDomains      int           `json:"total_domains"`
	LastBuild         *BuildInfo    `json:"last_build"`
	BuildStats        BuildStats    `json:"build_stats"`
	RecentErrors      []string      `json:"recent_errors"`
	SourceStatus      []SourceStatus `json:"source_status"`
	StorageUsage      StorageUsage  `json:"storage_usage"`
	UpdatedAt         time.Time     `json:"updated_at"`
}

type BuildInfo struct {
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

type BuildStats struct {
	TotalBuilds      int       `json:"total_builds"`
	AvgDomains       float64   `json:"avg_domains"`
	AvgFetched       float64   `json:"avg_fetched"`
	AvgDuplicates    float64   `json:"avg_duplicates"`
	LastBuildDate    string    `json:"last_build_date"`
	FirstBuildDate   string    `json:"first_build_date"`
	AvgBuildTimeMs   float64   `json:"avg_build_time_ms"`
}

type SourceStatus struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Enabled     bool      `json:"enabled"`
	DomainCount int       `json:"domain_count"`
	LastUpdate  string    `json:"last_update"`
	LastError   string    `json:"last_error"`
	UpdatedAt   string    `json:"updated_at"`
}

type StorageUsage struct {
	DatabaseSizeBytes int64 `json:"database_size_bytes"`
	SnapshotCount     int   `json:"snapshot_count"`
	TotalDomainsCount int   `json:"total_domains_count"`
}

func (s *Service) GetDashboardData() (*DashboardData, error) {
	data := &DashboardData{
		UpdatedAt: time.Now().UTC(),
	}

	if err := s.fetchSourceStats(data); err != nil {
		return nil, fmt.Errorf("fetch source stats: %w", err)
	}

	if err := s.fetchDomainStats(data); err != nil {
		return nil, fmt.Errorf("fetch domain stats: %w", err)
	}

	if err := s.fetchLastBuild(data); err != nil {
		return nil, fmt.Errorf("fetch last build: %w", err)
	}

	if err := s.fetchBuildStats(data); err != nil {
		return nil, fmt.Errorf("fetch build stats: %w", err)
	}

	if err := s.fetchSourceStatus(data); err != nil {
		return nil, fmt.Errorf("fetch source status: %w", err)
	}

	if err := s.fetchStorageUsage(data); err != nil {
		return nil, fmt.Errorf("fetch storage usage: %w", err)
	}

	return data, nil
}

func (s *Service) fetchSourceStats(data *DashboardData) error {
	var totalSources, enabledSources int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM sources`).Scan(&totalSources)
	if err != nil {
		return err
	}

	err = s.db.QueryRow(`SELECT COUNT(*) FROM sources WHERE enabled = 1`).Scan(&enabledSources)
	if err != nil {
		return err
	}

	data.TotalSources = totalSources
	data.EnabledSources = enabledSources
	data.DisabledSources = totalSources - enabledSources

	return nil
}

func (s *Service) fetchDomainStats(data *DashboardData) error {
	var totalDomains int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM domains`).Scan(&totalDomains)
	if err != nil {
		return err
	}

	data.TotalDomains = totalDomains
	return nil
}

func (s *Service) fetchLastBuild(data *DashboardData) error {
	var buildInfo BuildInfo
	err := s.db.QueryRow(`
		SELECT id, build_time, total_domains, total_sources, total_fetched, total_parsed, duplicates, errors, build_time_ms, created_at
		FROM snapshots
		ORDER BY created_at DESC
		LIMIT 1
	`).Scan(
		&buildInfo.ID,
		&buildInfo.BuildTime,
		&buildInfo.TotalDomains,
		&buildInfo.TotalSources,
		&buildInfo.TotalFetched,
		&buildInfo.TotalParsed,
		&buildInfo.Duplicates,
		&buildInfo.Errors,
		&buildInfo.BuildTimeMs,
		&buildInfo.CreatedAt,
	)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			data.RecentErrors = []string{}
			return nil
		}
		return err
	}

	data.LastBuild = &buildInfo
	data.RecentErrors = parseErrorString(buildInfo.Errors)
	return nil
}

func (s *Service) fetchBuildStats(data *DashboardData) error {
	var stats BuildStats
	var totalBuilds int

	err := s.db.QueryRow(`SELECT COUNT(*) FROM snapshots`).Scan(&totalBuilds)
	if err != nil {
		return err
	}

	stats.TotalBuilds = totalBuilds

	if totalBuilds == 0 {
		return nil
	}

	err = s.db.QueryRow(`
		SELECT
			AVG(total_domains),
			AVG(total_fetched),
			AVG(duplicates),
			AVG(build_time_ms),
			MAX(created_at),
			MIN(created_at)
		FROM snapshots
	`).Scan(
		&stats.AvgDomains,
		&stats.AvgFetched,
		&stats.AvgDuplicates,
		&stats.AvgBuildTimeMs,
		&stats.LastBuildDate,
		&stats.FirstBuildDate,
	)
	if err != nil {
		return err
	}

	data.BuildStats = stats
	return nil
}

func (s *Service) fetchSourceStatus(data *DashboardData) error {
	rows, err := s.db.Query(`
		SELECT id, name, enabled, domain_count, last_update, last_error, updated_at
		FROM sources
		ORDER BY name ASC
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var sourceStatus []SourceStatus
	for rows.Next() {
		var status SourceStatus
		err := rows.Scan(
			&status.ID,
			&status.Name,
			&status.Enabled,
			&status.DomainCount,
			&status.LastUpdate,
			&status.LastError,
			&status.UpdatedAt,
		)
		if err != nil {
			return err
		}
		sourceStatus = append(sourceStatus, status)
	}

	data.SourceStatus = sourceStatus
	return nil
}

func (s *Service) fetchStorageUsage(data *DashboardData) error {
	var snapshotCount, totalDomainsCount int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM snapshots`).Scan(&snapshotCount)
	if err != nil {
		return err
	}

	err = s.db.QueryRow(`SELECT COUNT(*) FROM domains`).Scan(&totalDomainsCount)
	if err != nil {
		return err
	}

	data.StorageUsage = StorageUsage{
		SnapshotCount:     snapshotCount,
		TotalDomainsCount: totalDomainsCount,
	}

	return nil
}

func parseErrorString(errorsStr string) []string {
	if errorsStr == "" {
		return []string{}
	}
	return []string{errorsStr}
}
