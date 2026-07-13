package handlers

import (
	"net/http"

	"waf/control-plane/internal/services"
)

type managementSafeguardStatuser interface {
	Status() (services.ManagementSafeguardStatus, error)
}
type ManagementSafeguardStatusHandler struct{ service managementSafeguardStatuser }

func NewManagementSafeguardStatusHandler(service managementSafeguardStatuser) *ManagementSafeguardStatusHandler {
	return &ManagementSafeguardStatusHandler{service: service}
}
func (h *ManagementSafeguardStatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	item, err := h.service.Status()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "management safeguard status unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, item)
}
