package web

import (
	"database/sql"
	"net/http"
	"time"

	"domain-list-manager/internal/api"
	"domain-list-manager/internal/dto"
	"domain-list-manager/internal/dashboard"
)

type dashboardHandler struct {
	svc *dashboard.Service
}

func newDashboardHandler(db *sql.DB) *dashboardHandler {
	return &dashboardHandler{
		svc: dashboard.NewService(db),
	}
}

func (h *dashboardHandler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	data, err := h.svc.GetDashboardData()
	if err != nil {
		api.WriteInternalServerError(w, "Failed to get dashboard data")
		return
	}

	response := convertToDashboardResponse(data)

	api.WriteSuccess(w, response, "")
}

func convertToDashboardResponse(data *dashboard.DashboardData) dto.DashboardResponse {
	response := dto.DashboardResponse{
		TotalSources:    data.TotalSources,
		EnabledSources:  data.EnabledSources,
		DisabledSources: data.DisabledSources,
		TotalDomains:    data.TotalDomains,
		RecentErrors:    data.RecentErrors,
		UpdatedAt:       data.UpdatedAt.Format(time.RFC3339),
	}

	if data.LastBuild != nil {
		response.LastBuild = &dto.LastBuildInfo{
			ID:           data.LastBuild.ID,
			BuildTime:    data.LastBuild.BuildTime,
			TotalDomains: data.LastBuild.TotalDomains,
			TotalSources: data.LastBuild.TotalSources,
			TotalFetched: data.LastBuild.TotalFetched,
			TotalParsed:  data.LastBuild.TotalParsed,
			Duplicates:   data.LastBuild.Duplicates,
			Errors:       data.LastBuild.Errors,
			BuildTimeMs:  data.LastBuild.BuildTimeMs,
			CreatedAt:    data.LastBuild.CreatedAt,
		}
	}

	if data.BuildStats.TotalBuilds > 0 {
		response.BuildStats = dto.BuildStatistics{
			TotalBuilds:    data.BuildStats.TotalBuilds,
			AvgDomains:     data.BuildStats.AvgDomains,
			AvgFetched:     data.BuildStats.AvgFetched,
			AvgDuplicates:  data.BuildStats.AvgDuplicates,
			LastBuildDate:  data.BuildStats.LastBuildDate,
			FirstBuildDate: data.BuildStats.FirstBuildDate,
			AvgBuildTimeMs: data.BuildStats.AvgBuildTimeMs,
		}
	}

	if len(data.SourceStatus) > 0 {
		response.SourceStatus = make([]dto.SourceStatusDTO, len(data.SourceStatus))
		for i, status := range data.SourceStatus {
			response.SourceStatus[i] = dto.SourceStatusDTO{
				ID:          status.ID,
				Name:        status.Name,
				Enabled:     status.Enabled,
				DomainCount: status.DomainCount,
				LastUpdate:  status.LastUpdate,
				LastError:   status.LastError,
				UpdatedAt:   status.UpdatedAt,
			}
		}
	}

	response.StorageUsage = dto.StorageUsageDTO{
		SnapshotCount:     data.StorageUsage.SnapshotCount,
		TotalDomainsCount: data.StorageUsage.TotalDomainsCount,
	}

	return response
}
