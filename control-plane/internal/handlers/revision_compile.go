package handlers

import (
	"context"
	"net/http"

	"waf/control-plane/internal/jobs"
	"waf/control-plane/internal/revisions"
	"waf/control-plane/internal/services"
)

type revisionCompileService interface {
	Create(ctx context.Context) (services.CompileRequestResult, error)
}

type RevisionCompileHandler struct {
	compile revisionCompileService
}

func NewRevisionCompileHandler(compile revisionCompileService) *RevisionCompileHandler {
	return &RevisionCompileHandler{compile: compile}
}

func (h *RevisionCompileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/revisions/compile" || r.Method != http.MethodPost {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	result, err := h.compile.Create(withActorIP(r))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, revisionCompileResponse{
		Revision: result.Revision,
		Job:      result.Job,
	})
}

type revisionCompileResponse struct {
	Revision revisions.Revision `json:"revision"`
	Job      jobs.Job           `json:"job"`
}
