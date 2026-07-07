package setting

import (
	"fmt"
	"strconv"

	"domain-list-manager/internal/config"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) EnsureDefaults() error {
	defaults := []Setting{
		{Key: "fetcher.timeout", Value: "30", Description: "Timeout for fetching sources in seconds"},
		{Key: "fetcher.max_retries", Value: "3", Description: "Maximum number of retries for fetching"},
		{Key: "fetcher.max_body_size", Value: "52428800", Description: "Maximum response body size in bytes"},
		{Key: "fetcher.max_redirects", Value: "10", Description: "Maximum number of HTTP redirects"},
		{Key: "http_client.user_agent", Value: "DomainListManager/1.0", Description: "User-Agent for HTTP requests"},
		{Key: "builder.snapshot_count", Value: "10", Description: "Number of build snapshots to keep"},
		{Key: "builder.output_path", Value: "output/domains.lst", Description: "Path to publish domains.lst"},
	}

	for _, d := range defaults {
		if _, err := s.repo.Get(d.Key); err != nil {
			if err.Error() == fmt.Sprintf("setting not found: %s", d.Key) {
				if err := s.repo.Create(&d); err != nil {
					return fmt.Errorf("ensure default %s: %w", d.Key, err)
				}
			}
		}
	}
	return nil
}

func (s *Service) ApplyToConfig(cfg *config.Config) error {
	settings, err := s.repo.List()
	if err != nil {
		return fmt.Errorf("apply settings to config: %w", err)
	}

	m := make(map[string]string)
	for _, st := range settings {
		m[st.Key] = st.Value
	}

	if v, ok := m["fetcher.timeout"]; ok {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.Fetcher.Timeout = n
		}
	}

	if v, ok := m["fetcher.max_retries"]; ok {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			cfg.Fetcher.MaxRetries = n
		}
	}

	if v, ok := m["fetcher.max_body_size"]; ok {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			cfg.Fetcher.MaxBodySize = n
		}
	}

	if v, ok := m["fetcher.max_redirects"]; ok {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.Fetcher.MaxRedirects = n
		}
	}

	if v, ok := m["http_client.user_agent"]; ok && v != "" {
		cfg.HTTPClient.UserAgent = v
	}

	snapshotCount := m["builder.snapshot_count"]
	outputPath := m["builder.output_path"]
	if n, err := strconv.Atoi(snapshotCount); err == nil && n > 0 {
		cfg.Builder.SnapshotCount = n
	}
	if outputPath != "" {
		cfg.Builder.OutputPath = outputPath
	}

	return nil
}

func (s *Service) GetMap() (map[string]string, error) {
	settings, err := s.repo.List()
	if err != nil {
		return nil, fmt.Errorf("get settings map: %w", err)
	}

	m := make(map[string]string)
	for _, st := range settings {
		m[st.Key] = st.Value
	}
	return m, nil
}
