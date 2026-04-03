package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"waf/control-plane/internal/accesspolicies"
	"waf/control-plane/internal/sites"
)

type siteService interface {
	Create(ctx context.Context, site sites.Site) (sites.Site, error)
	List() ([]sites.Site, error)
	Update(ctx context.Context, site sites.Site) (sites.Site, error)
	Delete(ctx context.Context, id string) error
}

type siteBanService interface {
	Ban(ctx context.Context, siteID string, address string) (accesspolicies.AccessPolicy, error)
	Unban(ctx context.Context, siteID string, address string) (accesspolicies.AccessPolicy, error)
}

type SitesHandler struct {
	sites siteService
	bans  siteBanService
}

type siteBanRequest struct {
	IP string `json:"ip"`
}

func NewSitesHandler(sites siteService, bans siteBanService) *SitesHandler {
	return &SitesHandler{sites: sites, bans: bans}
}

func (h *SitesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/api/sites" && r.Method == http.MethodGet:
		h.list(w, r)
	case r.URL.Path == "/api/sites" && r.Method == http.MethodPost:
		h.create(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/sites/") && r.Method == http.MethodPut:
		h.update(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/sites/") && r.Method == http.MethodDelete:
		h.delete(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/sites/") && strings.HasSuffix(r.URL.Path, "/ban") && r.Method == http.MethodPost:
		h.ban(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/sites/") && strings.HasSuffix(r.URL.Path, "/unban") && r.Method == http.MethodPost:
		h.unban(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (h *SitesHandler) list(w http.ResponseWriter, r *http.Request) {
	items, err := h.sites.List()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "site store unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *SitesHandler) create(w http.ResponseWriter, r *http.Request) {
	var site sites.Site
	if err := json.NewDecoder(r.Body).Decode(&site); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	created, err := h.sites.Create(withActorIP(r), site)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *SitesHandler) update(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/sites/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "site id is required"})
		return
	}

	var site sites.Site
	if err := json.NewDecoder(r.Body).Decode(&site); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	site.ID = id
	updated, err := h.sites.Update(withActorIP(r), site)
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

func (h *SitesHandler) delete(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/sites/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "site id is required"})
		return
	}
	if err := h.sites.Delete(withActorIP(r), id); err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *SitesHandler) ban(w http.ResponseWriter, r *http.Request) {
	siteID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/sites/"), "/ban")
	siteID = strings.TrimSuffix(siteID, "/")
	var req siteBanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	updated, err := h.bans.Ban(withActorIP(r), siteID, req.IP)
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

func (h *SitesHandler) unban(w http.ResponseWriter, r *http.Request) {
	siteID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/sites/"), "/unban")
	siteID = strings.TrimSuffix(siteID, "/")
	var req siteBanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	updated, err := h.bans.Unban(withActorIP(r), siteID, req.IP)
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
