package web

import (
	"fmt"
	"net/http"
	"strings"

	"domain-list-manager/internal/api"
	"domain-list-manager/internal/dto"
	"domain-list-manager/internal/intersection"

	"github.com/go-chi/chi/v5"
)

type intersectionHandler struct {
	service *intersection.Service
}

func newIntersectionHandler(service *intersection.Service) *intersectionHandler {
	return &intersectionHandler{service: service}
}

func (h *intersectionHandler) Analyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.WriteError(w, "Method not allowed", http.StatusMethodNotAllowed, "")
		return
	}

	result, err := h.service.Analyze()
	if err != nil {
		api.WriteInternalServerError(w, fmt.Sprintf("Analyze intersections: %v", err))
		return
	}

	dto := convertToDTO(result.Report)

	api.WriteSuccess(w, dto, "Intersection analysis completed")
}

func (h *intersectionHandler) GetSourcesForDomain(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.WriteMethodNotAllowed(w, "GET")
		return
	}

	domainName := chi.URLParam(r, "domain")
	if domainName == "" {
		api.WriteBadRequest(w, "Domain name is required")
		return
	}

	domainName = strings.ToLower(strings.TrimSpace(domainName))

	domains, err := h.service.GetDomainsByName(domainName)
	if err != nil {
		api.WriteInternalServerError(w, fmt.Sprintf("Failed to get domains: %v", err))
		return
	}

	var result []map[string]interface{}
	for _, d := range domains {
		var sourceID string
		if d.SourceID != nil {
			sourceID = *d.SourceID
		}
		result = append(result, map[string]interface{}{
			"id":         d.ID,
			"name":       d.Name,
			"source_id":  sourceID,
		})
	}

	api.WriteSuccess(w, map[string]interface{}{
		"domain": domainName,
		"sources": result,
	}, "")
}

func convertToDTO(report *intersection.IntersectionReport) *dto.IntersectionReportDTO {
	intersecting := make([]dto.IntersectingDomainDTO, 0, len(report.IntersectingDomains))
	for _, d := range report.IntersectingDomains {
		intersecting = append(intersecting, dto.IntersectingDomainDTO{
			Domain:      d.Domain,
			SourceCount: d.SourceCount,
			Sources:     d.Sources,
		})
	}

	unique := make([]dto.UniqueDomainDTO, 0, len(report.UniqueDomains))
	for _, d := range report.UniqueDomains {
		unique = append(unique, dto.UniqueDomainDTO{
			Domain: d.Domain,
			Source: d.Source,
		})
	}

	var sourceDomains []dto.SourceDomainInfoDTO
	for _, sd := range report.SourceDomains {
		sourceDomains = append(sourceDomains, dto.SourceDomainInfoDTO{
			SourceID:    sd.SourceID,
			SourceName:  sd.SourceName,
			DomainCount: sd.DomainCount,
		})
	}

	return &dto.IntersectionReportDTO{
		IntersectingDomains: intersecting,
		UniqueDomains:       unique,
		Summary: dto.ReportSummaryDTO{
			TotalSources:      report.Summary.TotalSources,
			TotalDomains:      report.Summary.TotalDomains,
			IntersectingCount: report.Summary.IntersectingCount,
			UniqueCount:       report.Summary.UniqueCount,
		},
		SourceDomains: sourceDomains,
		AnalyzedAt:    report.AnalyzedAt,
	}
}
