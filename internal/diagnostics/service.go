package diagnostics

import (
	"database/sql"
	"fmt"
	"net/mail"
	"regexp"
	"sort"
	"strings"
	"time"

	"domain-list-manager/internal/fetcher"
	"domain-list-manager/internal/intersection"
	"domain-list-manager/internal/parser"
	"domain-list-manager/internal/source"
)

type Service struct {
	sourceRepo source.Repository
	db         *sql.DB
	fetcher    *fetcher.Fetcher
}

func NewService(sourceRepo source.Repository, db *sql.DB, fetcher *fetcher.Fetcher) *Service {
	return &Service{
		sourceRepo: sourceRepo,
		db:         db,
		fetcher:    fetcher,
	}
}

type DiagnosticsResult struct {
	Intersections    IntersectionDiagnostics   `json:"intersections"`
	ParsingErrors    []ParsingError              `json:"parsing_errors"`
	InvalidDomains   []InvalidDomainReport       `json:"invalid_domains"`
	OverallSummary   OverallSummary              `json:"overall_summary"`
	AnalyzedAt       string                      `json:"analyzed_at"`
}

type IntersectionDiagnostics struct {
	TotalIntersections int                           `json:"total_intersections"`
	IntersectingDomains []IntersectingDomainSummary    `json:"intersecting_domains"`
	Summary             intersection.ReportSummary   `json:"summary"`
	SourceDomains       []intersection.SourceDomainInfo `json:"source_domains"`
}

type IntersectingDomainSummary struct {
	Domain      string   `json:"domain"`
	SourceCount int      `json:"source_count"`
	Sources     []string `json:"sources"`
}

type ParsingError struct {
	SourceID   string `json:"source_id"`
	SourceName string `json:"source_name"`
	Error      string `json:"error"`
	Timestamp  string `json:"timestamp"`
}

type InvalidDomainReport struct {
	ID     string `json:"id"`
	Domain string `json:"domain"`
	Reason string `json:"reason"`
	Source string `json:"source"`
	Count  int    `json:"count"`
}

type OverallSummary struct {
	TotalSources      int `json:"total_sources"`
	EnabledSources    int `json:"enabled_sources"`
	TotalDomains      int `json:"total_domains"`
	IntersectingCount int `json:"intersecting_count"`
	ParsingErrorCount int `json:"parsing_error_count"`
	InvalidDomainCount int `json:"invalid_domain_count"`
}

func (s *Service) RunDiagnostics() (*DiagnosticsResult, error) {
	result := &DiagnosticsResult{
		AnalyzedAt: time.Now().UTC().Format(time.RFC3339),
	}

	if err := s.analyzeIntersections(result); err != nil {
		return nil, fmt.Errorf("analyze intersections: %w", err)
	}

	s.analyzeParsingErrors(result)
	s.analyzeInvalidDomains(result)
	s.computeOverallSummary(result)

	return result, nil
}

func (s *Service) analyzeIntersections(result *DiagnosticsResult) error {
	sources, err := s.sourceRepo.List()
	if err != nil {
		return err
	}

	intersectingDomains := s.getIntersectingDomains()
	result.Intersections.IntersectingDomains = intersectingDomains
	result.Intersections.TotalIntersections = len(intersectingDomains)

	enabledCount := 0
	for _, src := range sources {
		if src.Enabled {
			enabledCount++
		}
	}

	result.Intersections.Summary = intersection.ReportSummary{
		TotalSources:      enabledCount,
		TotalDomains:      s.getTotalDomains(),
		IntersectingCount: len(intersectingDomains),
	}

	return nil
}

func (s *Service) getIntersectingDomains() []IntersectingDomainSummary {
	var intersectingDomains []IntersectingDomainSummary

	rows, err := s.db.Query(`
		SELECT d1.name, COUNT(*) as source_count
		FROM domains d1
		WHERE d1.name IN (
			SELECT d2.name FROM domains d2
			INNER JOIN (
				SELECT name FROM domains GROUP BY name HAVING COUNT(*) > 1
			) d3 ON d2.name = d3.name
		)
		GROUP BY d1.name
		ORDER BY source_count DESC
		LIMIT 100
	`)
	if err != nil {
		return intersectingDomains
	}
	defer rows.Close()

	for rows.Next() {
		var domain string
		var count int
		if err := rows.Scan(&domain, &count); err != nil {
			continue
		}
		intersectingDomains = append(intersectingDomains, IntersectingDomainSummary{
			Domain:      domain,
			SourceCount: count,
		})
	}

	return intersectingDomains
}

func (s *Service) analyzeParsingErrors(result *DiagnosticsResult) {
	rows, err := s.db.Query(`SELECT id, name, last_error FROM sources WHERE last_error != '' ORDER BY updated_at DESC`)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var srcID, srcName, lastError string
		if err := rows.Scan(&srcID, &srcName, &lastError); err != nil {
			continue
		}

		result.ParsingErrors = append(result.ParsingErrors, ParsingError{
			SourceID:   srcID,
			SourceName: srcName,
			Error:      lastError,
		})
	}
}

func (s *Service) analyzeInvalidDomains(result *DiagnosticsResult) {
	rows, err := s.db.Query(`SELECT id, name, COALESCE(source_id, 'manual') as source_id FROM domains`)
	if err != nil {
		return
	}
	defer rows.Close()

	var invalidDomains []InvalidDomainReport

	for rows.Next() {
		var id, domain, sourceID string
		if err := rows.Scan(&id, &domain, &sourceID); err != nil {
			continue
		}

		domain = strings.ToLower(strings.TrimSpace(domain))
		if domain == "" {
			invalidDomains = append(invalidDomains, InvalidDomainReport{
				ID:     id,
				Domain: domain,
				Reason: "empty domain",
				Source: sourceID,
				Count:  1,
			})
			continue
		}

		sourceID = strings.TrimSpace(sourceID)
		if sourceID == "" {
			sourceID = "manual"
		}

		reasons := validateDomain(domain)
		if len(reasons) > 0 {
			invalidDomains = append(invalidDomains, InvalidDomainReport{
				ID:     id,
				Domain: domain,
				Reason: strings.Join(reasons, ", "),
				Source: sourceID,
				Count:  1,
			})
		}
	}

	sort.Slice(invalidDomains, func(i, j int) bool {
		return invalidDomains[i].Domain < invalidDomains[j].Domain
	})

	result.InvalidDomains = invalidDomains
}

var domainRegex = regexp.MustCompile(`^([a-z0-9]([a-z0-9-]*[a-z0-9])?\.)*[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

func validateDomain(domain string) []string {
	var reasons []string

	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "" {
		return []string{"empty domain"}
	}

	if len(domain) > 253 {
		reasons = append(reasons, "domain too long")
	}

	if strings.HasPrefix(domain, "-") || strings.HasSuffix(domain, "-") {
		reasons = append(reasons, "starts or ends with hyphen")
	}

	if strings.Contains(domain, "--") {
		reasons = append(reasons, "consecutive hyphens")
	}

	if strings.Contains(domain, "..") {
		reasons = append(reasons, "consecutive dots")
	}

	_, err := mail.ParseAddress(domain + " <" + domain + ">")
	if err != nil && !domainRegex.MatchString(domain) {
		reasons = append(reasons, "invalid domain format")
	}

	if len(reasons) == 0 {
		parts := strings.Split(domain, ".")
		for _, part := range parts {
			if len(part) > 63 {
				reasons = append(reasons, fmt.Sprintf("label '%s' too long", part))
			}
			if part == "" {
				reasons = append(reasons, "empty label")
			}
		}
	}

	return reasons
}

func (s *Service) computeOverallSummary(result *DiagnosticsResult) {
	result.OverallSummary.IntersectingCount = result.Intersections.TotalIntersections
	result.OverallSummary.ParsingErrorCount = len(result.ParsingErrors)
	result.OverallSummary.InvalidDomainCount = len(result.InvalidDomains)
}

func (s *Service) getTotalDomains() int {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM domains`).Scan(&count)
	if err != nil {
		return 0
	}
	return count
}

func (s *Service) GetSourceDiagnostics(sourceID string) (*SourceDiagnostics, error) {
	src, err := s.sourceRepo.Get(sourceID)
	if err != nil {
		return nil, fmt.Errorf("get source: %w", err)
	}

	diagnostics := &SourceDiagnostics{
		ID:             src.ID,
		Name:           src.Name,
		Description:    src.Description,
		Enabled:        src.Enabled,
		LastUpdate:     src.LastUpdate.Format(time.RFC3339),
		LastError:      src.LastError,
		DomainCount:    src.DomainCount,
		ParserType:     src.ParserType,
		UpdateInterval: src.UpdateInterval,
		CreatedAt:      src.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      src.UpdatedAt.Format(time.RFC3339),
	}

	rows, err := s.db.Query(`SELECT DISTINCT name FROM domains`)
	if err != nil {
		return diagnostics, nil
	}
	defer rows.Close()

	domainCount := 0
	for rows.Next() {
		domainCount++
	}
	diagnostics.TotalDomainsInDB = domainCount

	content, err := s.fetcher.Fetch(src.URL, "", "")
	if err != nil {
		diagnostics.LastFetchError = err.Error()
		f := false
		diagnostics.LastFetchSuccess = &f
		return diagnostics, nil
	}

	if content != nil && len(content.Body) > 0 {
		diagnostics.LastFetchSize = len(content.Body)
		t := true
		diagnostics.LastFetchSuccess = &t

		ap := parser.NewAutoParser()
		entries, parseErr := ap.Parse(content.Body)
		if parseErr != nil {
			diagnostics.LastParseError = parseErr.Error()
		} else {
			diagnostics.LastParseCount = len(entries)
			ps := true
			diagnostics.ParsedSuccessfully = &ps
		}
	}

	return diagnostics, nil
}

type SourceDiagnostics struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	Enabled          bool      `json:"enabled"`
	LastUpdate       string    `json:"last_update"`
	LastError        string    `json:"last_error"`
	DomainCount      int       `json:"domain_count"`
	ParserType       string    `json:"parser_type"`
	UpdateInterval   int       `json:"update_interval"`
	CreatedAt        string    `json:"created_at"`
	UpdatedAt        string    `json:"updated_at"`
	TotalDomainsInDB int       `json:"total_domains_in_db"`
	LastFetchSuccess *bool     `json:"last_fetch_success"`
	LastFetchSize    int       `json:"last_fetch_size"`
	LastFetchError   string    `json:"last_fetch_error"`
	LastParseError   string    `json:"last_parse_error"`
	LastParseCount   int       `json:"last_parse_count"`
	ParsedSuccessfully *bool   `json:"parsed_successfully"`
}
