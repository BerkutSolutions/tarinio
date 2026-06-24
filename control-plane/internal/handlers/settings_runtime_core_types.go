package handlers

import (
	"net/http"
	"sync"
	"time"

	"waf/control-plane/internal/appmeta"
	"waf/control-plane/internal/storage"
	"waf/internal/loggingconfig"
)

type SettingsRuntimeHandler struct{}

type runtimeIndexFetcher struct {
	url    string
	client *http.Client
	token  string
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

type RuntimeSecuritySettings struct {
	AllowInsecureVaultTLS      bool `json:"allow_insecure_vault_tls"`
	LoginRateLimitEnabled      bool `json:"login_rate_limit_enabled"`
	LoginRateLimitMaxAttempts  int  `json:"login_rate_limit_max_attempts"`
	LoginRateLimitWindowSecond int  `json:"login_rate_limit_window_seconds"`
	LoginRateLimitBlockSecond  int  `json:"login_rate_limit_block_seconds"`
}

type persistedRuntimeSettings struct {
	UpdateChecksEnabled bool                    `json:"update_checks_enabled"`
	Language            string                  `json:"language,omitempty"`
	LastCheckedAt       string                  `json:"last_checked_at,omitempty"`
	LatestVersion       string                  `json:"latest_version,omitempty"`
	ReleaseURL          string                  `json:"release_url,omitempty"`
	HasUpdate           bool                    `json:"has_update"`
	Storage             StorageRetention        `json:"storage"`
	Security            RuntimeSecuritySettings `json:"security,omitempty"`
	Logging             loggingconfig.Settings  `json:"logging,omitempty"`
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
	security            RuntimeSecuritySettings
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
	security: RuntimeSecuritySettings{
		AllowInsecureVaultTLS:      false,
		LoginRateLimitEnabled:      true,
		LoginRateLimitMaxAttempts:  10,
		LoginRateLimitWindowSecond: 300,
		LoginRateLimitBlockSecond:  600,
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
