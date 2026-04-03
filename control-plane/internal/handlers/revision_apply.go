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

type RevisionApplyHandler struct {
	apply revisionApplyService
}

func NewRevisionApplyHandler(apply revisionApplyService) *RevisionApplyHandler {
	return &RevisionApplyHandler{apply: apply}
}

func (h *RevisionApplyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, "/api/revisions/") || !strings.HasSuffix(r.URL.Path, "/apply") || r.Method != http.MethodPost {
		w.WriteHeader(http.StatusNotFound)
		return
	}

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
}
