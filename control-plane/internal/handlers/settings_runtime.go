package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"waf/control-plane/internal/appmeta"
)

type SettingsRuntimeHandler struct{}

type runtimeIndexFetcher struct {
	url    string
	client *http.Client
}

type StorageRetention struct {
	LogsDays     int `json:"logs_days"`
	ActivityDays int `json:"activity_days"`
	EventsDays   int `json:"events_days"`
	BansDays     int `json:"bans_days"`
}

type persistedRuntimeSettings struct {
	UpdateChecksEnabled bool             `json:"update_checks_enabled"`
	LastCheckedAt       string           `json:"last_checked_at,omitempty"`
	LatestVersion       string           `json:"latest_version,omitempty"`
	ReleaseURL          string           `json:"release_url,omitempty"`
	HasUpdate           bool             `json:"has_update"`
	Storage             StorageRetention `json:"storage"`
}

var runtimeSettingsState = struct {
	mu                  sync.RWMutex
	statePath           string
	initialized         bool
	updateChecksEnabled bool
	lastCheckedAt       string
	latestVersion       string
	releaseURL          string
	hasUpdate           bool
	storage             StorageRetention
}{
	updateChecksEnabled: true,
	lastCheckedAt:       "",
	latestVersion:       appmeta.AppVersion,
	releaseURL:          "",
	hasUpdate:           false,
	storage: StorageRetention{
		LogsDays:     14,
		ActivityDays: 30,
		EventsDays:   30,
		BansDays:     30,
	},
}

var runtimeRequestIndexes = &runtimeIndexFetcher{
	url:    "",
	client: &http.Client{Timeout: 3 * time.Second},
}

func NewSettingsRuntimeHandler(settingsRoot string, runtimeHealthURL string) *SettingsRuntimeHandler {
	runtimeSettingsState.mu.Lock()
	defer runtimeSettingsState.mu.Unlock()

	if strings.TrimSpace(runtimeSettingsState.statePath) == "" {
		root := strings.TrimSpace(settingsRoot)
		if root != "" {
			runtimeSettingsState.statePath = filepath.Join(root, "runtime_settings.json")
		}
	}
	if !runtimeSettingsState.initialized {
		loadPersistedRuntimeSettingsLocked()
		runtimeSettingsState.initialized = true
	}
	if runtimeRequestIndexes != nil {
		runtimeRequestIndexes.url = deriveRuntimeIndexesURL(runtimeHealthURL)
	}
	return &SettingsRuntimeHandler{}
}

func CurrentStorageRetention() StorageRetention {
	runtimeSettingsState.mu.RLock()
	defer runtimeSettingsState.mu.RUnlock()
	return runtimeSettingsState.storage
}

func (h *SettingsRuntimeHandler) responsePayload() map[string]any {
	runtimeSettingsState.mu.RLock()
	defer runtimeSettingsState.mu.RUnlock()
	return responsePayloadLocked(nil)
}

func responsePayloadLocked(indexes map[string]any) map[string]any {
	payload := map[string]any{
		"deployment_mode":       "standalone",
		"app_version":           appmeta.AppVersion,
		"update_checks_enabled": runtimeSettingsState.updateChecksEnabled,
		"storage":               runtimeSettingsState.storage,
		"update": map[string]any{
			"has_update":     runtimeSettingsState.hasUpdate,
			"latest_version": runtimeSettingsState.latestVersion,
			"checked_at":     runtimeSettingsState.lastCheckedAt,
			"release_url":    runtimeSettingsState.releaseURL,
		},
	}
	if indexes != nil {
		payload["storage_indexes"] = indexes
	}
	return payload
}

func runtimeIndexesFromQuery(values url.Values) map[string]any {
	if runtimeRequestIndexes == nil {
		return nil
	}
	if strings.TrimSpace(values.Get("storage_indexes_limit")) == "" && strings.TrimSpace(values.Get("storage_indexes_offset")) == "" {
		return nil
	}
	limit := 10
	offset := 0
	if raw := strings.TrimSpace(values.Get("storage_indexes_limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if raw := strings.TrimSpace(values.Get("storage_indexes_offset")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 0 {
			offset = parsed
		}
	}
	indexes, err := runtimeRequestIndexes.Fetch(limit, offset)
	if err != nil {
		return map[string]any{
			"items":  []map[string]any{},
			"total":  0,
			"limit":  limit,
			"offset": offset,
			"error":  err.Error(),
		}
	}
	return indexes
}

func deriveRuntimeIndexesURL(healthURL string) string {
	raw := strings.TrimSpace(healthURL)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	parsed.Path = "/requests/indexes"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func (f *runtimeIndexFetcher) Fetch(limit int, offset int) (map[string]any, error) {
	if f == nil || strings.TrimSpace(f.url) == "" {
		return map[string]any{
			"items":  []map[string]any{},
			"total":  0,
			"limit":  limit,
			"offset": offset,
		}, nil
	}
	target, err := url.Parse(f.url)
	if err != nil {
		return nil, err
	}
	q := target.Query()
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	target.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, err
	}
	client := f.client
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("runtime indexes endpoint returned %d", resp.StatusCode)
	}
	var payload map[string]any
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (f *runtimeIndexFetcher) Delete(date string) error {
	if f == nil || strings.TrimSpace(f.url) == "" {
		return nil
	}
	day := strings.TrimSpace(date)
	if day == "" {
		return fmt.Errorf("date is required")
	}
	if _, err := time.Parse("2006-01-02", day); err != nil {
		return fmt.Errorf("date must be in YYYY-MM-DD format")
	}

	target, err := url.Parse(f.url)
	if err != nil {
		return err
	}
	q := target.Query()
	q.Set("date", day)
	target.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodDelete, target.String(), nil)
	if err != nil {
		return err
	}
	client := f.client
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("runtime indexes delete endpoint returned %d", resp.StatusCode)
	}
	return nil
}

func responsePayloadWithoutIndexesLocked() map[string]any {
	return map[string]any{
		"deployment_mode":       "standalone",
		"app_version":           appmeta.AppVersion,
		"update_checks_enabled": runtimeSettingsState.updateChecksEnabled,
		"storage":               runtimeSettingsState.storage,
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

func (h *SettingsRuntimeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
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
		if runtimeRequestIndexes == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "runtime indexes fetcher unavailable"})
			return
		}
		day := strings.TrimSpace(r.URL.Query().Get("date"))
		if err := runtimeRequestIndexes.Delete(day); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func parseStorageRetention(raw map[string]any, current StorageRetention) (StorageRetention, error) {
	out := current
	if value, ok := raw["logs_days"]; ok {
		parsed, err := parsePositiveRetentionInt(value)
		if err != nil {
			return StorageRetention{}, err
		}
		out.LogsDays = parsed
	}
	if value, ok := raw["activity_days"]; ok {
		parsed, err := parsePositiveRetentionInt(value)
		if err != nil {
			return StorageRetention{}, err
		}
		out.ActivityDays = parsed
	}
	if value, ok := raw["events_days"]; ok {
		parsed, err := parsePositiveRetentionInt(value)
		if err != nil {
			return StorageRetention{}, err
		}
		out.EventsDays = parsed
	}
	if value, ok := raw["bans_days"]; ok {
		parsed, err := parsePositiveRetentionInt(value)
		if err != nil {
			return StorageRetention{}, err
		}
		out.BansDays = parsed
	}
	return normalizeStorageRetention(out), nil
}

func parsePositiveRetentionInt(value any) (int, error) {
	switch typed := value.(type) {
	case float64:
		parsed := int(typed)
		if parsed <= 0 {
			return 0, strconv.ErrSyntax
		}
		return parsed, nil
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err != nil || parsed <= 0 {
			return 0, strconv.ErrSyntax
		}
		return parsed, nil
	default:
		return 0, strconv.ErrSyntax
	}
}

func normalizeStorageRetention(input StorageRetention) StorageRetention {
	if input.LogsDays <= 0 {
		input.LogsDays = 14
	}
	if input.ActivityDays <= 0 {
		input.ActivityDays = 30
	}
	if input.EventsDays <= 0 {
		input.EventsDays = 30
	}
	if input.BansDays <= 0 {
		input.BansDays = 30
	}
	return input
}

func loadPersistedRuntimeSettingsLocked() {
	path := strings.TrimSpace(runtimeSettingsState.statePath)
	if path == "" {
		return
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var stored persistedRuntimeSettings
	if err := json.Unmarshal(content, &stored); err != nil {
		return
	}
	runtimeSettingsState.updateChecksEnabled = stored.UpdateChecksEnabled
	runtimeSettingsState.lastCheckedAt = strings.TrimSpace(stored.LastCheckedAt)
	runtimeSettingsState.latestVersion = strings.TrimSpace(stored.LatestVersion)
	runtimeSettingsState.releaseURL = strings.TrimSpace(stored.ReleaseURL)
	runtimeSettingsState.hasUpdate = stored.HasUpdate
	runtimeSettingsState.storage = normalizeStorageRetention(stored.Storage)
	if runtimeSettingsState.latestVersion == "" {
		runtimeSettingsState.latestVersion = appmeta.AppVersion
	}
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
		UpdateChecksEnabled: runtimeSettingsState.updateChecksEnabled,
		LastCheckedAt:       runtimeSettingsState.lastCheckedAt,
		LatestVersion:       runtimeSettingsState.latestVersion,
		ReleaseURL:          runtimeSettingsState.releaseURL,
		HasUpdate:           runtimeSettingsState.hasUpdate,
		Storage:             normalizeStorageRetention(runtimeSettingsState.storage),
	}
	content, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return
	}
	content = append(content, '\n')
	_ = os.WriteFile(path, content, 0o644)
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
