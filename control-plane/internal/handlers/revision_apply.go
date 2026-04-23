package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"waf/control-plane/internal/auth"
	"waf/control-plane/internal/jobs"
	"waf/control-plane/internal/rbac"
	"waf/control-plane/internal/revisions"
)

type revisionApplyService interface {
	Apply(ctx context.Context, revisionID string) (jobs.Job, error)
}

type revisionDeleteService interface {
	Delete(ctx context.Context, revisionID string) error
}

type revisionApproveService interface {
	ApproveRevision(ctx context.Context, revisionID, comment string) (revisions.Revision, error)
}

type RevisionApplyHandler struct {
	apply   revisionApplyService
	delete  revisionDeleteService
	approve revisionApproveService
}

func NewRevisionApplyHandler(apply revisionApplyService, deleteService revisionDeleteService, approve revisionApproveService) *RevisionApplyHandler {
	return &RevisionApplyHandler{apply: apply, delete: deleteService, approve: approve}
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
		session, ok := auth.SessionFromContext(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "authentication required"})
			return
		}
		if !sessionHasPermission(session, rbac.PermissionRevisionsWrite) {
			writeJSON(w, http.StatusForbidden, map[string]any{"error": "permission denied"})
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
		if job.Status == jobs.StatusFailed {
			message := strings.TrimSpace(job.Result)
			if message == "" {
				message = "apply failed"
			}
			writeJSON(w, http.StatusBadRequest, map[string]any{
				"error": message,
				"job":   job,
			})
			return
		}
		writeJSON(w, http.StatusCreated, job)
		return
	}
	if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/approve") {
		revisionID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/revisions/"), "/approve")
		revisionID = strings.TrimSuffix(revisionID, "/")
		if strings.TrimSpace(revisionID) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "revision id is required"})
			return
		}
		session, ok := auth.SessionFromContext(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "authentication required"})
			return
		}
		if !sessionHasPermission(session, rbac.PermissionRevisionsApprove) {
			writeJSON(w, http.StatusForbidden, map[string]any{"error": "permission denied"})
			return
		}
		if h.approve == nil {
			writeJSON(w, http.StatusNotImplemented, map[string]any{"error": "revision approve is unavailable"})
			return
		}
		var payload struct {
			Comment string `json:"comment"`
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		result, err := h.approve.ApproveRevision(withActorIP(r), revisionID, payload.Comment)
		if err != nil {
			status := http.StatusBadRequest
			if strings.Contains(err.Error(), "not found") {
				status = http.StatusNotFound
			}
			if strings.Contains(strings.ToLower(err.Error()), "not allowed") || strings.Contains(strings.ToLower(err.Error()), "disabled") {
				status = http.StatusForbidden
			}
			writeJSON(w, status, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"revision": result, "approved": true})
		return
	}
	if r.Method == http.MethodDelete {
		revisionID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/revisions/"), "/")
		if strings.Contains(revisionID, "/") || strings.TrimSpace(revisionID) == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		session, ok := auth.SessionFromContext(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "authentication required"})
			return
		}
		if !sessionHasPermission(session, rbac.PermissionRevisionsWrite) {
			writeJSON(w, http.StatusForbidden, map[string]any{"error": "permission denied"})
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

func sessionHasPermission(session auth.SessionView, permission rbac.Permission) bool {
	required := strings.TrimSpace(string(permission))
	if required == "" {
		return true
	}
	for _, item := range session.Permissions {
		if strings.TrimSpace(item) == required {
			return true
		}
	}
	return false
}
