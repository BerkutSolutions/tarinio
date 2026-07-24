package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"waf/control-plane/internal/antiddos"
)

type antiDDoSService interface {
	Get() (antiddos.Settings, error)
	Upsert(ctx context.Context, item antiddos.Settings) (antiddos.Settings, error)
}

type AntiDDoSHandler struct {
	service antiDDoSService
}

func NewAntiDDoSHandler(service antiDDoSService) *AntiDDoSHandler {
	return &AntiDDoSHandler{service: service}
}

func (h *AntiDDoSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/anti-ddos/settings" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.get(w, r)
	case http.MethodPut:
		h.upsert(w, r)
	case http.MethodPost:
		h.upsert(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *AntiDDoSHandler) get(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.Get()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *AntiDDoSHandler) upsert(w http.ResponseWriter, r *http.Request) {
	var item antiddos.Settings
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	if item.ConnLimit < 0 || item.RatePerSecond < 0 || item.RateBurst < 0 || item.L7RequestsPS < 0 || item.L7Burst < 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "anti-ddos limits must not be negative"})
		return
	}
	if item.EnforceL7Rate && (item.L7StatusCode < 100 || item.L7StatusCode > 599) {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "anti-ddos l7_status_code must be between 100 and 599"})
		return
	}
	for _, port := range item.Ports {
		if port <= 0 || port > 65535 {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "anti-ddos ports must be between 1 and 65535"})
			return
		}
	}
	updated, err := h.service.Upsert(withActorIP(r), item)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, updated)
}
