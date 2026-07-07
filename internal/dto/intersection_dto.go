package dto

type IntersectionReportDTO struct {
	IntersectingDomains []IntersectingDomainDTO `json:"intersecting_domains"`
	UniqueDomains       []UniqueDomainDTO       `json:"unique_domains"`
	Summary             ReportSummaryDTO        `json:"summary"`
	SourceDomains       []SourceDomainInfoDTO   `json:"source_domains"`
	AnalyzedAt          string                  `json:"analyzed_at"`
}

type IntersectingDomainDTO struct {
	Domain      string   `json:"domain"`
	SourceCount int      `json:"source_count"`
	Sources     []string `json:"sources"`
}

type UniqueDomainDTO struct {
	Domain string `json:"domain"`
	Source string `json:"source"`
}

type ReportSummaryDTO struct {
	TotalSources      int `json:"total_sources"`
	TotalDomains      int `json:"total_domains"`
	IntersectingCount int `json:"intersecting_count"`
	UniqueCount       int `json:"unique_count"`
}

type SourceDomainInfoDTO struct {
	SourceID    string `json:"source_id"`
	SourceName  string `json:"source_name"`
	DomainCount int    `json:"domain_count"`
}
