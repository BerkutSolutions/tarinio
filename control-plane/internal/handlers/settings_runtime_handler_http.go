package handlers

import (
	"net/http"
	"net/url"
	"strings"

	"waf/control-plane/internal/auth"
	"waf/control-plane/internal/rbac"
	"waf/internal/loggingconfig"
)

func (h *SettingsRuntimeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if r.URL.Path == "/api/settings/runtime/storage-indexes" {
			values := cloneIndexQuery(r.URL.Query())
			writeJSON(w, http.StatusOK, runtimeIndexesFromQuery(values))
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
		_, changesStorageRetention := body["storage"]
		_, changesLogging := body["logging"]
		if (changesStorageRetention || changesLogging) && !canWriteStorageRetention(r) {
			writeJSON(w, http.StatusForbidden, map[string]any{"error": "settings.storage.write permission required for storage retention"})
			return
		}

		runtimeSettingsState.mu.Lock()
		nextUpdateChecksEnabled := runtimeSettingsState.updateChecksEnabled
		nextLanguage := runtimeSettingsState.language
		nextLoginAppearance := runtimeSettingsState.loginAppearance
		nextHealthcheckAppearance := runtimeSettingsState.healthcheckAppearance
		nextLogging := runtimeSettingsState.logging
		nextSecurity := runtimeSettingsState.security
		nextStorage := runtimeSettingsState.storage

		updated := false
		if raw, exists := body["update_checks_enabled"]; exists {
			flag, typeOK := raw.(bool)
			if !typeOK {
				runtimeSettingsState.mu.Unlock()
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": "update_checks_enabled must be boolean"})
				return
			}
			nextUpdateChecksEnabled = flag
			updated = true
		}
		if raw, exists := body["language"]; exists {
			value, typeOK := raw.(string)
			if !typeOK {
				runtimeSettingsState.mu.Unlock()
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": "language must be string"})
				return
			}
			nextLanguage = normalizeRuntimeLanguage(value)
			updated = true
		}
		if raw, exists := body["login_appearance"]; exists {
			value, typeOK := raw.(string)
			if !typeOK {
				runtimeSettingsState.mu.Unlock()
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": "login_appearance must be string"})
				return
			}
			normalized := normalizeLoginAppearance(value)
			if normalized != strings.TrimSpace(value) {
				runtimeSettingsState.mu.Unlock()
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": "unknown login_appearance"})
				return
			}
			nextLoginAppearance = normalized
			updated = true
		}
		if raw, exists := body["healthcheck_appearance"]; exists {
			value, typeOK := raw.(string)
			if !typeOK {
				runtimeSettingsState.mu.Unlock()
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": "healthcheck_appearance must be string"})
				return
			}
			normalized := normalizeHealthcheckAppearance(value)
			if normalized != strings.TrimSpace(value) {
				runtimeSettingsState.mu.Unlock()
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": "unknown healthcheck_appearance"})
				return
			}
			nextHealthcheckAppearance = normalized
			updated = true
		}
		if raw, exists := body["logging"]; exists {
			typed, typeOK := raw.(map[string]any)
			if !typeOK {
				runtimeSettingsState.mu.Unlock()
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": "logging must be object"})
				return
			}
			next, err := parseLoggingSettings(typed, nextLogging, runtimeSettingsState.pepper, nextSecurity.AllowInsecureVaultTLS)
			if err != nil {
				runtimeSettingsState.mu.Unlock()
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
				return
			}
			nextLogging = next
			updated = true
		}
		if raw, exists := body["security"]; exists {
			typed, typeOK := raw.(map[string]any)
			if !typeOK {
				runtimeSettingsState.mu.Unlock()
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": "security must be object"})
				return
			}
			next, err := parseRuntimeSecuritySettings(typed, nextSecurity)
			if err != nil {
				runtimeSettingsState.mu.Unlock()
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
				return
			}
			nextSecurity = next
			if !nextSecurity.AllowInsecureVaultTLS {
				nextLogging.Vault.TLSSkipVerify = false
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
			next, err := parseStorageRetention(typed, nextStorage)
			if err != nil {
				runtimeSettingsState.mu.Unlock()
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
				return
			}
			nextStorage = next
			nextLogging.Retention.HotDays = next.HotIndexDays
			nextLogging.Retention.ColdDays = next.ColdIndexDays
			nextLogging = loggingconfig.Normalize(nextLogging)
			updated = true
		}
		if !updated {
			runtimeSettingsState.mu.Unlock()
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "at least one field must be provided"})
			return
		}
		runtimeSettingsState.updateChecksEnabled = nextUpdateChecksEnabled
		runtimeSettingsState.language = nextLanguage
		runtimeSettingsState.loginAppearance = nextLoginAppearance
		runtimeSettingsState.healthcheckAppearance = nextHealthcheckAppearance
		runtimeSettingsState.logging = nextLogging
		runtimeSettingsState.security = nextSecurity
		runtimeSettingsState.storage = nextStorage
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

func cloneIndexQuery(input url.Values) url.Values {
	values := make(url.Values, len(input))
	for key, items := range input {
		values[key] = append([]string(nil), items...)
	}
	if strings.TrimSpace(values.Get("storage_indexes_limit")) == "" {
		values.Set("storage_indexes_limit", values.Get("limit"))
	}
	if strings.TrimSpace(values.Get("storage_indexes_limit")) == "" {
		values.Set("storage_indexes_limit", "10")
	}
	if strings.TrimSpace(values.Get("storage_indexes_offset")) == "" {
		values.Set("storage_indexes_offset", values.Get("offset"))
	}
	if strings.TrimSpace(values.Get("storage_indexes_offset")) == "" {
		values.Set("storage_indexes_offset", "0")
	}
	return values
}

func canWriteStorageRetention(r *http.Request) bool {
	session, ok := auth.SessionFromContext(r.Context())
	if !ok {
		return false
	}
	for _, permission := range session.Permissions {
		if rbac.Permission(permission) == rbac.PermissionSettingsStorageWrite {
			return true
		}
	}
	return false
}
