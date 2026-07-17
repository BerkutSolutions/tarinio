package handlers

import (
	"net/http"

	"waf/control-plane/internal/services"
)

type setupStatusService interface {
	Status() (services.SetupStatus, error)
}

type SetupHandler struct {
	setup setupStatusService
}

func NewSetupHandler(setup setupStatusService) *SetupHandler {
	return &SetupHandler{setup: setup}
}

func (h *SetupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/setup/status" || r.Method != http.MethodGet {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	status, err := h.setup.Status()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	// This endpoint is intentionally public so a first-time browser can select
	// the bootstrap flow. Do not disclose user, site, or revision inventory.
	writeJSON(w, http.StatusOK, map[string]bool{"needs_bootstrap": status.NeedsBootstrap})
}
