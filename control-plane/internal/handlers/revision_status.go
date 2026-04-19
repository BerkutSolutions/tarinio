package handlers

import (
	"context"
	"net/http"
)

type revisionStatusService interface {
	ClearTimeline(ctx context.Context) error
}

type RevisionStatusHandler struct {
	service revisionStatusService
}

func NewRevisionStatusHandler(service revisionStatusService) *RevisionStatusHandler {
	return &RevisionStatusHandler{service: service}
}

func (h *RevisionStatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/revisions/statuses" || r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err := h.service.ClearTimeline(withActorIP(r)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"cleared": true})
}
