package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"waf/control-plane/internal/managementhosts"
)

type directIPAccessService interface {
	Get() (managementhosts.Settings, error)
	UpdateDirectIPAccess(context.Context, bool) (managementhosts.Settings, error)
}

type DirectIPAccessHandler struct {
	service directIPAccessService
	apply   func(context.Context) error
}

func NewDirectIPAccessHandler(service directIPAccessService, apply func(context.Context) error) *DirectIPAccessHandler {
	return &DirectIPAccessHandler{service: service, apply: apply}
}

func (h *DirectIPAccessHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/settings/direct-ip-access" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	switch r.Method {
	case http.MethodGet:
		item, err := h.service.Get()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"block_direct_ip_access": item.BlockDirectIPAccess})
	case http.MethodPut:
		var input struct {
			BlockDirectIPAccess *bool `json:"block_direct_ip_access"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.BlockDirectIPAccess == nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "block_direct_ip_access must be boolean"})
			return
		}
		item, err := h.service.UpdateDirectIPAccess(withActorIP(r), *input.BlockDirectIPAccess)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if h.apply != nil {
			if err := h.apply(r.Context()); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "policy saved but runtime apply failed: " + err.Error()})
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]bool{"block_direct_ip_access": item.BlockDirectIPAccess})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
