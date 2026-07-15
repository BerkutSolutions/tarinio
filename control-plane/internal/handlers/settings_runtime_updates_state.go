package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"waf/control-plane/internal/appmeta"
	"waf/control-plane/internal/storage"
	"waf/internal/loggingconfig"
)

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
	runtimeSettingsState.lastCheckedAt = checkedAt
	runtimeSettingsState.latestVersion = latestVersion
	runtimeSettingsState.hasUpdate = hasUpdate
	runtimeSettingsState.releaseURL = releaseURL
	savePersistedRuntimeSettingsLocked()
	runtimeSettingsState.mu.Unlock()
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

func loadPersistedRuntimeSettingsLocked() {
	path := strings.TrimSpace(runtimeSettingsState.statePath)
	if path == "" {
		return
	}
	var (
		content []byte
		err     error
	)
	if !storage.IsNilBackend(runtimeSettingsState.backend) {
		content, err = storage.NewBackendJSONState(runtimeSettingsState.backend, "settings/runtime_settings.json", path).Load()
		if err != nil {
			return
		}
	} else {
		content, err = os.ReadFile(path)
		if err != nil {
			return
		}
	}
	var stored persistedRuntimeSettings
	if err := json.Unmarshal(content, &stored); err != nil {
		return
	}
	runtimeSettingsState.updateChecksEnabled = stored.UpdateChecksEnabled
	runtimeSettingsState.language = normalizeRuntimeLanguage(stored.Language)
	runtimeSettingsState.lastCheckedAt = strings.TrimSpace(stored.LastCheckedAt)
	runtimeSettingsState.latestVersion = strings.TrimSpace(stored.LatestVersion)
	runtimeSettingsState.releaseURL = strings.TrimSpace(stored.ReleaseURL)
	runtimeSettingsState.hasUpdate = stored.HasUpdate
	runtimeSettingsState.storage = normalizeStorageRetention(stored.Storage)
	runtimeSettingsState.security = normalizeRuntimeSecuritySettings(stored.Security)
	runtimeSettingsState.loginAppearance = normalizeLoginAppearance(stored.LoginAppearance)
	runtimeSettingsState.healthcheckAppearance = normalizeHealthcheckAppearance(stored.HealthcheckAppearance)
	runtimeSettingsState.logging = loggingconfig.Normalize(stored.Logging)
	if !runtimeSettingsState.security.AllowInsecureVaultTLS {
		runtimeSettingsState.logging.Vault.TLSSkipVerify = false
	}
	if runtimeSettingsState.storage.HotIndexDays <= 0 {
		runtimeSettingsState.storage.HotIndexDays = runtimeSettingsState.logging.Retention.HotDays
	}
	if runtimeSettingsState.storage.ColdIndexDays <= 0 {
		runtimeSettingsState.storage.ColdIndexDays = runtimeSettingsState.logging.Retention.ColdDays
	}
	normalizePersistedUpdateStateLocked()
}

func savePersistedRuntimeSettingsLocked() {
	path := strings.TrimSpace(runtimeSettingsState.statePath)
	if path == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	payload := persistedRuntimeSettings{
		UpdateChecksEnabled:   runtimeSettingsState.updateChecksEnabled,
		Language:              normalizeRuntimeLanguage(runtimeSettingsState.language),
		LastCheckedAt:         runtimeSettingsState.lastCheckedAt,
		LatestVersion:         runtimeSettingsState.latestVersion,
		ReleaseURL:            runtimeSettingsState.releaseURL,
		HasUpdate:             runtimeSettingsState.hasUpdate,
		Storage:               normalizeStorageRetention(runtimeSettingsState.storage),
		Security:              normalizeRuntimeSecuritySettings(runtimeSettingsState.security),
		LoginAppearance:       normalizeLoginAppearance(runtimeSettingsState.loginAppearance),
		HealthcheckAppearance: normalizeHealthcheckAppearance(runtimeSettingsState.healthcheckAppearance),
		Logging:               loggingconfig.Normalize(runtimeSettingsState.logging),
	}
	content, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return
	}
	content = append(content, '\n')
	_ = os.WriteFile(path, content, 0o644)
	if !storage.IsNilBackend(runtimeSettingsState.backend) {
		_ = storage.NewBackendJSONState(runtimeSettingsState.backend, "settings/runtime_settings.json", path).Save(content)
		return
	}
}

func normalizePersistedUpdateStateLocked() {
	currentVersion := strings.TrimSpace(appmeta.AppVersion)
	latestVersion := strings.TrimSpace(runtimeSettingsState.latestVersion)
	if latestVersion == "" || !isVersionGreater(latestVersion, currentVersion) {
		runtimeSettingsState.latestVersion = currentVersion
		runtimeSettingsState.hasUpdate = false
		return
	}
	runtimeSettingsState.latestVersion = latestVersion
	runtimeSettingsState.hasUpdate = true
}
