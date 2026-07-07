package web

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"domain-list-manager/internal/api"
	"domain-list-manager/internal/builder"
	"domain-list-manager/internal/dto"
	"domain-list-manager/internal/history"
	"domain-list-manager/internal/source"
)

type builderHandler struct {
	builder    *builder.Builder
	source     source.Repository
	historySvc *history.HistoryService
}

func newBuilderHandler(builder *builder.Builder, source source.Repository, historySvc *history.HistoryService) *builderHandler {
	return &builderHandler{
		builder:    builder,
		source:     source,
		historySvc: historySvc,
	}
}

func (h *builderHandler) Build(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.WriteError(w, "Method not allowed", http.StatusMethodNotAllowed, "")
		return
	}

	result, err := h.builder.Build()
	if err != nil {
		api.WriteInternalServerError(w, fmt.Sprintf("Build failed: %v", err))
		return
	}

	resp := dto.BuildResponse{
		Success:      len(result.Errors) == 0,
		Domains:      result.Domains,
		TotalDomains: result.TotalDomains,
		TotalSources: result.TotalSources,
		TotalFetched: result.TotalFetched,
		TotalParsed:  result.TotalParsed,
		Duplicates:   result.Duplicates,
		Errors:       result.Errors,
		BuildTimeMs:  result.BuildTime.Milliseconds(),
	}

	if !resp.Success && len(resp.Errors) == 0 {
		resp.Errors = []string{"Build completed with errors"}
	}

	api.WriteSuccess(w, resp, "Build completed")

	err = h.builder.WriteOutput(result)
	if err != nil {
		resp.Errors = append(resp.Errors, fmt.Sprintf("failed to write output: %v", err))
	}

	if h.historySvc != nil {
		err := h.historySvc.SaveSnapshot(&history.BuildInfo{
			BuildTime:    time.Now(),
			TotalDomains: result.TotalDomains,
			TotalSources: result.TotalSources,
			TotalFetched: result.TotalFetched,
			TotalParsed:  result.TotalParsed,
			Duplicates:   result.Duplicates,
			Errors:       fmt.Sprintf("%v", result.Errors),
			BuildTimeMs:  result.BuildTime.Milliseconds(),
			Domains:      result.Domains,
		})
		if err != nil {
			fmt.Printf("failed to save snapshot: %v\n", err)
		}
	}
}

func (h *builderHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.WriteError(w, "Method not allowed", http.StatusMethodNotAllowed, "")
		return
	}

	sources, err := h.source.List()
	if err != nil {
		api.WriteInternalServerError(w, fmt.Sprintf("Failed to list sources: %v", err))
		return
	}

	outputPath := h.builder.OutputPath()
	hasOutput, _ := fileExists(outputPath)

	resp := dto.BuildStatusResponse{
		TotalSources: len(sources),
		HasOutput:    hasOutput,
		OutputPath:   outputPath,
	}

	for _, s := range sources {
		if s.DomainCount > 0 {
			resp.TotalDomains += s.DomainCount
		}
		if !s.LastUpdate.IsZero() {
			if resp.LastBuildTime == "" {
				resp.LastBuildTime = s.LastUpdate.Format(time.RFC3339)
			} else {
				if prevTime, err := time.Parse(time.RFC3339, resp.LastBuildTime); err == nil {
					if s.LastUpdate.After(prevTime) {
						resp.LastBuildTime = s.LastUpdate.Format(time.RFC3339)
					}
				}
			}
		}
	}

	api.WriteSuccess(w, resp, "")
}

func (h *builderHandler) WriteOutput(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.WriteError(w, "Method not allowed", http.StatusMethodNotAllowed, "")
		return
	}

	result, err := h.builder.Build()
	if err != nil {
		api.WriteInternalServerError(w, fmt.Sprintf("Build failed: %v", err))
		return
	}

	if err := h.builder.WriteOutput(result); err != nil {
		api.WriteInternalServerError(w, fmt.Sprintf("Failed to write output: %v", err))
		return
	}

	if h.historySvc != nil {
		err := h.historySvc.SaveSnapshot(&history.BuildInfo{
			BuildTime:    time.Now(),
			TotalDomains: result.TotalDomains,
			TotalSources: result.TotalSources,
			TotalFetched: result.TotalFetched,
			TotalParsed:  result.TotalParsed,
			Duplicates:   result.Duplicates,
			Errors:       fmt.Sprintf("%v", result.Errors),
			BuildTimeMs:  result.BuildTime.Milliseconds(),
			Domains:      result.Domains,
		})
		if err != nil {
			fmt.Printf("failed to save snapshot: %v\n", err)
		}
	}

	api.WriteSuccess(w, map[string]interface{}{
		"success":      true,
		"total_domains": result.TotalDomains,
		"output_path":  h.builder.OutputPath(),
	}, "Build completed and output written")
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
