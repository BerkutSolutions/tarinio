package handlers

import (
	"net/http"
	"strings"
)

const defaultHealthcheckAppearance = "variant-1"

func normalizeHealthcheckAppearance(value string) string {
	switch strings.TrimSpace(value) {
	case "variant-1", "variant-2", "variant-3", "variant-4", "variant-5":
		return strings.TrimSpace(value)
	default:
		return defaultHealthcheckAppearance
	}
}

func CurrentHealthcheckAppearance() string {
	runtimeSettingsState.mu.RLock()
	defer runtimeSettingsState.mu.RUnlock()
	return normalizeHealthcheckAppearance(runtimeSettingsState.healthcheckAppearance)
}

type HealthcheckAppearanceHandler struct{}

func (h *HealthcheckAppearanceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"healthcheck_appearance": CurrentHealthcheckAppearance()})
}
