package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type APIDocEndpoint struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	Description string `json:"description"`
	Auth        bool   `json:"auth_required"`
}

type APIDocGroup struct {
	Name        string            `json:"name"`
	Endpoints   []APIDocEndpoint  `json:"endpoints"`
}

type APIDocResponse struct {
	Version string         `json:"version"`
	BaseURL string         `json:"base_url"`
	Groups  []APIDocGroup `json:"groups"`
}

func GetAPIDocumentation(w http.ResponseWriter, r *http.Request) {
	doc := APIDocResponse{
		Version: "v1",
		BaseURL: "/api/v1",
		Groups: []APIDocGroup{
			{
				Name: "Domains",
				Endpoints: []APIDocEndpoint{
					{Method: "GET", Path: "/api/v1/domains", Description: "List all custom domains", Auth: true},
					{Method: "GET", Path: "/api/v1/domains/{id}", Description: "Get domain by ID", Auth: true},
					{Method: "POST", Path: "/api/v1/domains", Description: "Create a new domain", Auth: true},
					{Method: "PUT", Path: "/api/v1/domains/{id}", Description: "Update a domain", Auth: true},
					{Method: "DELETE", Path: "/api/v1/domains/{id}", Description: "Delete a domain", Auth: true},
					{Method: "POST", Path: "/api/v1/domains/bulk", Description: "Bulk add domains (JSON array of names)", Auth: true},
					{Method: "POST", Path: "/api/v1/domains/import-txt", Description: "Import domains from TXT file", Auth: true},
					{Method: "GET", Path: "/api/v1/domains/export-txt", Description: "Export all domains as TXT", Auth: true},
				},
			},
			{
				Name: "Sources",
				Endpoints: []APIDocEndpoint{
					{Method: "GET", Path: "/api/v1/sources", Description: "List all sources", Auth: true},
					{Method: "GET", Path: "/api/v1/sources/{id}", Description: "Get source by ID", Auth: true},
					{Method: "POST", Path: "/api/v1/sources", Description: "Create a new source", Auth: true},
					{Method: "PUT", Path: "/api/v1/sources/{id}", Description: "Update a source", Auth: true},
					{Method: "DELETE", Path: "/api/v1/sources/{id}", Description: "Delete a source", Auth: true},
					{Method: "POST", Path: "/api/v1/sources/{id}/enable", Description: "Enable a source", Auth: true},
					{Method: "POST", Path: "/api/v1/sources/{id}/disable", Description: "Disable a source", Auth: true},
				},
			},
			{
				Name: "Settings",
				Endpoints: []APIDocEndpoint{
					{Method: "GET", Path: "/api/v1/settings", Description: "List all settings", Auth: true},
					{Method: "GET", Path: "/api/v1/settings/{key}", Description: "Get setting by key", Auth: true},
					{Method: "POST", Path: "/api/v1/settings", Description: "Create a new setting", Auth: true},
					{Method: "PUT", Path: "/api/v1/settings/{key}", Description: "Update a setting", Auth: true},
					{Method: "DELETE", Path: "/api/v1/settings/{key}", Description: "Delete a setting", Auth: true},
				},
			},
			{
				Name: "Build",
				Endpoints: []APIDocEndpoint{
					{Method: "POST", Path: "/api/v1/build", Description: "Trigger a build (returns result + domains)", Auth: true},
					{Method: "POST", Path: "/api/v1/build/write", Description: "Trigger a build and write to output file", Auth: true},
					{Method: "GET", Path: "/api/v1/build/status", Description: "Get build status", Auth: true},
				},
			},
			{
				Name: "History",
				Endpoints: []APIDocEndpoint{
					{Method: "GET", Path: "/api/v1/history", Description: "List build history snapshots", Auth: true},
					{Method: "GET", Path: "/api/v1/history/{id}", Description: "Get snapshot details", Auth: true},
					{Method: "GET", Path: "/api/v1/history/{id}/domains", Description: "Get domains from snapshot", Auth: true},
					{Method: "DELETE", Path: "/api/v1/history/{id}", Description: "Delete a snapshot", Auth: true},
					{Method: "GET", Path: "/api/v1/history/diff", Description: "Diff between two snapshots (snapshot_1, snapshot_2)", Auth: true},
				},
			},
			{
				Name: "Scheduler",
				Endpoints: []APIDocEndpoint{
					{Method: "GET", Path: "/api/v1/scheduler/status", Description: "Get scheduler status", Auth: true},
					{Method: "POST", Path: "/api/v1/scheduler/start", Description: "Start scheduler", Auth: true},
					{Method: "POST", Path: "/api/v1/scheduler/stop", Description: "Stop scheduler", Auth: true},
					{Method: "POST", Path: "/api/v1/scheduler/trigger", Description: "Trigger manual update", Auth: true},
				},
			},
			{
				Name: "Diagnostics",
				Endpoints: []APIDocEndpoint{
					{Method: "GET", Path: "/api/v1/diagnostics", Description: "Run full diagnostics", Auth: true},
					{Method: "GET", Path: "/api/v1/diagnostics/source/{id}", Description: "Get source diagnostics", Auth: true},
				},
			},
			{
				Name: "Intersection",
				Endpoints: []APIDocEndpoint{
					{Method: "GET", Path: "/api/v1/intersections", Description: "Analyze domain intersections between sources", Auth: true},
				},
			},
			{
				Name: "Dashboard",
				Endpoints: []APIDocEndpoint{
					{Method: "GET", Path: "/api/v1/dashboard", Description: "Get dashboard statistics", Auth: true},
				},
			},
			{
				Name: "Auth",
				Endpoints: []APIDocEndpoint{
					{Method: "GET", Path: "/login", Description: "Login page", Auth: false},
					{Method: "POST", Path: "/login", Description: "Login (username, password)", Auth: false},
					{Method: "POST", Path: "/logout", Description: "Logout", Auth: true},
				},
			},
			{
				Name: "API",
				Endpoints: []APIDocEndpoint{
					{Method: "GET", Path: "/api/v1/docs", Description: "Get this API documentation", Auth: false},
				},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

func FormatEndpoint(e APIDocEndpoint) string {
	return fmt.Sprintf("%-8s %s", e.Method, e.Path)
}
