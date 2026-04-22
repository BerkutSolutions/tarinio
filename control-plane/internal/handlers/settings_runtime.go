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
	"waf/control-plane/internal/storage"
	"waf/internal/loggingconfig"
	"waf/internal/secretcrypto"
	"waf/internal/vaultkv"
)

type SettingsRuntimeHandler struct{}

type runtimeIndexFetcher struct {
	url    string
	client *http.Client
}

type storageIndexItem struct {
	Stream      string `json:"stream,omitempty"`
	Date        string `json:"date"`
	FileName    string `json:"file_name"`
	SizeBytes   int64  `json:"size_bytes"`
	Lines       int    `json:"lines"`
	UpdatedAt   string `json:"updated_at"`
	StorageType string `json:"storage_type,omitempty"`
}

type StorageRetention struct {
	LogsDays      int `json:"logs_days"`
	ActivityDays  int `json:"activity_days"`
	EventsDays    int `json:"events_days"`
	BansDays      int `json:"bans_days"`
	HotIndexDays  int `json:"hot_index_days,omitempty"`
	ColdIndexDays int `json:"cold_index_days,omitempty"`
}

type persistedRuntimeSettings struct {
	UpdateChecksEnabled bool                   `json:"update_checks_enabled"`
	Language            string                 `json:"language,omitempty"`
	LastCheckedAt       string                 `json:"last_checked_at,omitempty"`
	LatestVersion       string                 `json:"latest_version,omitempty"`
	ReleaseURL          string                 `json:"release_url,omitempty"`
	HasUpdate           bool                   `json:"has_update"`
	Storage             StorageRetention       `json:"storage"`
	Logging             loggingconfig.Settings `json:"logging,omitempty"`
}

var runtimeSettingsState = struct {
	mu                  sync.RWMutex
	statePath           string
	backend             storage.Backend
	pepper              string
	initialized         bool
	updateChecksEnabled bool
	language            string
	lastCheckedAt       string
	latestVersion       string
	releaseURL          string
	hasUpdate           bool
	storage             StorageRetention
	logging             loggingconfig.Settings
}{
	updateChecksEnabled: true,
	language:            "en",
	lastCheckedAt:       "",
	latestVersion:       appmeta.AppVersion,
	releaseURL:          "",
	hasUpdate:           false,
	storage: StorageRetention{
		LogsDays:      14,
		ActivityDays:  30,
		EventsDays:    30,
		BansDays:      30,
		HotIndexDays:  loggingconfig.DefaultHotDays,
		ColdIndexDays: loggingconfig.DefaultColdDays,
	},
	logging: loggingconfig.Normalize(loggingconfig.Settings{
		Backend: loggingconfig.BackendOpenSearch,
		Hot: loggingconfig.HotSettings{
			Backend: loggingconfig.BackendOpenSearch,
		},
		Cold: loggingconfig.ColdSettings{
			Backend: loggingconfig.BackendOpenSearch,
		},
		Retention: loggingconfig.RetentionSettings{
			HotDays:  loggingconfig.DefaultHotDays,
			ColdDays: loggingconfig.DefaultColdDays,
		},
		OpenSearch: loggingconfig.OpenSearchSettings{
			IndexPrefix:   "waf-hot",
			RequestsIndex: "waf-requests",
			EventsIndex:   "waf-events",
			ActivityIndex: "waf-activity",
		},
		ClickHouse: loggingconfig.ClickHouseSettings{
			Database:         "waf_logs",
			Table:            "request_logs",
			MigrationEnabled: true,
		},
		SecretProvider: loggingconfig.SecretProviderVault,
		Vault: loggingconfig.VaultSettings{
			Enabled:    true,
			Mount:      "secret",
			PathPrefix: "tarinio",
		},
	}),
}

var runtimeRequestIndexes = &runtimeIndexFetcher{
	url:    "",
	client: &http.Client{Timeout: 3 * time.Second},
}

func defaultLoggingSettingsFromEnv(current loggingconfig.Settings) loggingconfig.Settings {
	current = loggingconfig.Normalize(current)
	clickhouseEndpoint := strings.TrimRight(strings.TrimSpace(os.Getenv("CLICKHOUSE_ENDPOINT")), "/")
	clickhouseUser := strings.TrimSpace(os.Getenv("CLICKHOUSE_USER"))
	if clickhouseUser == "" {
		clickhouseUser = "waf"
	}
	clickhousePassword := strings.TrimSpace(os.Getenv("CLICKHOUSE_PASSWORD"))
	clickhouseDatabase := strings.TrimSpace(os.Getenv("CLICKHOUSE_DB"))
	if clickhouseDatabase == "" {
		clickhouseDatabase = "waf_logs"
	}
	opensearchEndpoint := strings.TrimRight(strings.TrimSpace(os.Getenv("OPENSEARCH_ENDPOINT")), "/")
	if opensearchEndpoint == "" {
		opensearchEndpoint = "http://opensearch:9200"
	}
	opensearchUsername := strings.TrimSpace(os.Getenv("OPENSEARCH_USERNAME"))
	if opensearchUsername == "" {
		opensearchUsername = strings.TrimSpace(os.Getenv("OPENSEARCH_USER"))
	}
	opensearchPassword := strings.TrimSpace(os.Getenv("OPENSEARCH_PASSWORD"))
	opensearchAPIKey := strings.TrimSpace(os.Getenv("OPENSEARCH_API_KEY"))
	opensearchIndexPrefix := strings.TrimSpace(os.Getenv("OPENSEARCH_INDEX_PREFIX"))
	if opensearchIndexPrefix == "" {
		opensearchIndexPrefix = "waf-hot"
	}
	vaultAddr := strings.TrimRight(strings.TrimSpace(os.Getenv("VAULT_ADDR")), "/")
	vaultToken := envSecretValue("VAULT_TOKEN", "VAULT_TOKEN_FILE")
	vaultMount := strings.TrimSpace(os.Getenv("VAULT_MOUNT"))
	if vaultMount == "" {
		vaultMount = "secret"
	}
	vaultPathPrefix := strings.TrimSpace(os.Getenv("VAULT_PATH_PREFIX"))
	if vaultPathPrefix == "" {
		vaultPathPrefix = "tarinio"
	}

	if clickhousePassword != "" || strings.TrimSpace(current.ClickHouse.PasswordEnc) != "" {
		current.Cold.Backend = loggingconfig.BackendClickHouse
		current.ClickHouse.Endpoint = clickhouseEndpoint
		current.ClickHouse.Username = clickhouseUser
		current.ClickHouse.Database = clickhouseDatabase
		current.ClickHouse.Table = "request_logs"
		current.ClickHouse.MigrationEnabled = true
	}
	if opensearchPassword != "" || opensearchAPIKey != "" || strings.TrimSpace(current.OpenSearch.PasswordEnc) != "" || strings.TrimSpace(current.OpenSearch.APIKeyEnc) != "" {
		current.Backend = loggingconfig.BackendOpenSearch
		current.Hot.Backend = loggingconfig.BackendOpenSearch
		if current.Cold.Backend != loggingconfig.BackendClickHouse {
			current.Cold.Backend = loggingconfig.BackendOpenSearch
		}
		current.OpenSearch.Endpoint = opensearchEndpoint
		current.OpenSearch.Username = opensearchUsername
		current.OpenSearch.IndexPrefix = opensearchIndexPrefix
		current.OpenSearch.RequestsIndex = opensearchIndexPrefix + "-requests"
		current.OpenSearch.EventsIndex = opensearchIndexPrefix + "-events"
		current.OpenSearch.ActivityIndex = opensearchIndexPrefix + "-activity"
	}
	if parseEnvBool(os.Getenv("VAULT_ENABLED")) || (vaultAddr != "" && vaultToken != "") || strings.TrimSpace(current.Vault.TokenEnc) != "" {
		current.SecretProvider = loggingconfig.SecretProviderVault
		current.Vault.Enabled = true
		current.Vault.Address = vaultAddr
		current.Vault.Mount = vaultMount
		current.Vault.PathPrefix = vaultPathPrefix
		current.Vault.TLSSkipVerify = parseEnvBool(os.Getenv("VAULT_TLS_SKIP_VERIFY"))
	}
	if strings.TrimSpace(current.ClickHouse.PasswordEnc) == "" && clickhousePassword != "" {
		if encrypted, err := secretcrypto.Encrypt("waf:logging:clickhouse", clickhousePassword, strings.TrimSpace(runtimeSettingsState.pepper)); err == nil {
			current.ClickHouse.PasswordEnc = encrypted
		}
	}
	if strings.TrimSpace(current.OpenSearch.PasswordEnc) == "" && opensearchPassword != "" {
		if encrypted, err := secretcrypto.Encrypt("waf:logging:opensearch:password", opensearchPassword, strings.TrimSpace(runtimeSettingsState.pepper)); err == nil {
			current.OpenSearch.PasswordEnc = encrypted
		}
	}
	if strings.TrimSpace(current.OpenSearch.APIKeyEnc) == "" && opensearchAPIKey != "" {
		if encrypted, err := secretcrypto.Encrypt("waf:logging:opensearch:api_key", opensearchAPIKey, strings.TrimSpace(runtimeSettingsState.pepper)); err == nil {
			current.OpenSearch.APIKeyEnc = encrypted
		}
	}
	if strings.TrimSpace(current.Vault.TokenEnc) == "" && vaultToken != "" {
		if encrypted, err := secretcrypto.Encrypt("waf:logging:vault:token", vaultToken, strings.TrimSpace(runtimeSettingsState.pepper)); err == nil {
			current.Vault.TokenEnc = encrypted
		}
	}
	current.Routing.WriteRequestsToHot = current.Hot.Backend == loggingconfig.BackendOpenSearch
	current.Routing.WriteRequestsToCold = current.Cold.Backend == loggingconfig.BackendClickHouse || (current.Cold.Backend == loggingconfig.BackendOpenSearch && current.Hot.Backend != loggingconfig.BackendOpenSearch)
	current.Routing.WriteEventsToHot = current.Hot.Backend == loggingconfig.BackendOpenSearch
	current.Routing.WriteEventsToCold = current.Cold.Backend == loggingconfig.BackendClickHouse || (current.Cold.Backend == loggingconfig.BackendOpenSearch && current.Hot.Backend != loggingconfig.BackendOpenSearch)
	current.Routing.WriteActivityToHot = current.Hot.Backend == loggingconfig.BackendOpenSearch
	current.Routing.WriteActivityToCold = current.Cold.Backend == loggingconfig.BackendClickHouse || (current.Cold.Backend == loggingconfig.BackendOpenSearch && current.Hot.Backend != loggingconfig.BackendOpenSearch)
	current.Routing.KeepLocalFallback = true
	current.ClickHouse.Password = ""
	current.OpenSearch.Password = ""
	current.OpenSearch.APIKey = ""
	current.Vault.Token = ""
	if current.Retention.HotDays <= 0 {
		current.Retention.HotDays = loggingconfig.DefaultHotDays
	}
	if current.Retention.ColdDays <= 0 {
		current.Retention.ColdDays = loggingconfig.DefaultColdDays
	}
	return loggingconfig.Normalize(current)
}

func reconcileLoggingSettingsFromEnv(current loggingconfig.Settings) loggingconfig.Settings {
	current = loggingconfig.Normalize(current)
	clickhousePassword := strings.TrimSpace(os.Getenv("CLICKHOUSE_PASSWORD"))
	opensearchPassword := strings.TrimSpace(os.Getenv("OPENSEARCH_PASSWORD"))
	opensearchAPIKey := strings.TrimSpace(os.Getenv("OPENSEARCH_API_KEY"))
	vaultToken := envSecretValue("VAULT_TOKEN", "VAULT_TOKEN_FILE")

	if current.Cold.Backend == loggingconfig.BackendClickHouse && strings.TrimSpace(current.ClickHouse.PasswordEnc) == "" {
		if clickhousePassword == "" {
			if current.Hot.Backend == loggingconfig.BackendOpenSearch {
				current.Backend = loggingconfig.BackendOpenSearch
				current.Cold.Backend = loggingconfig.BackendOpenSearch
			} else {
				current.Backend = loggingconfig.BackendFile
				current.Cold.Backend = loggingconfig.BackendFile
			}
		} else {
			if encrypted, err := secretcrypto.Encrypt("waf:logging:clickhouse", clickhousePassword, strings.TrimSpace(runtimeSettingsState.pepper)); err == nil {
				current.ClickHouse.PasswordEnc = encrypted
			}
		}
	}
	if current.Hot.Backend == loggingconfig.BackendOpenSearch && strings.TrimSpace(current.OpenSearch.PasswordEnc) == "" && strings.TrimSpace(current.OpenSearch.APIKeyEnc) == "" {
		if opensearchPassword == "" && opensearchAPIKey == "" {
			current.Hot.Backend = loggingconfig.BackendFile
		} else {
			if opensearchPassword != "" {
				if encrypted, err := secretcrypto.Encrypt("waf:logging:opensearch:password", opensearchPassword, strings.TrimSpace(runtimeSettingsState.pepper)); err == nil {
					current.OpenSearch.PasswordEnc = encrypted
				}
			}
			if opensearchAPIKey != "" {
				if encrypted, err := secretcrypto.Encrypt("waf:logging:opensearch:api_key", opensearchAPIKey, strings.TrimSpace(runtimeSettingsState.pepper)); err == nil {
					current.OpenSearch.APIKeyEnc = encrypted
				}
			}
		}
	}
	if current.SecretProvider == loggingconfig.SecretProviderVault && current.Vault.Enabled && strings.TrimSpace(current.Vault.TokenEnc) == "" {
		if vaultToken == "" {
			current.SecretProvider = loggingconfig.SecretProviderFile
			current.Vault.Enabled = false
		} else {
			if encrypted, err := secretcrypto.Encrypt("waf:logging:vault:token", vaultToken, strings.TrimSpace(runtimeSettingsState.pepper)); err == nil {
				current.Vault.TokenEnc = encrypted
			}
		}
	}
	if endpoint := strings.TrimRight(strings.TrimSpace(os.Getenv("CLICKHOUSE_ENDPOINT")), "/"); endpoint != "" {
		current.ClickHouse.Endpoint = endpoint
	} else if current.Cold.Backend == loggingconfig.BackendClickHouse && strings.TrimSpace(current.ClickHouse.Endpoint) == "" {
		current.ClickHouse.Endpoint = "http://clickhouse:8123"
	}
	if username := strings.TrimSpace(os.Getenv("CLICKHOUSE_USER")); username != "" {
		current.ClickHouse.Username = username
	}
	if database := strings.TrimSpace(os.Getenv("CLICKHOUSE_DB")); database != "" {
		current.ClickHouse.Database = database
	}
	if endpoint := strings.TrimRight(strings.TrimSpace(os.Getenv("OPENSEARCH_ENDPOINT")), "/"); endpoint != "" {
		current.OpenSearch.Endpoint = endpoint
	} else if (current.Hot.Backend == loggingconfig.BackendOpenSearch || current.Cold.Backend == loggingconfig.BackendOpenSearch) && strings.TrimSpace(current.OpenSearch.Endpoint) == "" {
		current.OpenSearch.Endpoint = "http://opensearch:9200"
	}
	if username := strings.TrimSpace(os.Getenv("OPENSEARCH_USERNAME")); username != "" {
		current.OpenSearch.Username = username
	} else if username := strings.TrimSpace(os.Getenv("OPENSEARCH_USER")); username != "" {
		current.OpenSearch.Username = username
	}
	if prefix := strings.TrimSpace(os.Getenv("OPENSEARCH_INDEX_PREFIX")); prefix != "" {
		current.OpenSearch.IndexPrefix = prefix
	}
	if addr := strings.TrimRight(strings.TrimSpace(os.Getenv("VAULT_ADDR")), "/"); addr != "" {
		current.Vault.Address = addr
	}
	if mount := strings.TrimSpace(os.Getenv("VAULT_MOUNT")); mount != "" {
		current.Vault.Mount = mount
	}
	if prefix := strings.TrimSpace(os.Getenv("VAULT_PATH_PREFIX")); prefix != "" {
		current.Vault.PathPrefix = prefix
	}
	current.Vault.TLSSkipVerify = current.Vault.TLSSkipVerify || parseEnvBool(os.Getenv("VAULT_TLS_SKIP_VERIFY"))
	current.Routing.KeepLocalFallback = true
	if current.Hot.Backend == loggingconfig.BackendOpenSearch {
		current.Backend = loggingconfig.BackendOpenSearch
	} else if current.Cold.Backend == loggingconfig.BackendOpenSearch {
		current.Backend = loggingconfig.BackendOpenSearch
	} else if current.Cold.Backend == loggingconfig.BackendClickHouse {
		current.Backend = loggingconfig.BackendClickHouse
	} else {
		current.Backend = loggingconfig.BackendFile
	}
	if strings.TrimSpace(current.ClickHouse.Table) == "" {
		current.ClickHouse.Table = "request_logs"
	}
	return loggingconfig.Normalize(current)
}

func parseEnvBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func envSecretValue(envName string, fileEnvName string) string {
	if value := strings.TrimSpace(os.Getenv(envName)); value != "" {
		return value
	}
	path := strings.TrimSpace(os.Getenv(fileEnvName))
	if path == "" {
		return ""
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(content))
}

func NewSettingsRuntimeHandler(settingsRoot string, runtimeHealthURL string, pepper string) *SettingsRuntimeHandler {
	return NewSettingsRuntimeHandlerWithBackend(settingsRoot, runtimeHealthURL, nil, pepper)
}

func NewSettingsRuntimeHandlerWithBackend(settingsRoot string, runtimeHealthURL string, backend storage.Backend, pepper string) *SettingsRuntimeHandler {
	runtimeSettingsState.mu.Lock()
	defer runtimeSettingsState.mu.Unlock()
	runtimeSettingsState.pepper = strings.TrimSpace(pepper)
	if !runtimeSettingsState.initialized {
		runtimeSettingsState.logging = defaultLoggingSettingsFromEnv(runtimeSettingsState.logging)
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
				runtimeSettingsState.lastCheckedAt = stored.LastCheckedAt
				runtimeSettingsState.latestVersion = stored.LatestVersion
				runtimeSettingsState.releaseURL = stored.ReleaseURL
				runtimeSettingsState.hasUpdate = stored.HasUpdate
				runtimeSettingsState.storage = stored.Storage
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
	runtimeSettingsState.logging = reconcileLoggingSettingsFromEnv(runtimeSettingsState.logging)
	savePersistedRuntimeSettingsLocked()
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

func CurrentRuntimeLanguage() string {
	runtimeSettingsState.mu.RLock()
	defer runtimeSettingsState.mu.RUnlock()
	return normalizeRuntimeLanguage(runtimeSettingsState.language)
}

func (h *SettingsRuntimeHandler) responsePayload() map[string]any {
	runtimeSettingsState.mu.RLock()
	defer runtimeSettingsState.mu.RUnlock()
	return responsePayloadLocked(nil)
}

func currentSecretProviderLocked() string {
	provider := strings.TrimSpace(runtimeSettingsState.logging.SecretProvider)
	if provider == "" {
		return loggingconfig.SecretProviderFile
	}
	return provider
}

func currentLoggingSummaryLocked() map[string]any {
	logging := loggingconfig.MaskSecrets(runtimeSettingsState.logging)
	return map[string]any{
		"hot_backend":        logging.Hot.Backend,
		"cold_backend":       logging.Cold.Backend,
		"secret_provider":    currentSecretProviderLocked(),
		"retention":          logging.Retention,
		"routing":            logging.Routing,
		"opensearch_enabled": loggingconfig.EnabledOpenSearch(runtimeSettingsState.logging),
		"clickhouse_enabled": loggingconfig.EnabledClickHouse(runtimeSettingsState.logging),
		"vault_enabled":      loggingconfig.EnabledVault(runtimeSettingsState.logging),
	}
}

func responsePayloadLocked(indexes map[string]any) map[string]any {
	payload := map[string]any{
		"deployment_mode":       "standalone",
		"app_version":           appmeta.AppVersion,
		"update_checks_enabled": runtimeSettingsState.updateChecksEnabled,
		"language":              normalizeRuntimeLanguage(runtimeSettingsState.language),
		"storage":               runtimeSettingsState.storage,
		"logging":               loggingconfig.MaskSecrets(runtimeSettingsState.logging),
		"logging_summary":       currentLoggingSummaryLocked(),
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
	if strings.TrimSpace(values.Get("storage_indexes_limit")) == "" && strings.TrimSpace(values.Get("storage_indexes_offset")) == "" {
		return nil
	}
	stream := normalizeStorageIndexStream(values.Get("stream"))
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
	indexes, err := fetchStorageIndexes(stream, limit, offset)
	if err != nil {
		return map[string]any{
			"stream": stream,
			"items":  []map[string]any{},
			"total":  0,
			"limit":  limit,
			"offset": offset,
			"error":  err.Error(),
		}
	}
	return indexes
}

func normalizeStorageIndexStream(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "events":
		return "events"
	case "activity":
		return "activity"
	default:
		return "requests"
	}
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

func (f *runtimeIndexFetcher) Fetch(stream string, limit int, offset int) (map[string]any, error) {
	if f == nil || strings.TrimSpace(f.url) == "" {
		return map[string]any{
			"stream": stream,
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
	if strings.TrimSpace(stream) != "" {
		q.Set("stream", stream)
	}
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

func (f *runtimeIndexFetcher) Delete(stream string, date string) error {
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
	if strings.TrimSpace(stream) != "" {
		q.Set("stream", stream)
	}
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

func fetchStorageIndexes(stream string, limit int, offset int) (map[string]any, error) {
	stream = normalizeStorageIndexStream(stream)
	if stream == "requests" {
		if runtimeRequestIndexes == nil {
			return map[string]any{
				"stream": stream,
				"items":  []map[string]any{},
				"total":  0,
				"limit":  limit,
				"offset": offset,
			}, nil
		}
		return runtimeRequestIndexes.Fetch(stream, limit, offset)
	}
	runtimeSettingsState.mu.RLock()
	logging := runtimeSettingsState.logging
	pepper := runtimeSettingsState.pepper
	runtimeSettingsState.mu.RUnlock()
	return fetchOpenSearchStorageIndexes(stream, logging, pepper, limit, offset)
}

func deleteStorageIndexes(stream string, day string) error {
	stream = normalizeStorageIndexStream(stream)
	if stream == "requests" {
		if runtimeRequestIndexes == nil {
			return fmt.Errorf("runtime indexes fetcher unavailable")
		}
		return runtimeRequestIndexes.Delete(stream, day)
	}
	runtimeSettingsState.mu.RLock()
	logging := runtimeSettingsState.logging
	pepper := runtimeSettingsState.pepper
	runtimeSettingsState.mu.RUnlock()
	return deleteOpenSearchStorageIndex(stream, day, logging, pepper)
}

func responsePayloadWithoutIndexesLocked() map[string]any {
	return map[string]any{
		"deployment_mode":       "standalone",
		"app_version":           appmeta.AppVersion,
		"update_checks_enabled": runtimeSettingsState.updateChecksEnabled,
		"language":              normalizeRuntimeLanguage(runtimeSettingsState.language),
		"storage":               runtimeSettingsState.storage,
		"logging":               loggingconfig.MaskSecrets(runtimeSettingsState.logging),
		"logging_summary":       currentLoggingSummaryLocked(),
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
			next, err := parseLoggingSettings(typed, runtimeSettingsState.logging, runtimeSettingsState.pepper)
			if err != nil {
				runtimeSettingsState.mu.Unlock()
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
				return
			}
			runtimeSettingsState.logging = next
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
	if value, ok := raw["hot_index_days"]; ok {
		parsed, err := parsePositiveRetentionInt(value)
		if err != nil {
			return StorageRetention{}, err
		}
		out.HotIndexDays = parsed
	}
	if value, ok := raw["cold_index_days"]; ok {
		parsed, err := parsePositiveRetentionInt(value)
		if err != nil {
			return StorageRetention{}, err
		}
		out.ColdIndexDays = parsed
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
	if input.HotIndexDays <= 0 {
		input.HotIndexDays = loggingconfig.DefaultHotDays
	}
	if input.ColdIndexDays <= 0 {
		input.ColdIndexDays = loggingconfig.DefaultColdDays
	}
	if input.HotIndexDays > loggingconfig.MaxHotDays {
		input.HotIndexDays = loggingconfig.MaxHotDays
	}
	if input.ColdIndexDays > loggingconfig.MaxColdDays {
		input.ColdIndexDays = loggingconfig.MaxColdDays
	}
	return input
}

func fetchOpenSearchStorageIndexes(stream string, settings loggingconfig.Settings, pepper string, limit int, offset int) (map[string]any, error) {
	cfg, err := currentOpenSearchAdminConfig(settings, pepper)
	if err != nil {
		return nil, err
	}
	stream = normalizeStorageIndexStream(stream)
	if !cfg.enabled {
		return map[string]any{
			"stream": stream,
			"items":  []storageIndexItem{},
			"total":  0,
			"limit":  limit,
			"offset": offset,
		}, nil
	}
	indexName, timeField := openSearchIndexMetadata(settings, stream)
	if indexName == "" || timeField == "" {
		return map[string]any{
			"stream": stream,
			"items":  []storageIndexItem{},
			"total":  0,
			"limit":  limit,
			"offset": offset,
		}, nil
	}
	body := map[string]any{
		"size": 0,
		"aggs": map[string]any{
			"by_day": map[string]any{
				"date_histogram": map[string]any{
					"field":             timeField,
					"calendar_interval": "day",
					"order":             map[string]any{"_key": "desc"},
				},
			},
		},
	}
	var payload struct {
		Aggregations struct {
			ByDay struct {
				Buckets []struct {
					KeyAsString string `json:"key_as_string"`
					DocCount    int    `json:"doc_count"`
				} `json:"buckets"`
			} `json:"by_day"`
		} `json:"aggregations"`
	}
	if err := doOpenSearchJSONRequest(cfg, http.MethodPost, "/"+indexName+"/_search", body, &payload); err != nil {
		return nil, err
	}
	total := len(payload.Aggregations.ByDay.Buckets)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	items := make([]storageIndexItem, 0, maxInt(0, end-offset))
	for _, bucket := range payload.Aggregations.ByDay.Buckets[offset:end] {
		day := strings.TrimSpace(bucket.KeyAsString)
		if len(day) >= 10 {
			day = day[:10]
		}
		items = append(items, storageIndexItem{
			Stream:      stream,
			Date:        day,
			FileName:    indexName,
			Lines:       bucket.DocCount,
			SizeBytes:   0,
			UpdatedAt:   bucket.KeyAsString,
			StorageType: "opensearch-hot",
		})
	}
	return map[string]any{
		"stream":       stream,
		"items":        items,
		"total":        total,
		"limit":        limit,
		"offset":       offset,
		"storage_type": "opensearch-hot",
	}, nil
}

func deleteOpenSearchStorageIndex(stream string, day string, settings loggingconfig.Settings, pepper string) error {
	trimmedDay := strings.TrimSpace(day)
	if trimmedDay == "" {
		return fmt.Errorf("date is required")
	}
	if _, err := time.Parse("2006-01-02", trimmedDay); err != nil {
		return fmt.Errorf("date must be in YYYY-MM-DD format")
	}
	cfg, err := currentOpenSearchAdminConfig(settings, pepper)
	if err != nil {
		return err
	}
	if !cfg.enabled {
		return nil
	}
	indexName, timeField := openSearchIndexMetadata(settings, normalizeStorageIndexStream(stream))
	if indexName == "" || timeField == "" {
		return nil
	}
	body := map[string]any{
		"query": map[string]any{
			"range": map[string]any{
				timeField: map[string]any{
					"gte": trimmedDay + "T00:00:00Z",
					"lte": trimmedDay + "T23:59:59.999999999Z",
				},
			},
		},
	}
	return doOpenSearchJSONRequest(cfg, http.MethodPost, "/"+indexName+"/_delete_by_query", body, nil)
}

type openSearchAdminConfig struct {
	enabled  bool
	endpoint string
	username string
	password string
	apiKey   string
}

func currentOpenSearchAdminConfig(settings loggingconfig.Settings, pepper string) (openSearchAdminConfig, error) {
	settings = loggingconfig.Normalize(settings)
	if !loggingconfig.EnabledOpenSearch(settings) {
		return openSearchAdminConfig{}, nil
	}
	password := strings.TrimSpace(settings.OpenSearch.Password)
	apiKey := strings.TrimSpace(settings.OpenSearch.APIKey)
	if settings.SecretProvider == loggingconfig.SecretProviderVault && settings.Vault.Enabled {
		client, err := currentVaultClient(settings, pepper)
		if err != nil {
			return openSearchAdminConfig{}, err
		}
		if password == "" {
			password, err = client.Get("logging/opensearch", "password")
			if err != nil {
				return openSearchAdminConfig{}, err
			}
		}
		if apiKey == "" {
			apiKey, err = client.Get("logging/opensearch", "api_key")
			if err != nil {
				return openSearchAdminConfig{}, err
			}
		}
	} else {
		var err error
		if password == "" && strings.TrimSpace(settings.OpenSearch.PasswordEnc) != "" {
			password, err = secretcrypto.Decrypt("waf:logging:opensearch:password", settings.OpenSearch.PasswordEnc, pepper)
			if err != nil {
				return openSearchAdminConfig{}, err
			}
		}
		if apiKey == "" && strings.TrimSpace(settings.OpenSearch.APIKeyEnc) != "" {
			apiKey, err = secretcrypto.Decrypt("waf:logging:opensearch:api_key", settings.OpenSearch.APIKeyEnc, pepper)
			if err != nil {
				return openSearchAdminConfig{}, err
			}
		}
	}
	return openSearchAdminConfig{
		enabled:  true,
		endpoint: strings.TrimRight(strings.TrimSpace(settings.OpenSearch.Endpoint), "/"),
		username: strings.TrimSpace(settings.OpenSearch.Username),
		password: strings.TrimSpace(password),
		apiKey:   strings.TrimSpace(apiKey),
	}, nil
}

func currentVaultClient(settings loggingconfig.Settings, pepper string) (vaultkv.Client, error) {
	token := strings.TrimSpace(settings.Vault.Token)
	if token == "" && strings.TrimSpace(settings.Vault.TokenEnc) != "" {
		decrypted, err := secretcrypto.Decrypt("waf:logging:vault:token", settings.Vault.TokenEnc, pepper)
		if err != nil {
			return vaultkv.Client{}, err
		}
		token = decrypted
	}
	return vaultkv.Client{
		Address:       settings.Vault.Address,
		Token:         token,
		Mount:         settings.Vault.Mount,
		PathPrefix:    settings.Vault.PathPrefix,
		TLSSkipVerify: settings.Vault.TLSSkipVerify,
	}, nil
}

func openSearchIndexMetadata(settings loggingconfig.Settings, stream string) (string, string) {
	switch normalizeStorageIndexStream(stream) {
	case "events":
		return strings.TrimSpace(settings.OpenSearch.EventsIndex), "occurred_at"
	case "activity":
		return strings.TrimSpace(settings.OpenSearch.ActivityIndex), "occurred_at"
	default:
		return strings.TrimSpace(settings.OpenSearch.RequestsIndex), "timestamp"
	}
}

func doOpenSearchJSONRequest(cfg openSearchAdminConfig, method string, path string, body any, out any) error {
	if !cfg.enabled || strings.TrimSpace(cfg.endpoint) == "" {
		return nil
	}
	var reader io.Reader
	if body != nil {
		content, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = strings.NewReader(string(content))
	}
	req, err := http.NewRequest(method, cfg.endpoint+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if cfg.apiKey != "" {
		req.Header.Set("Authorization", "ApiKey "+cfg.apiKey)
	} else if cfg.username != "" || cfg.password != "" {
		req.SetBasicAuth(cfg.username, cfg.password)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		content, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		return fmt.Errorf("opensearch request failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(content)))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(out)
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
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
	runtimeSettingsState.logging = loggingconfig.Normalize(stored.Logging)
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
		UpdateChecksEnabled: runtimeSettingsState.updateChecksEnabled,
		Language:            normalizeRuntimeLanguage(runtimeSettingsState.language),
		LastCheckedAt:       runtimeSettingsState.lastCheckedAt,
		LatestVersion:       runtimeSettingsState.latestVersion,
		ReleaseURL:          runtimeSettingsState.releaseURL,
		HasUpdate:           runtimeSettingsState.hasUpdate,
		Storage:             normalizeStorageRetention(runtimeSettingsState.storage),
		Logging:             loggingconfig.Normalize(runtimeSettingsState.logging),
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

func normalizeRuntimeLanguage(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "ru", "de", "sr", "zh":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "en"
	}
}

func parseLoggingSettings(raw map[string]any, current loggingconfig.Settings, pepper string) (loggingconfig.Settings, error) {
	content, err := json.Marshal(raw)
	if err != nil {
		return loggingconfig.Settings{}, fmt.Errorf("encode logging settings: %w", err)
	}
	var incoming loggingconfig.Settings
	if err := json.Unmarshal(content, &incoming); err != nil {
		return loggingconfig.Settings{}, fmt.Errorf("decode logging settings: %w", err)
	}
	current = loggingconfig.Normalize(current)
	incoming = loggingconfig.Normalize(incoming)

	if strings.TrimSpace(incoming.ClickHouse.Password) == loggingconfig.MaskedSecretValue {
		incoming.ClickHouse.PasswordEnc = current.ClickHouse.PasswordEnc
		incoming.ClickHouse.Password = ""
	}
	if strings.TrimSpace(incoming.OpenSearch.Password) == loggingconfig.MaskedSecretValue {
		incoming.OpenSearch.PasswordEnc = current.OpenSearch.PasswordEnc
		incoming.OpenSearch.Password = ""
	}
	if strings.TrimSpace(incoming.OpenSearch.APIKey) == loggingconfig.MaskedSecretValue {
		incoming.OpenSearch.APIKeyEnc = current.OpenSearch.APIKeyEnc
		incoming.OpenSearch.APIKey = ""
	}
	if strings.TrimSpace(incoming.Vault.Token) == loggingconfig.MaskedSecretValue {
		incoming.Vault.TokenEnc = current.Vault.TokenEnc
		incoming.Vault.Token = ""
	}
	if strings.TrimSpace(incoming.ClickHouse.Password) != "" {
		secretEnc, err := secretcrypto.Encrypt("waf:logging:clickhouse", incoming.ClickHouse.Password, pepper)
		if err != nil {
			return loggingconfig.Settings{}, fmt.Errorf("encrypt clickhouse password: %w", err)
		}
		incoming.ClickHouse.PasswordEnc = secretEnc
		incoming.ClickHouse.Password = ""
	}
	if strings.TrimSpace(incoming.OpenSearch.Password) != "" {
		secretEnc, err := secretcrypto.Encrypt("waf:logging:opensearch:password", incoming.OpenSearch.Password, pepper)
		if err != nil {
			return loggingconfig.Settings{}, fmt.Errorf("encrypt opensearch password: %w", err)
		}
		incoming.OpenSearch.PasswordEnc = secretEnc
		incoming.OpenSearch.Password = ""
	}
	if strings.TrimSpace(incoming.OpenSearch.APIKey) != "" {
		secretEnc, err := secretcrypto.Encrypt("waf:logging:opensearch:api_key", incoming.OpenSearch.APIKey, pepper)
		if err != nil {
			return loggingconfig.Settings{}, fmt.Errorf("encrypt opensearch api key: %w", err)
		}
		incoming.OpenSearch.APIKeyEnc = secretEnc
		incoming.OpenSearch.APIKey = ""
	}
	if strings.TrimSpace(incoming.Vault.Token) != "" {
		secretEnc, err := secretcrypto.Encrypt("waf:logging:vault:token", incoming.Vault.Token, pepper)
		if err != nil {
			return loggingconfig.Settings{}, fmt.Errorf("encrypt vault token: %w", err)
		}
		incoming.Vault.TokenEnc = secretEnc
		incoming.Vault.Token = ""
	}
	if strings.TrimSpace(incoming.ClickHouse.PasswordEnc) == "" {
		incoming.ClickHouse.PasswordEnc = current.ClickHouse.PasswordEnc
	}
	if strings.TrimSpace(incoming.OpenSearch.PasswordEnc) == "" {
		incoming.OpenSearch.PasswordEnc = current.OpenSearch.PasswordEnc
	}
	if strings.TrimSpace(incoming.OpenSearch.APIKeyEnc) == "" {
		incoming.OpenSearch.APIKeyEnc = current.OpenSearch.APIKeyEnc
	}
	if strings.TrimSpace(incoming.Vault.TokenEnc) == "" {
		incoming.Vault.TokenEnc = current.Vault.TokenEnc
	}
	if incoming.Cold.Backend == loggingconfig.BackendClickHouse && strings.TrimSpace(incoming.ClickHouse.Endpoint) == "" {
		return loggingconfig.Settings{}, fmt.Errorf("clickhouse endpoint is required when backend is clickhouse")
	}
	if (incoming.Hot.Backend == loggingconfig.BackendOpenSearch || incoming.Cold.Backend == loggingconfig.BackendOpenSearch) && strings.TrimSpace(incoming.OpenSearch.Endpoint) == "" {
		return loggingconfig.Settings{}, fmt.Errorf("opensearch endpoint is required when opensearch storage is enabled")
	}
	if incoming.SecretProvider == loggingconfig.SecretProviderVault && incoming.Vault.Enabled && strings.TrimSpace(incoming.Vault.Address) == "" {
		return loggingconfig.Settings{}, fmt.Errorf("vault address is required when vault secret provider is enabled")
	}
	if incoming.SecretProvider == loggingconfig.SecretProviderVault && incoming.Vault.Enabled {
		if err := storeLoggingSecretsInVault(incoming); err != nil {
			return loggingconfig.Settings{}, err
		}
		// Once secrets are safely persisted in Vault, keep only masked/runtime references in local settings.
		incoming.ClickHouse.PasswordEnc = ""
		incoming.ClickHouse.Password = ""
		incoming.OpenSearch.PasswordEnc = ""
		incoming.OpenSearch.Password = ""
		incoming.OpenSearch.APIKeyEnc = ""
		incoming.OpenSearch.APIKey = ""
	}
	return loggingconfig.Normalize(incoming), nil
}

func storeLoggingSecretsInVault(input loggingconfig.Settings) error {
	token := strings.TrimSpace(input.Vault.Token)
	if token == "" && strings.TrimSpace(input.Vault.TokenEnc) != "" {
		decrypted, err := secretcrypto.Decrypt("waf:logging:vault:token", input.Vault.TokenEnc, strings.TrimSpace(runtimeSettingsState.pepper))
		if err != nil {
			return fmt.Errorf("decrypt vault token: %w", err)
		}
		token = strings.TrimSpace(decrypted)
	}
	clickhousePassword := strings.TrimSpace(input.ClickHouse.Password)
	if clickhousePassword == "" && strings.TrimSpace(input.ClickHouse.PasswordEnc) != "" {
		decrypted, err := secretcrypto.Decrypt("waf:logging:clickhouse", input.ClickHouse.PasswordEnc, strings.TrimSpace(runtimeSettingsState.pepper))
		if err != nil {
			return fmt.Errorf("decrypt clickhouse password: %w", err)
		}
		clickhousePassword = strings.TrimSpace(decrypted)
	}
	opensearchPassword := strings.TrimSpace(input.OpenSearch.Password)
	if opensearchPassword == "" && strings.TrimSpace(input.OpenSearch.PasswordEnc) != "" {
		decrypted, err := secretcrypto.Decrypt("waf:logging:opensearch:password", input.OpenSearch.PasswordEnc, strings.TrimSpace(runtimeSettingsState.pepper))
		if err != nil {
			return fmt.Errorf("decrypt opensearch password: %w", err)
		}
		opensearchPassword = strings.TrimSpace(decrypted)
	}
	opensearchAPIKey := strings.TrimSpace(input.OpenSearch.APIKey)
	if opensearchAPIKey == "" && strings.TrimSpace(input.OpenSearch.APIKeyEnc) != "" {
		decrypted, err := secretcrypto.Decrypt("waf:logging:opensearch:api_key", input.OpenSearch.APIKeyEnc, strings.TrimSpace(runtimeSettingsState.pepper))
		if err != nil {
			return fmt.Errorf("decrypt opensearch api key: %w", err)
		}
		opensearchAPIKey = strings.TrimSpace(decrypted)
	}
	client := vaultkv.Client{
		Address:       strings.TrimSpace(input.Vault.Address),
		Token:         token,
		Mount:         strings.TrimSpace(input.Vault.Mount),
		PathPrefix:    strings.TrimSpace(input.Vault.PathPrefix),
		TLSSkipVerify: input.Vault.TLSSkipVerify,
	}
	if strings.TrimSpace(client.Address) == "" {
		return fmt.Errorf("vault address is required when vault secret provider is enabled")
	}
	if strings.TrimSpace(client.Token) == "" {
		return fmt.Errorf("vault token is required when vault secret provider is enabled")
	}
	if clickhousePassword != "" {
		if err := client.Put("logging/clickhouse", map[string]string{"password": clickhousePassword}); err != nil {
			return fmt.Errorf("store clickhouse secret in vault: %w", err)
		}
	}
	if opensearchPassword != "" || opensearchAPIKey != "" {
		payload := map[string]string{}
		if opensearchPassword != "" {
			payload["password"] = opensearchPassword
		}
		if opensearchAPIKey != "" {
			payload["api_key"] = opensearchAPIKey
		}
		if err := client.Put("logging/opensearch", payload); err != nil {
			return fmt.Errorf("store opensearch secrets in vault: %w", err)
		}
	}
	return nil
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
