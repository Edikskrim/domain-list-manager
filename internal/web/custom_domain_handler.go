package web

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"unicode/utf8"
	"strings"
	"time"

	"domain-list-manager/internal/api"
	"domain-list-manager/internal/builder"
	"domain-list-manager/internal/domain"
	"domain-list-manager/internal/uuid"
)

type customDomainHandler struct {
	repo         domain.Repository
	builder      *builder.Builder
}

func newCustomDomainHandler(repo domain.Repository) *customDomainHandler {
	return &customDomainHandler{repo: repo}
}

func stripComment(line string) string {
	line = strings.TrimSpace(line)
	line = strings.ReplaceAll(line, "\x00", "")
	// Remove BOM (U+FEFF, UTF-8: EF BB BF) at start
	if len(line) >= 3 && line[:3] == "\xef\xbb\xbf" {
		line = line[3:]
	}
	// Remove leading non-breaking space (UTF-8: C2 A0)
	for len(line) > 0 && line[0] == 0xC2 && len(line) > 1 && line[1] == 0xA0 {
		line = line[2:]
	}
	line = strings.TrimSpace(line)
	// Normalize fullwidth number sign (FFE3) to ASCII #
	for {
		r, width := utf8.DecodeRuneInString(line)
		if r == 0xFFE3 {
			line = "\x23" + line[width:]
		} else {
			break
		}
	}
	// Strip everything after #
	if idx := strings.Index(line, "#"); idx != -1 {
		line = strings.TrimSpace(line[:idx])
	}
	return line
}

// BulkAdd adds multiple domains from a JSON array.
func (h *customDomainHandler) BulkAdd(w http.ResponseWriter, r *http.Request) {
	var names []string
	if err := json.NewDecoder(r.Body).Decode(&names); err != nil {
		api.WriteBadRequest(w, "Invalid JSON: expected array of domain names")
		return
	}

	if len(names) == 0 {
		api.WriteBadRequest(w, "Empty domain list")
		return
	}

	now := time.Now().UTC()
	created := 0
	skipped := 0
	failed := make([]string, 0)
	sourceID := "manual"

	for _, name := range names {
		name = stripComment(name)
		if name == "" {
			continue
		}

		id := uuid.Generate()
		d := &domain.Domain{
			ID:        id,
			Name:      name,
			SourceID:  &sourceID,
			CreatedAt: now,
			UpdatedAt: now,
		}

		if err := h.repo.Create(d); err != nil {
			failed = append(failed, fmt.Sprintf("%s: %v", name, err))
			skipped++
			continue
		}
		created++
	}

	api.WriteSuccess(w, map[string]interface{}{
		"created": created,
		"skipped": skipped,
		"failed":  failed,
	}, "Bulk add completed")
}

// ImportTXT imports domains from a TXT file body.
func (h *customDomainHandler) ImportTXT(w http.ResponseWriter, r *http.Request) {
	body := r.Body
	if body == nil {
		api.WriteBadRequest(w, "No body provided")
		return
	}
	defer body.Close()

	var created int
	var failed []string

	scanner := bufio.NewScanner(body)
	now := time.Now().UTC()

	for scanner.Scan() {
		line := stripComment(scanner.Text())
		if line == "" {
			continue
		}

		id := uuid.Generate()
		d := &domain.Domain{
			ID:        id,
			Name:      line,
			CreatedAt: now,
			UpdatedAt: now,
		}

		if err := h.repo.Create(d); err != nil {
			failed = append(failed, line)
			continue
		}
		created++
	}

	if err := scanner.Err(); err != nil {
		api.WriteBadRequest(w, fmt.Sprintf("Failed to read TXT: %v", err))
		return
	}

	if created == 0 {
		api.WriteBadRequest(w, "No domains were imported")
		return
	}

	api.WriteSuccess(w, map[string]interface{}{
		"created": created,
		"failed":  failed,
	}, "TXT import completed")
}

// ImportFromURL imports domains from a remote TXT file.
func (h *customDomainHandler) ImportFromURL(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteBadRequest(w, "Invalid JSON: expected url field")
		return
	}
	if req.URL == "" {
		api.WriteBadRequest(w, "URL is required")
		return
	}

	resp, err := http.Get(req.URL)
	if err != nil {
		api.WriteBadRequest(w, fmt.Sprintf("Failed to fetch URL: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		api.WriteBadRequest(w, fmt.Sprintf("HTTP %d fetching URL", resp.StatusCode))
		return
	}

	var created int
	var failed []string
	scanner := bufio.NewScanner(resp.Body)
	now := time.Now().UTC()
	for scanner.Scan() {
		line := stripComment(scanner.Text())
		if line == "" {
			continue
		}
		id := uuid.Generate()
		d := &domain.Domain{ID: id, Name: line, CreatedAt: now, UpdatedAt: now}
		if err := h.repo.Create(d); err != nil {
			failed = append(failed, line)
			continue
		}
		created++
	}
	if err := scanner.Err(); err != nil {
		api.WriteBadRequest(w, fmt.Sprintf("Failed to read content: %v", err))
		return
	}
	if created == 0 {
		api.WriteBadRequest(w, "No domains were imported")
		return
	}
	api.WriteSuccess(w, map[string]interface{}{"created": created, "failed": failed}, "URL import completed")
}

// ExportTXT exports all domains as TXT.
func (h *customDomainHandler) ExportTXT(w http.ResponseWriter, r *http.Request) {
	if h.builder != nil {
		http.ServeFile(w, r, h.builder.OutputPath())
		return
	}

	domains, err := h.repo.List()
	if err != nil {
		api.WriteInternalServerError(w, fmt.Sprintf("Failed to list domains: %v", err))
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	for _, d := range domains {
		fmt.Fprintln(w, d.Name)
	}
}
