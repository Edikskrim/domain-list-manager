package web

import (
	"net/http"
	"strings"

	"domain-list-manager/internal/api"
	"domain-list-manager/internal/diagnostics"
	"domain-list-manager/internal/dto"
)

type diagnosticsHandler struct {
	svc *diagnostics.Service
}

func newDiagnosticsHandler(svc *diagnostics.Service) *diagnosticsHandler {
	return &diagnosticsHandler{
		svc: svc,
	}
}

func (h *diagnosticsHandler) RunDiagnostics(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.RunDiagnostics()
	if err != nil {
		api.WriteInternalServerError(w, "Failed to run diagnostics")
		return
	}

	response := convertToDiagnosticsResponse(result)

	api.WriteSuccess(w, response, "Diagnostics completed")
}

func (h *diagnosticsHandler) GetSourceDiagnostics(w http.ResponseWriter, r *http.Request) {
	sourceID := strings.TrimPrefix(r.URL.Path, "/diagnostics/source/")
	if sourceID == "" || sourceID == r.URL.Path {
		api.WriteBadRequest(w, "Source ID is required")
		return
	}

	sourceID = strings.TrimSpace(sourceID)

	diag, err := h.svc.GetSourceDiagnostics(sourceID)
	if err != nil {
		api.WriteInternalServerError(w, "Failed to get source diagnostics")
		return
	}

	response := dto.SourceDiagnosticsResponse{
		ID:               diag.ID,
		Name:             diag.Name,
		Description:      diag.Description,
		Enabled:          diag.Enabled,
		LastUpdate:       diag.LastUpdate,
		LastError:        diag.LastError,
		DomainCount:      diag.DomainCount,
		ParserType:       diag.ParserType,
		UpdateInterval:   diag.UpdateInterval,
		CreatedAt:        diag.CreatedAt,
		UpdatedAt:        diag.UpdatedAt,
		TotalDomainsInDB: diag.TotalDomainsInDB,
		LastFetchSize:    diag.LastFetchSize,
		LastFetchError:   diag.LastFetchError,
		LastParseError:   diag.LastParseError,
		LastParseCount:   diag.LastParseCount,
	}

	if diag.LastFetchSuccess != nil {
		response.LastFetchSuccess = diag.LastFetchSuccess
	}

	if diag.ParsedSuccessfully != nil {
		response.ParsedSuccessfully = diag.ParsedSuccessfully
	}

	api.WriteSuccess(w, response, "")
}

func convertToDiagnosticsResponse(result *diagnostics.DiagnosticsResult) dto.DiagnosticsResponse {
	response := dto.DiagnosticsResponse{
		Intersections: dto.IntersectionDiagnosticsDTO{
			TotalIntersections:  result.Intersections.TotalIntersections,
			IntersectingDomains: make([]dto.IntersectingDomainSummaryDTO, len(result.Intersections.IntersectingDomains)),
			Summary: dto.DiagnosticsReportSummaryDTO{
				TotalSources:      result.Intersections.Summary.TotalSources,
				TotalDomains:      result.Intersections.Summary.TotalDomains,
				IntersectingCount: result.Intersections.Summary.IntersectingCount,
				UniqueCount:       result.Intersections.Summary.UniqueCount,
			},
			SourceDomains: make([]dto.DiagnosticsSourceDomainInfoDTO, len(result.Intersections.SourceDomains)),
		},
		ParsingErrors:  make([]dto.ParsingErrorDTO, len(result.ParsingErrors)),
		InvalidDomains: make([]dto.InvalidDomainReportDTO, len(result.InvalidDomains)),
		OverallSummary: dto.OverallSummaryDTO{
			TotalSources:       result.OverallSummary.TotalSources,
			EnabledSources:     result.OverallSummary.EnabledSources,
			TotalDomains:       result.OverallSummary.TotalDomains,
			IntersectingCount:  result.OverallSummary.IntersectingCount,
			ParsingErrorCount:  result.OverallSummary.ParsingErrorCount,
			InvalidDomainCount: result.OverallSummary.InvalidDomainCount,
		},
		AnalyzedAt: result.AnalyzedAt,
	}

	for i, d := range result.Intersections.IntersectingDomains {
		response.Intersections.IntersectingDomains[i] = dto.IntersectingDomainSummaryDTO{
			Domain:      d.Domain,
			SourceCount: d.SourceCount,
			Sources:     d.Sources,
		}
	}

	for i, s := range result.Intersections.SourceDomains {
		response.Intersections.SourceDomains[i] = dto.DiagnosticsSourceDomainInfoDTO{
			SourceID:    s.SourceID,
			SourceName:  s.SourceName,
			DomainCount: s.DomainCount,
		}
	}

	for i, e := range result.ParsingErrors {
		response.ParsingErrors[i] = dto.ParsingErrorDTO{
			SourceID:   e.SourceID,
			SourceName: e.SourceName,
			Error:      e.Error,
		}
	}

	for i, d := range result.InvalidDomains {
		response.InvalidDomains[i] = dto.InvalidDomainReportDTO{
			ID:     d.ID,
			Domain: d.Domain,
			Reason: d.Reason,
			Source: d.Source,
			Count:  d.Count,
		}
	}

	return response
}
