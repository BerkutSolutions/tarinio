package handlers

import (
	"net/http"
	"net/url"
	"strings"

	"waf/control-plane/internal/services"
)

type dashboardService interface {
	Stats() (services.DashboardStats, error)
	Probe(kind string, query url.Values) error
}

type DashboardHandler struct {
	service dashboardService
}

func NewDashboardHandler(service dashboardService) *DashboardHandler {
	return &DashboardHandler{service: service}
}

func (h *DashboardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/dashboard/stats" || r.Method != http.MethodGet {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if probeKind := strings.TrimSpace(r.URL.Query().Get("probe")); probeKind != "" {
		if err := h.service.Probe(probeKind, r.URL.Query()); err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}
	stats, err := h.service.Stats()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, stats)
}
