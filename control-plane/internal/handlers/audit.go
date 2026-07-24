package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"waf/control-plane/internal/audits"
)

type auditService interface {
	List(query audits.Query) (audits.ListResult, error)
}

type AuditHandler struct {
	audits auditService
}

func NewAuditHandler(audits auditService) *AuditHandler {
	return &AuditHandler{audits: audits}
}

func (h *AuditHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/audit" || r.Method != http.MethodGet {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	query := audits.Query{
		Action:       r.URL.Query().Get("action"),
		Category:     r.URL.Query().Get("category"),
		ActorUserID:  r.URL.Query().Get("actor_user_id"),
		ActorIP:      r.URL.Query().Get("actor_ip"),
		ResourceType: r.URL.Query().Get("resource_type"),
		ResourceID:   r.URL.Query().Get("resource_id"),
		SiteID:       r.URL.Query().Get("site_id"),
		Status:       r.URL.Query().Get("status"),
		From:         r.URL.Query().Get("from"),
		To:           r.URL.Query().Get("to"),
		Limit:        parseInt(r.URL.Query().Get("limit"), 100),
		Offset:       parseInt(r.URL.Query().Get("offset"), 0),
	}
	if query.Category != "" && query.Category != "auth" && query.Category != "rollout" && query.Category != "config" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "category must be auth, rollout, or config"})
		return
	}
	if query.Limit <= 0 {
		query.Limit = 100
	}
	if query.Limit > 500 {
		query.Limit = 500
	}
	if query.Offset < 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "offset must be non-negative"})
		return
	}
	if query.Status != "" && query.Status != string(audits.StatusSucceeded) && query.Status != string(audits.StatusFailed) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "status must be succeeded or failed"})
		return
	}
	for name, value := range map[string]string{"from": query.From, "to": query.To} {
		if strings.TrimSpace(value) == "" {
			continue
		}
		if _, err := time.Parse(time.RFC3339, value); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": name + " must be RFC3339"})
			return
		}
	}
	if query.From == "" {
		storage := CurrentStorageRetention()
		if storage.ActivityDays > 0 {
			query.From = time.Now().UTC().AddDate(0, 0, -storage.ActivityDays).Format(time.RFC3339)
		}
	}
	result, err := h.audits.List(query)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "audit store unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func parseInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
