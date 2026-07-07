package dto

type DiagnosticsResponse struct {
	Intersections    IntersectionDiagnosticsDTO    `json:"intersections"`
	ParsingErrors    []ParsingErrorDTO               `json:"parsing_errors"`
	InvalidDomains   []InvalidDomainReportDTO        `json:"invalid_domains"`
	OverallSummary   OverallSummaryDTO               `json:"overall_summary"`
	AnalyzedAt       string                          `json:"analyzed_at"`
}

type IntersectionDiagnosticsDTO struct {
	TotalIntersections  int                              `json:"total_intersections"`
	IntersectingDomains []IntersectingDomainSummaryDTO    `json:"intersecting_domains"`
	Summary             DiagnosticsReportSummaryDTO       `json:"summary"`
	SourceDomains       []DiagnosticsSourceDomainInfoDTO  `json:"source_domains"`
}

type IntersectingDomainSummaryDTO struct {
	Domain      string   `json:"domain"`
	SourceCount int      `json:"source_count"`
	Sources     []string `json:"sources"`
}

type DiagnosticsReportSummaryDTO struct {
	TotalSources      int `json:"total_sources"`
	TotalDomains      int `json:"total_domains"`
	IntersectingCount int `json:"intersecting_count"`
	UniqueCount       int `json:"unique_count"`
}

type DiagnosticsSourceDomainInfoDTO struct {
	SourceID    string `json:"source_id"`
	SourceName  string `json:"source_name"`
	DomainCount int    `json:"domain_count"`
}

type ParsingErrorDTO struct {
	SourceID   string `json:"source_id"`
	SourceName string `json:"source_name"`
	Error      string `json:"error"`
	Timestamp  string `json:"timestamp"`
}

type InvalidDomainReportDTO struct {
	ID     string `json:"id"`
	Domain string `json:"domain"`
	Reason string `json:"reason"`
	Source string `json:"source"`
	Count  int    `json:"count"`
}

type OverallSummaryDTO struct {
	TotalSources       int `json:"total_sources"`
	EnabledSources     int `json:"enabled_sources"`
	TotalDomains       int `json:"total_domains"`
	IntersectingCount  int `json:"intersecting_count"`
	ParsingErrorCount  int `json:"parsing_error_count"`
	InvalidDomainCount int `json:"invalid_domain_count"`
}

type SourceDiagnosticsResponse struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	Description      string  `json:"description"`
	Enabled          bool    `json:"enabled"`
	LastUpdate       string  `json:"last_update"`
	LastError        string  `json:"last_error"`
	DomainCount      int     `json:"domain_count"`
	ParserType       string  `json:"parser_type"`
	UpdateInterval   int     `json:"update_interval"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
	TotalDomainsInDB int     `json:"total_domains_in_db"`
	LastFetchSuccess *bool   `json:"last_fetch_success"`
	LastFetchSize    int     `json:"last_fetch_size"`
	LastFetchError   string  `json:"last_fetch_error"`
	LastParseError   string  `json:"last_parse_error"`
	LastParseCount   int     `json:"last_parse_count"`
	ParsedSuccessfully *bool `json:"parsed_successfully"`
}
