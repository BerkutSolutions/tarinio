package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"waf/control-plane/internal/tlsconfigs"
)

type tlsConfigService interface {
	Create(ctx context.Context, item tlsconfigs.TLSConfig) (tlsconfigs.TLSConfig, error)
	List() ([]tlsconfigs.TLSConfig, error)
	Update(ctx context.Context, item tlsconfigs.TLSConfig) (tlsconfigs.TLSConfig, error)
	Delete(ctx context.Context, siteID string) error
}

type TLSConfigsHandler struct {
	tlsConfigs tlsConfigService
}

func NewTLSConfigsHandler(tlsConfigs tlsConfigService) *TLSConfigsHandler {
	return &TLSConfigsHandler{tlsConfigs: tlsConfigs}
}

func (h *TLSConfigsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/api/tls-configs" && r.Method == http.MethodGet:
		h.list(w, r)
	case r.URL.Path == "/api/tls-configs" && r.Method == http.MethodPost:
		h.create(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/tls-configs/") && r.Method == http.MethodPut:
		h.update(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/tls-configs/") && r.Method == http.MethodDelete:
		h.delete(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (h *TLSConfigsHandler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.tlsConfigs.List()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "tls config store unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *TLSConfigsHandler) create(w http.ResponseWriter, r *http.Request) {
	var item tlsconfigs.TLSConfig
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	created, err := h.tlsConfigs.Create(withActorIP(r), item)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *TLSConfigsHandler) update(w http.ResponseWriter, r *http.Request) {
	siteID := strings.TrimPrefix(r.URL.Path, "/api/tls-configs/")
	if siteID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "tls config site_id is required"})
		return
	}

	var item tlsconfigs.TLSConfig
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	item.SiteID = siteID
	updated, err := h.tlsConfigs.Update(withActorIP(r), item)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *TLSConfigsHandler) delete(w http.ResponseWriter, r *http.Request) {
	siteID := strings.TrimPrefix(r.URL.Path, "/api/tls-configs/")
	if siteID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "tls config site_id is required"})
		return
	}
	if err := h.tlsConfigs.Delete(withActorIP(r), siteID); err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
