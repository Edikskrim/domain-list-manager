package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"domain-list-manager/internal/api"
	"domain-list-manager/internal/source"
	"domain-list-manager/internal/uuid"

	"github.com/go-chi/chi/v5"
)

type sourceHandler struct {
	service *source.Service
	repo    source.Repository
}

func newSourceHandler(service *source.Service, repo source.Repository) *sourceHandler {
	return &sourceHandler{
		service: service,
		repo:    repo,
	}
}

func (h *sourceHandler) List(w http.ResponseWriter, r *http.Request) {
	sources, err := h.repo.List()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list sources: %v", err), http.StatusInternalServerError)
		return
	}

	if sources == nil {
		sources = []*source.Source{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sources)
}

func (h *sourceHandler) GetSource(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		api.WriteBadRequest(w, "Missing id parameter")
		return
	}

	src, err := h.repo.Get(id)
	if err != nil {
		api.WriteNotFound(w, "Source not found")
		return
	}

	api.WriteSuccess(w, src, "")
}

func (h *sourceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var s source.Source
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		api.WriteBadRequest(w, "Invalid JSON")
		return
	}

	if s.Name == "" {
		api.WriteBadRequest(w, "Name is required")
		return
	}

	if s.URL == "" {
		api.WriteBadRequest(w, "URL is required")
		return
	}

	if err := h.service.ValidateURL(&s); err != nil {
		api.WriteBadRequest(w, fmt.Sprintf("Invalid URL: %v", err))
		return
	}

	if s.ParserType == "" {
		s.ParserType = "raw"
	}

	if err := h.service.ValidateParserType(s.ParserType); err != nil {
		api.WriteBadRequest(w, fmt.Sprintf("Invalid parser type: %v", err))
		return
	}

	if s.UpdateInterval == 0 {
		s.UpdateInterval = 3600
	}

	s.ID = uuid.Generate()
	now := time.Now().UTC()
	s.CreatedAt = now
	s.UpdatedAt = now
	s.LastUpdate = time.Unix(0, 0).UTC()
	s.LastUpdate = time.Unix(0, 0).UTC()

	if err := h.repo.Create(&s); err != nil {
		api.WriteInternalServerError(w, fmt.Sprintf("Failed to create source: %v", err))
		return
	}

	api.WriteCreated(w, s, "Source created")
}

func (h *sourceHandler) UpdateSource(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		api.WriteBadRequest(w, "Missing id parameter")
		return
	}

	existing, err := h.repo.Get(id)
	if err != nil {
		api.WriteNotFound(w, "Source not found")
		return
	}

	var s source.Source
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		api.WriteBadRequest(w, "Invalid JSON")
		return
	}

	if s.Name != "" {
		existing.Name = s.Name
	}
	if s.Description != "" {
		existing.Description = s.Description
	}
	if s.URL != "" {
		existing.URL = s.URL
		if err := h.service.ValidateURL(existing); err != nil {
			api.WriteBadRequest(w, fmt.Sprintf("Invalid URL: %v", err))
			return
		}
	}
	if s.ParserType != "" {
		existing.ParserType = s.ParserType
		if err := h.service.ValidateParserType(s.ParserType); err != nil {
			api.WriteBadRequest(w, fmt.Sprintf("Invalid parser type: %v", err))
			return
		}
	}
	if s.UpdateInterval > 0 {
		existing.UpdateInterval = s.UpdateInterval
	}
	if s.Enabled {
		existing.Enabled = s.Enabled
	}
	existing.UpdatedAt = time.Now().UTC()

	if err := h.repo.Update(id, existing); err != nil {
		api.WriteInternalServerError(w, fmt.Sprintf("Failed to update source: %v", err))
		return
	}

	api.WriteSuccess(w, existing, "Source updated")
}

func (h *sourceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		api.WriteBadRequest(w, "Missing id parameter")
		return
	}

	if err := h.repo.Delete(id); err != nil {
		api.WriteNotFound(w, "Source not found")
		return
	}

	api.WriteSuccess(w, nil, "Source deleted")
}

func (h *sourceHandler) Enable(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		api.WriteBadRequest(w, "Missing id parameter")
		return
	}

	if err := h.repo.Enable(id); err != nil {
		api.WriteNotFound(w, "Source not found")
		return
	}

	api.WriteSuccess(w, nil, "Source enabled")
}

func (h *sourceHandler) Disable(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		api.WriteBadRequest(w, "Missing id parameter")
		return
	}

	if err := h.repo.Disable(id); err != nil {
		api.WriteNotFound(w, "Source not found")
		return
	}

	api.WriteSuccess(w, nil, "Source disabled")
}
