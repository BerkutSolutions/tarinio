package handlers

import (
	"encoding/json"
	"net/http"
)

type healthService interface {
	RevisionCount() (int, error)
}

type HealthHandler struct {
	revisions healthService
}

func NewHealthHandler(revisions healthService) *HealthHandler {
	return &HealthHandler{revisions: revisions}
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	count, err := h.revisions.RevisionCount()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"status": "degraded",
			"error":  "revision store unavailable",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":         "ok",
		"revision_count": count,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	writeJSON(w, status, payload)
}
