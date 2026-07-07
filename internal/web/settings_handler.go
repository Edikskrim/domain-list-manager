package web

import (
	"encoding/json"
	"fmt"
	"net/http"

	"domain-list-manager/internal/api"
	"domain-list-manager/internal/setting"

	"github.com/go-chi/chi/v5"
)

type settingsHandler struct {
	service *setting.Service
	repo    setting.Repository
}

func newSettingsHandler(service *setting.Service, repo setting.Repository) *settingsHandler {
	return &settingsHandler{
		service: service,
		repo:    repo,
	}
}

func (h *settingsHandler) List(w http.ResponseWriter, r *http.Request) {
	settings, err := h.service.GetMap()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list settings: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}

func (h *settingsHandler) GetSetting(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if key == "" {
		api.WriteBadRequest(w, "Missing key parameter")
		return
	}

	setting, err := h.repo.Get(key)
	if err != nil {
		api.WriteNotFound(w, "Setting not found")
		return
	}

	api.WriteSuccess(w, setting, "")
}

func (h *settingsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var s setting.Setting
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		api.WriteBadRequest(w, "Invalid JSON")
		return
	}

	if s.Key == "" {
		api.WriteBadRequest(w, "Key is required")
		return
	}

	if err := h.repo.Create(&s); err != nil {
		api.WriteInternalServerError(w, fmt.Sprintf("Failed to create setting: %v", err))
		return
	}

	api.WriteCreated(w, s, "Setting created")
}

func (h *settingsHandler) UpdateSetting(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if key == "" {
		api.WriteBadRequest(w, "Missing key parameter")
		return
	}

	var s setting.Setting
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		api.WriteBadRequest(w, "Invalid JSON")
		return
	}

	s.Key = key
	if err := h.repo.Update(key, &s); err != nil {
		api.WriteNotFound(w, "Setting not found")
		return
	}

	api.WriteSuccess(w, s, "Setting updated")
}

func (h *settingsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if key == "" {
		api.WriteBadRequest(w, "Missing key parameter")
		return
	}

	if err := h.repo.Delete(key); err != nil {
		api.WriteNotFound(w, "Setting not found")
		return
	}

	api.WriteSuccess(w, nil, "Setting deleted")
}
