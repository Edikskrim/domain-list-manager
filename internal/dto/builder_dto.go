package dto

// BuildResponse represents the API response for a build operation.
type BuildResponse struct {
	Success      bool     `json:"success"`
	Domains      []string `json:"domains,omitempty"`
	TotalDomains int      `json:"total_domains"`
	TotalSources int      `json:"total_sources"`
	TotalFetched int      `json:"total_fetched"`
	TotalParsed  int      `json:"total_parsed"`
	Duplicates   int      `json:"duplicates_removed"`
	Errors       []string `json:"errors,omitempty"`
	BuildTimeMs  int64    `json:"build_time_ms"`
}

// BuildStatusResponse represents the status of the last build.
type BuildStatusResponse struct {
	LastBuildTime string `json:"last_build_time"`
	TotalDomains  int    `json:"total_domains"`
	TotalSources  int    `json:"total_sources"`
	TotalFetched  int    `json:"total_fetched"`
	BuildTimeMs   int64  `json:"build_time_ms"`
	HasOutput     bool   `json:"has_output"`
	OutputPath    string `json:"output_path,omitempty"`
}
