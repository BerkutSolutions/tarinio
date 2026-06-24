package handlers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"waf/control-plane/internal/storage"
	"waf/internal/loggingconfig"
)

func NewSettingsRuntimeHandler(settingsRoot string, runtimeHealthURL string, pepper string) *SettingsRuntimeHandler {
	return NewSettingsRuntimeHandlerWithBackend(settingsRoot, runtimeHealthURL, nil, pepper)
}

func NewSettingsRuntimeHandlerWithBackend(settingsRoot string, runtimeHealthURL string, backend storage.Backend, pepper string) *SettingsRuntimeHandler {
	runtimeSettingsState.mu.Lock()
	defer runtimeSettingsState.mu.Unlock()
	runtimeSettingsState.pepper = strings.TrimSpace(pepper)
	if !runtimeSettingsState.initialized {
		runtimeSettingsState.security = normalizeRuntimeSecuritySettings(runtimeSettingsState.security)
		runtimeSettingsState.logging = defaultLoggingSettingsFromEnv(runtimeSettingsState.logging, runtimeSettingsState.security)
	}

	if strings.TrimSpace(runtimeSettingsState.statePath) == "" {
		root := strings.TrimSpace(settingsRoot)
		if root != "" {
			runtimeSettingsState.statePath = filepath.Join(root, "runtime_settings.json")
		}
	}
	if !storage.IsNilBackend(backend) && strings.TrimSpace(runtimeSettingsState.statePath) != "" {
		runtimeSettingsState.backend = backend
		content, err := storage.NewBackendJSONState(backend, "settings/runtime_settings.json", runtimeSettingsState.statePath).Load()
		if err == nil {
			var stored persistedRuntimeSettings
			if jsonErr := json.Unmarshal(content, &stored); jsonErr == nil {
				runtimeSettingsState.updateChecksEnabled = stored.UpdateChecksEnabled
				runtimeSettingsState.language = normalizeRuntimeLanguage(stored.Language)
				runtimeSettingsState.lastCheckedAt = stored.LastCheckedAt
				runtimeSettingsState.latestVersion = stored.LatestVersion
				runtimeSettingsState.releaseURL = stored.ReleaseURL
				runtimeSettingsState.hasUpdate = stored.HasUpdate
				runtimeSettingsState.storage = normalizeStorageRetention(stored.Storage)
				runtimeSettingsState.security = normalizeRuntimeSecuritySettings(stored.Security)
				runtimeSettingsState.logging = loggingconfig.Normalize(stored.Logging)
				normalizePersistedUpdateStateLocked()
				runtimeSettingsState.initialized = true
			}
		}
	}
	if !runtimeSettingsState.initialized {
		loadPersistedRuntimeSettingsLocked()
		runtimeSettingsState.initialized = true
	}
	runtimeSettingsState.security = normalizeRuntimeSecuritySettings(runtimeSettingsState.security)
	runtimeSettingsState.logging = reconcileLoggingSettingsFromEnv(runtimeSettingsState.logging, runtimeSettingsState.security)
	savePersistedRuntimeSettingsLocked()
	if runtimeRequestIndexes != nil {
		runtimeRequestIndexes.url = deriveRuntimeIndexesURL(runtimeHealthURL)
		runtimeRequestIndexes.token = strings.TrimSpace(os.Getenv("WAF_RUNTIME_API_TOKEN"))
	}
	return &SettingsRuntimeHandler{}
}

func CurrentStorageRetention() StorageRetention {
	runtimeSettingsState.mu.RLock()
	defer runtimeSettingsState.mu.RUnlock()
	return runtimeSettingsState.storage
}

func CurrentRuntimeSecuritySettings() RuntimeSecuritySettings {
	runtimeSettingsState.mu.RLock()
	defer runtimeSettingsState.mu.RUnlock()
	return normalizeRuntimeSecuritySettings(runtimeSettingsState.security)
}

func CurrentRuntimeLanguage() string {
	runtimeSettingsState.mu.RLock()
	defer runtimeSettingsState.mu.RUnlock()
	return normalizeRuntimeLanguage(runtimeSettingsState.language)
}
