package handlers

import (
	"context"
	"net/http"

	"waf/control-plane/internal/audits"
	"waf/control-plane/internal/auth"
	"waf/control-plane/internal/services"
)

func writeOWASPCRSError(w http.ResponseWriter, err error) {
	code := services.RuntimeCRSErrorCode(err)
	if code == "" {
		code = "crs_update_failed"
	}
	writeJSON(w, http.StatusBadGateway, map[string]string{
		"code":  code,
		"error": "CRS update request failed",
	})
}

type owaspCRSService interface {
	Status(ctx context.Context) (services.RuntimeCRSStatus, error)
	CheckUpdates(ctx context.Context, dryRun bool) (services.RuntimeCRSStatus, error)
	Update(ctx context.Context) (services.RuntimeCRSStatus, error)
	SetHourlyAutoUpdate(ctx context.Context, enabled bool) (services.RuntimeCRSStatus, error)
}

type OWASPCRSHandler struct {
	service owaspCRSService
	audits  interface{ Emit(audits.AuditEvent) }
}

func NewOWASPCRSHandler(service owaspCRSService) *OWASPCRSHandler {
	return &OWASPCRSHandler{service: service}
}

func (h *OWASPCRSHandler) SetAuditEmitter(emitter interface{ Emit(audits.AuditEvent) }) {
	h.audits = emitter
}

func (h *OWASPCRSHandler) emitAudit(r *http.Request, action string, err error) {
	if h.audits == nil {
		return
	}
	actor := ""
	if session, ok := auth.SessionFromContext(r.Context()); ok {
		actor = session.UserID
	}
	status := audits.StatusSucceeded
	if err != nil {
		status = audits.StatusFailed
	}
	h.audits.Emit(audits.AuditEvent{ActorUserID: actor, ActorIP: audits.ActorIPFromContext(r.Context()), Action: action, ResourceType: "owasp_crs", ResourceID: "runtime", Status: status, Summary: action})
}

func (h *OWASPCRSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/api/owasp-crs/status" && r.Method == http.MethodGet:
		status, err := h.service.Status(r.Context())
		if err != nil {
			writeOWASPCRSError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, status)
	case r.URL.Path == "/api/owasp-crs/check-updates" && r.Method == http.MethodPost:
		body, ok := readJSONBody(w, r)
		if !ok {
			return
		}
		dryRun := false
		if raw, exists := body["dry_run"]; exists {
			value, typeOK := raw.(bool)
			if !typeOK {
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": "dry_run must be boolean"})
				return
			}
			dryRun = value
		}
		status, err := h.service.CheckUpdates(r.Context(), dryRun)
		if err != nil {
			writeOWASPCRSError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, status)
	case r.URL.Path == "/api/owasp-crs/update" && r.Method == http.MethodPost:
		body, ok := readJSONBody(w, r)
		if !ok {
			return
		}
		if raw, exists := body["enable_hourly_auto_update"]; exists {
			value, typeOK := raw.(bool)
			if !typeOK {
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": "enable_hourly_auto_update must be boolean"})
				return
			}
			status, err := h.service.SetHourlyAutoUpdate(r.Context(), value)
			h.emitAudit(r, "owasp_crs.hourly_auto_update", err)
			if err != nil {
				writeOWASPCRSError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, status)
			return
		}
		status, err := h.service.Update(r.Context())
		h.emitAudit(r, "owasp_crs.update", err)
		if err != nil {
			writeOWASPCRSError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, status)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
