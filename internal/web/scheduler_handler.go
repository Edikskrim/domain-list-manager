package web

import (
	"fmt"
	"net/http"

	"domain-list-manager/internal/api"
	"domain-list-manager/internal/dto"
	"domain-list-manager/internal/scheduler"
)

type schedulerHandler struct {
	scheduler *scheduler.Scheduler
}

func newSchedulerHandler(svc *scheduler.Scheduler) *schedulerHandler {
	return &schedulerHandler{scheduler: svc}
}

func (h *schedulerHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	status := h.scheduler.GetStatus()

	response := dto.SchedulerStatusResponse{
		Running:     status.Running,
		LastUpdate:  status.LastUpdate,
		NextUpdate:  status.NextUpdate,
		UpdateTime:  status.UpdateTime.Milliseconds(),
		UpdateCount: status.UpdateCount,
		SourceCount: status.SourceCount,
		ErrorCount:  status.ErrorCount,
	}

	api.WriteSuccess(w, response, "")
}

func (h *schedulerHandler) Start(w http.ResponseWriter, r *http.Request) {
	if err := h.scheduler.Start(); err != nil {
		api.WriteError(w, fmt.Sprintf("Failed to start scheduler: %v", err), http.StatusBadRequest, "")
		return
	}

	api.WriteSuccess(w, map[string]string{"running": "true"}, "Scheduler started")
}

func (h *schedulerHandler) Stop(w http.ResponseWriter, r *http.Request) {
	if err := h.scheduler.Stop(); err != nil {
		api.WriteError(w, fmt.Sprintf("Failed to stop scheduler: %v", err), http.StatusBadRequest, "")
		return
	}

	api.WriteSuccess(w, map[string]string{"running": "false"}, "Scheduler stopped")
}

func (h *schedulerHandler) TriggerUpdate(w http.ResponseWriter, r *http.Request) {
	results, err := h.scheduler.TriggerUpdate()
	if err != nil {
		api.WriteError(w, fmt.Sprintf("Trigger update failed: %v", err), http.StatusBadRequest, "")
		return
	}

	for _, r := range results {
		fmt.Printf("[trigger] source=%s updated=%v skipped=%v reason=%s error=%s\n", r.SourceName, r.Updated, r.Skipped, r.Reason, r.Error)
	}

	triggerResults := make([]dto.SchedulerTriggerResult, 0, len(results))
	for _, r := range results {
		triggerResults = append(triggerResults, dto.SchedulerTriggerResult{
			SourceID:   r.SourceID,
			SourceName: r.SourceName,
			Updated:    r.Updated,
			Skipped:    r.Skipped,
			Reason:     r.Reason,
			Error:      r.Error,
			UpdatedAt:  r.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	hasError := false
	for _, r := range results {
		if r.Error != "" {
			hasError = true
			break
		}
	}

	api.WriteSuccess(w, dto.SchedulerTriggerResponse{
		Success: !hasError,
		Results: triggerResults,
	}, "Update triggered")
}
