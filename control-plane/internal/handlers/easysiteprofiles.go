package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"waf/control-plane/internal/easysiteprofiles"
)

type easySiteProfileService interface {
	List() ([]easysiteprofiles.EasySiteProfile, error)
	Get(siteID string) (easysiteprofiles.EasySiteProfile, error)
	Upsert(ctx context.Context, profile easysiteprofiles.EasySiteProfile) (easysiteprofiles.EasySiteProfile, error)
	Delete(ctx context.Context, siteID string) error
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
	case strings.HasPrefix(r.URL.Path, "/api/easy-site-profiles/") && r.Method == http.MethodDelete:
		h.delete(w, r)
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
	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	var item easysiteprofiles.EasySiteProfile
	if err := json.Unmarshal(rawBody, &item); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	applyEasyProfileTopLevelAliases(rawBody, &item)
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

func (h *EasySiteProfilesHandler) delete(w http.ResponseWriter, r *http.Request) {
	siteID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/easy-site-profiles/"), "/")
	if siteID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "site id is required"})
		return
	}
	if err := h.profiles.Delete(withActorIP(r), siteID); err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]any{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func applyEasyProfileTopLevelAliases(rawBody []byte, item *easysiteprofiles.EasySiteProfile) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rawBody, &raw); err != nil {
		return
	}
	applyRawString(raw, "security_mode", &item.FrontService.SecurityMode)
	applyRawBool(raw, "use_blacklist", &item.SecurityBehaviorAndLimits.UseBlacklist)
	applyRawStringSlice(raw, "blacklist_ip", &item.SecurityBehaviorAndLimits.BlacklistIP)
	applyRawStringSlice(raw, "blacklist_user_agent", &item.SecurityBehaviorAndLimits.BlacklistUserAgent)
	applyRawStringSlice(raw, "blacklist_uri", &item.SecurityBehaviorAndLimits.BlacklistURI)
	applyRawStringSlice(raw, "exceptions_uri", &item.SecurityBehaviorAndLimits.ExceptionsURI)
	applyRawBool(raw, "use_limit_req", &item.SecurityBehaviorAndLimits.UseLimitReq)
	applyRawString(raw, "limit_req_rate", &item.SecurityBehaviorAndLimits.LimitReqRate)
	applyRawBool(raw, "use_bad_behavior", &item.SecurityBehaviorAndLimits.UseBadBehavior)
	applyRawStringSlice(raw, "blacklist_country", &item.SecurityCountryPolicy.BlacklistCountry)
	applyRawStringSlice(raw, "whitelist_country", &item.SecurityCountryPolicy.WhitelistCountry)
	applyRawBool(raw, "show_geo_block_page", &item.SecurityCountryPolicy.ShowGeoBlockPage)
	if value, ok := raw["custom_limit_rules"]; ok {
		_ = json.Unmarshal(value, &item.SecurityBehaviorAndLimits.CustomLimitRules)
	}
	if value, ok := raw["auth_session_ttl_min"]; ok {
		_ = json.Unmarshal(value, &item.SecurityAuthBasic.SessionInactivityMinutes)
	}
	if value, ok := raw["virtual_patches"]; ok {
		_ = json.Unmarshal(value, &item.VirtualPatches)
	}
	if value, ok := raw["modsecurity_exclusion_rules"]; ok {
		_ = json.Unmarshal(value, &item.SecurityModSecurity.ExclusionRules)
	}
}

func applyRawString(raw map[string]json.RawMessage, key string, target *string) {
	if value, ok := raw[key]; ok {
		_ = json.Unmarshal(value, target)
	}
}

func applyRawBool(raw map[string]json.RawMessage, key string, target *bool) {
	if value, ok := raw[key]; ok {
		_ = json.Unmarshal(value, target)
	}
}

func applyRawStringSlice(raw map[string]json.RawMessage, key string, target *[]string) {
	if value, ok := raw[key]; ok {
		_ = json.Unmarshal(value, target)
	}
}
