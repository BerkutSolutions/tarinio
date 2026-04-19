package handlers

import (
	"context"
	"net/http"
	"strings"

	"waf/control-plane/internal/jobs"
)

type revisionApplyService interface {
	Apply(ctx context.Context, revisionID string) (jobs.Job, error)
}

type revisionDeleteService interface {
	Delete(ctx context.Context, revisionID string) error
}

type RevisionApplyHandler struct {
	apply  revisionApplyService
	delete revisionDeleteService
}

func NewRevisionApplyHandler(apply revisionApplyService, deleteService revisionDeleteService) *RevisionApplyHandler {
	return &RevisionApplyHandler{apply: apply, delete: deleteService}
}

func (h *RevisionApplyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/api/revisions/") {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/apply") {
		revisionID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/revisions/"), "/apply")
		revisionID = strings.TrimSuffix(revisionID, "/")
		if strings.TrimSpace(revisionID) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "revision id is required"})
			return
		}

		job, err := h.apply.Apply(withActorIP(r), revisionID)
		if err != nil {
			status := http.StatusBadRequest
			if strings.Contains(err.Error(), "not found") {
				status = http.StatusNotFound
			}
			writeJSON(w, status, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, job)
		return
	}
	if r.Method == http.MethodDelete {
		revisionID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/revisions/"), "/")
		if strings.Contains(revisionID, "/") || strings.TrimSpace(revisionID) == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if h.delete == nil {
			writeJSON(w, http.StatusNotImplemented, map[string]any{"error": "revision delete is unavailable"})
			return
		}
		err := h.delete.Delete(withActorIP(r), revisionID)
		if err != nil {
			status := http.StatusBadRequest
			switch {
			case strings.Contains(err.Error(), "not found"):
				status = http.StatusNotFound
			case strings.Contains(err.Error(), "cannot be deleted"):
				status = http.StatusConflict
			}
			writeJSON(w, status, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"deleted": true, "revision_id": revisionID})
		return
	}
	w.WriteHeader(http.StatusNotFound)
}
