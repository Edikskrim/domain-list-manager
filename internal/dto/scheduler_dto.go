package dto

import "time"

type SchedulerStatusResponse struct {
	Running     bool      `json:"running"`
	LastUpdate  *time.Time `json:"last_update"`
	NextUpdate  *time.Time `json:"next_update"`
	UpdateTime  int64     `json:"update_interval_ms"`
	UpdateCount int       `json:"update_count"`
	SourceCount int       `json:"source_count"`
	ErrorCount  int       `json:"error_count"`
}

type SchedulerTriggerResponse struct {
	Success bool                   `json:"success"`
	Results []SchedulerTriggerResult `json:"results,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

type SchedulerTriggerResult struct {
	SourceID   string `json:"source_id"`
	SourceName string `json:"source_name"`
	Updated    bool   `json:"updated"`
	Skipped    bool   `json:"skipped"`
	Reason     string `json:"reason,omitempty"`
	Error      string `json:"error,omitempty"`
	UpdatedAt  string `json:"updated_at"`
}
