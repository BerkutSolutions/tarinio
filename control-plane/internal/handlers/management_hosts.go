package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"waf/control-plane/internal/managementhosts"
)

type managementHostsService interface {
	Get() (managementhosts.Settings, error)
	Update(context.Context, []string, int64) (managementhosts.Settings, error)
}
type ManagementHostsHandler struct{ service managementHostsService }

func NewManagementHostsHandler(service managementHostsService) *ManagementHostsHandler {
	return &ManagementHostsHandler{service: service}
}

func (h *ManagementHostsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/settings/management-hosts" {
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
		writeJSON(w, http.StatusOK, item)
	case http.MethodPut:
		var input struct {
			Hosts   []string `json:"management_hosts"`
			Version int64    `json:"version"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
			return
		}
		item, err := h.service.Update(withActorIP(r), input.Hosts, input.Version)
		if err != nil {
			if errors.Is(err, managementhosts.ErrVersionConflict) {
				writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, item)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
