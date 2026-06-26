package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"waf/control-plane/internal/jobs"
	"waf/control-plane/internal/revisions"
	"waf/control-plane/internal/services"
)

type revisionCompileService interface {
	Create(ctx context.Context) (services.CompileRequestResult, error)
	CreateWithTargets(ctx context.Context, targetSiteIDs []string) (services.CompileRequestResult, error)
}

type RevisionCompileHandler struct {
	compile revisionCompileService
}

func NewRevisionCompileHandler(compile revisionCompileService) *RevisionCompileHandler {
	return &RevisionCompileHandler{compile: compile}
}

type revisionCompileRequest struct {
	TargetSiteIDs []string `json:"target_site_ids,omitempty"`
}

func (h *RevisionCompileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/revisions/compile" || r.Method != http.MethodPost {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	targetSiteIDs := decodeCompileTargetSiteIDs(r)

	result, err := h.compile.CreateWithTargets(withActorIP(r), targetSiteIDs)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, revisionCompileResponse{
		Revision: result.Revision,
		Job:      result.Job,
	})
}

func decodeCompileTargetSiteIDs(r *http.Request) []string {
	if r.Body == nil {
		return nil
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
	if err != nil || len(body) == 0 {
		return nil
	}
	var payload revisionCompileRequest
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}
	out := make([]string, 0, len(payload.TargetSiteIDs))
	for _, raw := range payload.TargetSiteIDs {
		token := strings.TrimSpace(raw)
		if token != "" {
			out = append(out, token)
		}
	}
	return out
}

type revisionCompileResponse struct {
	Revision revisions.Revision `json:"revision"`
	Job      jobs.Job           `json:"job"`
}
