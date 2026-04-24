package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"waf/control-plane/internal/easysiteprofiles"
)

type easySiteProfileService interface {
	List() ([]easysiteprofiles.EasySiteProfile, error)
	Get(siteID string) (easysiteprofiles.EasySiteProfile, error)
	Upsert(ctx context.Context, profile easysiteprofiles.EasySiteProfile) (easysiteprofiles.EasySiteProfile, error)
}

type EasySiteProfilesHandler struct {
	profiles easySiteProfileService
}

func NewEasySiteProfilesHandler(profiles easySiteProfileService) *EasySiteProfilesHandler {
	return &EasySiteProfilesHandler{profiles: profiles}
}

func (h *EasySiteProfilesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/api/easy-site-profiles" && r.Method == http.MethodGet:
		h.list(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/easy-site-profiles/") && r.Method == http.MethodGet:
		h.get(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/easy-site-profiles/") && r.Method == http.MethodPut:
		h.upsert(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/easy-site-profiles/") && r.Method == http.MethodPost:
		h.upsert(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (h *EasySiteProfilesHandler) list(w http.ResponseWriter, _ *http.Request) {
	items, err := h.profiles.List()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *EasySiteProfilesHandler) get(w http.ResponseWriter, r *http.Request) {
	siteID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/easy-site-profiles/"), "/")
	if siteID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "site id is required"})
		return
	}
	item, err := h.profiles.Get(siteID)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *EasySiteProfilesHandler) upsert(w http.ResponseWriter, r *http.Request) {
	siteID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/easy-site-profiles/"), "/")
	if siteID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "site id is required"})
		return
	}
	var item easysiteprofiles.EasySiteProfile
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	item.SiteID = siteID
	updated, err := h.profiles.Upsert(withActorIP(r), item)
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
