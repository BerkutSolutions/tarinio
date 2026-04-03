package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"waf/control-plane/internal/upstreams"
)

type upstreamService interface {
	Create(ctx context.Context, item upstreams.Upstream) (upstreams.Upstream, error)
	List() ([]upstreams.Upstream, error)
	Update(ctx context.Context, item upstreams.Upstream) (upstreams.Upstream, error)
	Delete(ctx context.Context, id string) error
}

type UpstreamsHandler struct {
	upstreams upstreamService
}

func NewUpstreamsHandler(upstreams upstreamService) *UpstreamsHandler {
	return &UpstreamsHandler{upstreams: upstreams}
}

func (h *UpstreamsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/api/upstreams" && r.Method == http.MethodGet:
		h.list(w, r)
	case r.URL.Path == "/api/upstreams" && r.Method == http.MethodPost:
		h.create(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/upstreams/") && r.Method == http.MethodPut:
		h.update(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/upstreams/") && r.Method == http.MethodDelete:
		h.delete(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (h *UpstreamsHandler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.upstreams.List()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "upstream store unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *UpstreamsHandler) create(w http.ResponseWriter, r *http.Request) {
	var item upstreams.Upstream
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	created, err := h.upstreams.Create(withActorIP(r), item)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *UpstreamsHandler) update(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/upstreams/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "upstream id is required"})
		return
	}
	var item upstreams.Upstream
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	item.ID = id
	updated, err := h.upstreams.Update(withActorIP(r), item)
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

func (h *UpstreamsHandler) delete(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/upstreams/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "upstream id is required"})
		return
	}
	if err := h.upstreams.Delete(withActorIP(r), id); err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
