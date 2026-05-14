package handlers

import (
	"net/http"
	"strings"

	"waf/internal/loggingconfig"
)

func (h *SettingsRuntimeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if r.URL.Path == "/api/settings/runtime/storage-indexes" {
			writeJSON(w, http.StatusOK, runtimeIndexesFromQuery(r.URL.Query()))
			return
		}
		indexes := runtimeIndexesFromQuery(r.URL.Query())
		runtimeSettingsState.mu.RLock()
		payload := responsePayloadLocked(indexes)
		runtimeSettingsState.mu.RUnlock()
		writeJSON(w, http.StatusOK, payload)
	case http.MethodPut:
		if r.URL.Path != "/api/settings/runtime" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		body, ok := readJSONBody(w, r)
		if !ok {
			return
		}

		runtimeSettingsState.mu.Lock()

		updated := false
		if raw, exists := body["update_checks_enabled"]; exists {
			flag, typeOK := raw.(bool)
			if !typeOK {
				runtimeSettingsState.mu.Unlock()
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": "update_checks_enabled must be boolean"})
				return
			}
			runtimeSettingsState.updateChecksEnabled = flag
			updated = true
		}
		if raw, exists := body["language"]; exists {
			value, typeOK := raw.(string)
			if !typeOK {
				runtimeSettingsState.mu.Unlock()
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": "language must be string"})
				return
			}
			runtimeSettingsState.language = normalizeRuntimeLanguage(value)
			updated = true
		}
		if raw, exists := body["logging"]; exists {
			typed, typeOK := raw.(map[string]any)
			if !typeOK {
				runtimeSettingsState.mu.Unlock()
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": "logging must be object"})
				return
			}
			next, err := parseLoggingSettings(typed, runtimeSettingsState.logging, runtimeSettingsState.pepper, runtimeSettingsState.security.AllowInsecureVaultTLS)
			if err != nil {
				runtimeSettingsState.mu.Unlock()
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
				return
			}
			runtimeSettingsState.logging = next
			updated = true
		}
		if raw, exists := body["security"]; exists {
			typed, typeOK := raw.(map[string]any)
			if !typeOK {
				runtimeSettingsState.mu.Unlock()
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": "security must be object"})
				return
			}
			next, err := parseRuntimeSecuritySettings(typed, runtimeSettingsState.security)
			if err != nil {
				runtimeSettingsState.mu.Unlock()
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
				return
			}
			runtimeSettingsState.security = next
			if !runtimeSettingsState.security.AllowInsecureVaultTLS {
				runtimeSettingsState.logging.Vault.TLSSkipVerify = false
			}
			updated = true
		}
		if raw, exists := body["storage"]; exists {
			typed, typeOK := raw.(map[string]any)
			if !typeOK {
				runtimeSettingsState.mu.Unlock()
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": "storage must be object"})
				return
			}
			next, err := parseStorageRetention(typed, runtimeSettingsState.storage)
			if err != nil {
				runtimeSettingsState.mu.Unlock()
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
				return
			}
			runtimeSettingsState.storage = next
			runtimeSettingsState.logging.Retention.HotDays = next.HotIndexDays
			runtimeSettingsState.logging.Retention.ColdDays = next.ColdIndexDays
			runtimeSettingsState.logging = loggingconfig.Normalize(runtimeSettingsState.logging)
			updated = true
		}
		if !updated {
			runtimeSettingsState.mu.Unlock()
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "at least one field must be provided"})
			return
		}
		savePersistedRuntimeSettingsLocked()
		payload := responsePayloadWithoutIndexesLocked()
		runtimeSettingsState.mu.Unlock()
		writeJSON(w, http.StatusOK, payload)
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
	case http.MethodDelete:
		if r.URL.Path != "/api/settings/runtime/storage-indexes" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		stream := normalizeStorageIndexStream(r.URL.Query().Get("stream"))
		day := strings.TrimSpace(r.URL.Query().Get("date"))
		if err := deleteStorageIndexes(stream, day); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
