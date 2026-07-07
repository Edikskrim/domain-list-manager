package web

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"domain-list-manager/internal/api"
	"domain-list-manager/internal/history"

	"github.com/go-chi/chi/v5"
)

type historyHandler struct {
	service    *history.HistoryService
	snapshotDB *sql.DB
}

func newHistoryHandler(service *history.HistoryService, snapshotDB *sql.DB) *historyHandler {
	return &historyHandler{
		service:    service,
		snapshotDB: snapshotDB,
	}
}

func (h *historyHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.WriteError(w, "Method not allowed", http.StatusMethodNotAllowed, "")
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		parsed, err := strconv.Atoi(limitStr)
		if err == nil && parsed > 0 && parsed <= 500 {
			limit = parsed
		}
	}

	snapshots, err := h.service.ListSnapshots(limit)
	if err != nil {
		api.WriteInternalServerError(w, fmt.Sprintf("Failed to list snapshots: %v", err))
		return
	}

	api.WriteSuccess(w, map[string]interface{}{
		"count":     len(snapshots),
		"snapshots": snapshots,
	}, "Snapshots listed")
}

func (h *historyHandler) Get(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.WriteError(w, "Method not allowed", http.StatusMethodNotAllowed, "")
		return
	}

	id := chi.URLParam(r, "id")

	snapshot, err := h.service.GetSnapshot(id)
	if err != nil {
		api.WriteNotFound(w, "Snapshot not found")
		return
	}

	api.WriteSuccess(w, snapshot, "")
}

func (h *historyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		api.WriteError(w, "Method not allowed", http.StatusMethodNotAllowed, "")
		return
	}

	id := chi.URLParam(r, "id")

	if err := h.service.DeleteSnapshot(id); err != nil {
		api.WriteNotFound(w, "Snapshot not found")
		return
	}

	api.WriteSuccess(w, nil, "Snapshot deleted")
}

func (h *historyHandler) Diff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.WriteError(w, "Method not allowed", http.StatusMethodNotAllowed, "")
		return
	}

	id1 := r.URL.Query().Get("snapshot_1")
	id2 := r.URL.Query().Get("snapshot_2")

	if id1 == "" || id2 == "" {
		api.WriteBadRequest(w, "Both snapshot_1 and snapshot_2 query parameters are required")
		return
	}

	if id1 == id2 {
		api.WriteBadRequest(w, "Cannot diff a snapshot against itself")
		return
	}

	diff, err := h.service.DiffSnapshots(id1, id2)
	if err != nil {
		api.WriteInternalServerError(w, fmt.Sprintf("Failed to diff snapshots: %v", err))
		return
	}

	api.WriteSuccess(w, diff, "Diff calculated")
}

func (h *historyHandler) GetDomains(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.WriteError(w, "Method not allowed", http.StatusMethodNotAllowed, "")
		return
	}

	id := chi.URLParam(r, "id")

	snapshot, err := h.service.GetSnapshot(id)
	if err != nil {
		api.WriteNotFound(w, "Snapshot not found")
		return
	}

	api.WriteSuccess(w, snapshot.Domains, "")
}
