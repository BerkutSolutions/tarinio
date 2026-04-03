package handlers

import (
	"net/http"

	"waf/control-plane/internal/services"
)

type dashboardService interface {
	Stats() (services.DashboardStats, error)
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
	stats, err := h.service.Stats()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, stats)
}
