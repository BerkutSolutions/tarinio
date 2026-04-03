package handlers

import (
	"encoding/json"
	"io"
	"net/http"
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

func (h *SettingsRuntimeHandler) checkUpdates() {
	runtimeSettingsState.mu.Lock()
	defer runtimeSettingsState.mu.Unlock()
	runtimeSettingsState.lastCheckedAt = time.Now().UTC().Format(time.RFC3339)
	runtimeSettingsState.latestVersion = appmeta.AppVersion
	runtimeSettingsState.hasUpdate = false
	runtimeSettingsState.releaseURL = ""
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
		h.checkUpdates()
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
