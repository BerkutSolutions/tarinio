package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"waf/control-plane/internal/appmeta"
)

type SettingsRuntimeHandler struct{}

var runtimeSettingsState = struct {
	mu                  sync.RWMutex
	updateChecksEnabled bool
	lastCheckedAt       string
	latestVersion       string
	releaseURL          string
	hasUpdate           bool
}{
	updateChecksEnabled: true,
	lastCheckedAt:       "",
	latestVersion:       appmeta.AppVersion,
	releaseURL:          "",
	hasUpdate:           false,
}

func NewSettingsRuntimeHandler() *SettingsRuntimeHandler {
	return &SettingsRuntimeHandler{}
}

func (h *SettingsRuntimeHandler) responsePayload() map[string]any {
	runtimeSettingsState.mu.RLock()
	defer runtimeSettingsState.mu.RUnlock()

	return map[string]any{
		"deployment_mode":       "standalone",
		"app_version":           appmeta.AppVersion,
		"update_checks_enabled": runtimeSettingsState.updateChecksEnabled,
		"update": map[string]any{
			"has_update":     runtimeSettingsState.hasUpdate,
			"latest_version": runtimeSettingsState.latestVersion,
			"checked_at":     runtimeSettingsState.lastCheckedAt,
			"release_url":    runtimeSettingsState.releaseURL,
		},
	}
}

func (h *SettingsRuntimeHandler) checkUpdates(manual bool) {
	if !manual {
		runtimeSettingsState.mu.RLock()
		enabled := runtimeSettingsState.updateChecksEnabled
		lastCheckedAt := runtimeSettingsState.lastCheckedAt
		runtimeSettingsState.mu.RUnlock()
		if !enabled {
			return
		}
		if parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(lastCheckedAt)); err == nil {
			if time.Since(parsed.UTC()) < time.Hour {
				return
			}
		}
	}

	currentVersion := strings.TrimSpace(appmeta.AppVersion)
	releaseURL := strings.TrimSpace(appmeta.RepositoryURL)
	if releaseURL != "" {
		releaseURL = strings.TrimRight(releaseURL, "/") + "/releases"
	}
	latestVersion := currentVersion
	hasUpdate := false
	checkedAt := time.Now().UTC().Format(time.RFC3339)

	if endpoint := strings.TrimSpace(appmeta.GitHubAPIReleases); endpoint != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err == nil {
			req.Header.Set("Accept", "application/vnd.github+json")
			req.Header.Set("User-Agent", "tarinio-control-plane")
			if resp, doErr := http.DefaultClient.Do(req); doErr == nil {
				defer resp.Body.Close()
				if resp.StatusCode >= 200 && resp.StatusCode < 300 {
					var payload struct {
						TagName string `json:"tag_name"`
						Name    string `json:"name"`
						HTMLURL string `json:"html_url"`
					}
					if decodeErr := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload); decodeErr == nil {
						candidateVersion := normalizeVersion(payload.TagName)
						if candidateVersion == "" {
							candidateVersion = normalizeVersion(payload.Name)
						}
						if candidateVersion != "" {
							latestVersion = candidateVersion
							hasUpdate = isVersionGreater(candidateVersion, currentVersion)
						}
						if link := strings.TrimSpace(payload.HTMLURL); link != "" {
							releaseURL = link
						}
					}
				}
			}
		}
	}

	runtimeSettingsState.mu.Lock()
	defer runtimeSettingsState.mu.Unlock()
	runtimeSettingsState.lastCheckedAt = checkedAt
	runtimeSettingsState.latestVersion = latestVersion
	runtimeSettingsState.hasUpdate = hasUpdate
	runtimeSettingsState.releaseURL = releaseURL
}

func normalizeVersion(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "v")
	value = strings.TrimPrefix(value, "V")
	return value
}

func isVersionGreater(latest string, current string) bool {
	lParts := parseVersionParts(normalizeVersion(latest))
	cParts := parseVersionParts(normalizeVersion(current))
	maxLen := len(lParts)
	if len(cParts) > maxLen {
		maxLen = len(cParts)
	}
	for i := 0; i < maxLen; i++ {
		lv := 0
		cv := 0
		if i < len(lParts) {
			lv = lParts[i]
		}
		if i < len(cParts) {
			cv = cParts[i]
		}
		if lv > cv {
			return true
		}
		if lv < cv {
			return false
		}
	}
	return false
}

func parseVersionParts(value string) []int {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	raw := strings.Split(trimmed, ".")
	parts := make([]int, 0, len(raw))
	for _, item := range raw {
		item = strings.TrimSpace(item)
		if item == "" {
			parts = append(parts, 0)
			continue
		}
		digits := strings.Builder{}
		for _, r := range item {
			if r < '0' || r > '9' {
				break
			}
			digits.WriteRune(r)
		}
		if digits.Len() == 0 {
			parts = append(parts, 0)
			continue
		}
		parsed, err := strconv.Atoi(digits.String())
		if err != nil {
			parts = append(parts, 0)
			continue
		}
		parts = append(parts, parsed)
	}
	return parts
}

func (h *SettingsRuntimeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, h.responsePayload())
	case http.MethodPut:
		if r.URL.Path != "/api/settings/runtime" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		body, ok := readJSONBody(w, r)
		if !ok {
			return
		}
		updateChecksEnabled := false
		if raw, exists := body["update_checks_enabled"]; exists {
			flag, typeOK := raw.(bool)
			if !typeOK {
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": "update_checks_enabled must be boolean"})
				return
			}
			updateChecksEnabled = flag
		} else {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "update_checks_enabled is required"})
			return
		}
		runtimeSettingsState.mu.Lock()
		runtimeSettingsState.updateChecksEnabled = updateChecksEnabled
		runtimeSettingsState.mu.Unlock()
		writeJSON(w, http.StatusOK, h.responsePayload())
	case http.MethodPost:
		if r.URL.Path != "/api/settings/runtime/check-updates" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		body, ok := readJSONBody(w, r)
		if !ok {
			return
		}
		manual := true
		if raw, exists := body["manual"]; exists {
			flag, typeOK := raw.(bool)
			if !typeOK {
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": "manual must be boolean"})
				return
			}
			manual = flag
		}
		h.checkUpdates(manual)
		writeJSON(w, http.StatusOK, h.responsePayload())
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func readJSONBody(w http.ResponseWriter, r *http.Request) (map[string]any, bool) {
	limited := io.LimitReader(r.Body, 1<<20)
	defer r.Body.Close()

	decoder := json.NewDecoder(limited)
	decoder.DisallowUnknownFields()

	var body map[string]any
	if err := decoder.Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON body"})
		return nil, false
	}
	if body == nil {
		body = map[string]any{}
	}
	return body, true
}
