package handlers

import (
	"net/http"

	"waf/control-plane/internal/services"
)

type tlsAutoRenewService interface {
	Settings() (services.TLSAutoRenewSettings, error)
	UpdateSettings(input services.TLSAutoRenewSettings) (services.TLSAutoRenewSettings, error)
}

type TLSAutoRenewHandler struct {
	service tlsAutoRenewService
}

func NewTLSAutoRenewHandler(service tlsAutoRenewService) *TLSAutoRenewHandler {
	return &TLSAutoRenewHandler{service: service}
}

func (h *TLSAutoRenewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/tls/auto-renew" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if h == nil || h.service == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "tls auto-renew service unavailable"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		settings, err := h.service.Settings()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, settings)
	case http.MethodPut:
		body, ok := readJSONBody(w, r)
		if !ok {
			return
		}
		enabled, okEnabled := body["enabled"].(bool)
		if !okEnabled {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "enabled must be boolean"})
			return
		}
		rawDays, okDays := body["renew_before_days"]
		if !okDays {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "renew_before_days is required"})
			return
		}
		days, okNumber := asInt(rawDays)
		if !okNumber {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "renew_before_days must be integer"})
			return
		}
		updated, err := h.service.UpdateSettings(services.TLSAutoRenewSettings{
			Enabled:         enabled,
			RenewBeforeDays: days,
		})
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, updated)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func asInt(value any) (int, bool) {
	switch v := value.(type) {
	case float64:
		return int(v), float64(int(v)) == v
	case int:
		return v, true
	default:
		return 0, false
	}
}
