package loggingconfig

import "strings"

const (
	BackendFile         = "file"
	BackendClickHouse   = "clickhouse"
	BackendOpenSearch   = "opensearch"
	SecretProviderFile  = "encrypted_file"
	SecretProviderVault = "vault"
	MaskedSecretValue   = "********"
	DefaultHotDays      = 30
	DefaultColdDays     = 730
	MaxHotDays          = 30
	MaxColdDays         = 730
)

type Settings struct {
	Backend        string             `json:"backend"`
	Hot            HotSettings        `json:"hot"`
	Cold           ColdSettings       `json:"cold"`
	Routing        RoutingSettings    `json:"routing"`
	Retention      RetentionSettings  `json:"retention"`
	Vault          VaultSettings      `json:"vault"`
	OpenSearch     OpenSearchSettings `json:"opensearch"`
	ClickHouse     ClickHouseSettings `json:"clickhouse"`
	SecretProvider string             `json:"secret_provider,omitempty"`
}

type HotSettings struct {
	Backend string `json:"backend,omitempty"`
}

type ColdSettings struct {
	Backend string `json:"backend,omitempty"`
}

type RoutingSettings struct {
	WriteRequestsToHot  bool `json:"write_requests_to_hot"`
	WriteRequestsToCold bool `json:"write_requests_to_cold"`
	WriteEventsToHot    bool `json:"write_events_to_hot"`
	WriteEventsToCold   bool `json:"write_events_to_cold"`
	WriteActivityToHot  bool `json:"write_activity_to_hot"`
	WriteActivityToCold bool `json:"write_activity_to_cold"`
	KeepLocalFallback   bool `json:"keep_local_fallback"`
}

type RetentionSettings struct {
	HotDays  int `json:"hot_days"`
	ColdDays int `json:"cold_days"`
}

type OpenSearchSettings struct {
	Endpoint      string `json:"endpoint,omitempty"`
	IndexPrefix   string `json:"index_prefix,omitempty"`
	Username      string `json:"username,omitempty"`
	Password      string `json:"password,omitempty"`
	PasswordEnc   string `json:"password_enc,omitempty"`
	APIKey        string `json:"api_key,omitempty"`
	APIKeyEnc     string `json:"api_key_enc,omitempty"`
	RequestsIndex string `json:"requests_index,omitempty"`
	EventsIndex   string `json:"events_index,omitempty"`
	ActivityIndex string `json:"activity_index,omitempty"`
}

type ClickHouseSettings struct {
	Endpoint         string `json:"endpoint,omitempty"`
	Database         string `json:"database,omitempty"`
	Username         string `json:"username,omitempty"`
	Password         string `json:"password,omitempty"`
	PasswordEnc      string `json:"password_enc,omitempty"`
	Table            string `json:"table,omitempty"`
	MigrationEnabled bool   `json:"migration_enabled"`
}

type VaultSettings struct {
	Enabled       bool   `json:"enabled"`
	Address       string `json:"address,omitempty"`
	Token         string `json:"token,omitempty"`
	TokenEnc      string `json:"token_enc,omitempty"`
	Mount         string `json:"mount,omitempty"`
	PathPrefix    string `json:"path_prefix,omitempty"`
	TLSSkipVerify bool   `json:"tls_skip_verify"`
}

func Normalize(input Settings) Settings {
	input.Backend = normalizeStorageBackend(input.Backend)
	input.SecretProvider = normalizeSecretProvider(input.SecretProvider, input.Vault.Enabled)

	input.Hot.Backend = normalizeHotBackend(input.Hot.Backend)
	input.Cold.Backend = normalizeColdBackend(input.Cold.Backend)
	if input.Hot.Backend == "" {
		input.Hot.Backend = legacyHotBackend(input.Backend)
	}
	if input.Cold.Backend == "" {
		input.Cold.Backend = legacyColdBackend(input.Backend)
	}

	input.OpenSearch.Endpoint = normalizeEndpoint(input.OpenSearch.Endpoint)
	input.OpenSearch.IndexPrefix = normalizeIdentifier(input.OpenSearch.IndexPrefix)
	if input.OpenSearch.IndexPrefix == "" {
		input.OpenSearch.IndexPrefix = "waf-hot"
	}
	input.OpenSearch.Username = strings.TrimSpace(input.OpenSearch.Username)
	input.OpenSearch.Password = strings.TrimSpace(input.OpenSearch.Password)
	input.OpenSearch.PasswordEnc = strings.TrimSpace(input.OpenSearch.PasswordEnc)
	input.OpenSearch.APIKey = strings.TrimSpace(input.OpenSearch.APIKey)
	input.OpenSearch.APIKeyEnc = strings.TrimSpace(input.OpenSearch.APIKeyEnc)
	input.OpenSearch.RequestsIndex = normalizeIdentifier(input.OpenSearch.RequestsIndex)
	if input.OpenSearch.RequestsIndex == "" {
		input.OpenSearch.RequestsIndex = "waf-requests"
	}
	input.OpenSearch.EventsIndex = normalizeIdentifier(input.OpenSearch.EventsIndex)
	if input.OpenSearch.EventsIndex == "" {
		input.OpenSearch.EventsIndex = "waf-events"
	}
	input.OpenSearch.ActivityIndex = normalizeIdentifier(input.OpenSearch.ActivityIndex)
	if input.OpenSearch.ActivityIndex == "" {
		input.OpenSearch.ActivityIndex = "waf-activity"
	}

	input.ClickHouse.Endpoint = normalizeEndpoint(input.ClickHouse.Endpoint)
	input.ClickHouse.Database = normalizeIdentifier(input.ClickHouse.Database)
	if input.ClickHouse.Database == "" {
		input.ClickHouse.Database = "waf_logs"
	}
	input.ClickHouse.Username = strings.TrimSpace(input.ClickHouse.Username)
	input.ClickHouse.Password = strings.TrimSpace(input.ClickHouse.Password)
	input.ClickHouse.PasswordEnc = strings.TrimSpace(input.ClickHouse.PasswordEnc)
	input.ClickHouse.Table = normalizeIdentifier(input.ClickHouse.Table)
	if input.ClickHouse.Table == "" {
		input.ClickHouse.Table = "request_logs"
	}

	input.Vault.Address = normalizeEndpoint(input.Vault.Address)
	input.Vault.Token = strings.TrimSpace(input.Vault.Token)
	input.Vault.TokenEnc = strings.TrimSpace(input.Vault.TokenEnc)
	input.Vault.Mount = normalizeIdentifierWithSlash(input.Vault.Mount)
	if input.Vault.Mount == "" {
		input.Vault.Mount = "secret"
	}
	input.Vault.PathPrefix = normalizeIdentifierWithSlash(input.Vault.PathPrefix)
	if input.Vault.PathPrefix == "" {
		input.Vault.PathPrefix = "tarinio"
	}

	if input.Retention.HotDays <= 0 {
		input.Retention.HotDays = DefaultHotDays
	}
	if input.Retention.ColdDays <= 0 {
		input.Retention.ColdDays = DefaultColdDays
	}
	if input.Retention.HotDays > MaxHotDays {
		input.Retention.HotDays = MaxHotDays
	}
	if input.Retention.ColdDays > MaxColdDays {
		input.Retention.ColdDays = MaxColdDays
	}

	normalizeRoutingDefaults(&input)
	return input
}

func normalizeRoutingDefaults(input *Settings) {
	if input == nil {
		return
	}
	if !input.Routing.WriteRequestsToHot && !input.Routing.WriteRequestsToCold &&
		!input.Routing.WriteEventsToHot && !input.Routing.WriteEventsToCold &&
		!input.Routing.WriteActivityToHot && !input.Routing.WriteActivityToCold &&
		!input.Routing.KeepLocalFallback {
		input.Routing.WriteRequestsToHot = input.Hot.Backend == BackendOpenSearch
		input.Routing.WriteRequestsToCold = input.Cold.Backend == BackendClickHouse || (input.Cold.Backend == BackendOpenSearch && input.Hot.Backend != BackendOpenSearch)
		input.Routing.WriteEventsToHot = input.Hot.Backend == BackendOpenSearch
		input.Routing.WriteEventsToCold = input.Cold.Backend == BackendClickHouse || (input.Cold.Backend == BackendOpenSearch && input.Hot.Backend != BackendOpenSearch)
		input.Routing.WriteActivityToHot = input.Hot.Backend == BackendOpenSearch
		input.Routing.WriteActivityToCold = input.Cold.Backend == BackendClickHouse || (input.Cold.Backend == BackendOpenSearch && input.Hot.Backend != BackendOpenSearch)
		input.Routing.KeepLocalFallback = true
	}
}

func normalizeEndpoint(value string) string {
	return strings.TrimRight(strings.TrimSpace(value), "/")
}

func normalizeStorageBackend(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case BackendClickHouse:
		return BackendClickHouse
	case BackendOpenSearch:
		return BackendOpenSearch
	default:
		return BackendFile
	}
}

func normalizeHotBackend(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case BackendOpenSearch:
		return BackendOpenSearch
	case BackendFile:
		return BackendFile
	default:
		return ""
	}
}

func normalizeColdBackend(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case BackendClickHouse:
		return BackendClickHouse
	case BackendOpenSearch:
		return BackendOpenSearch
	case BackendFile:
		return BackendFile
	default:
		return ""
	}
}

func normalizeSecretProvider(value string, vaultEnabled bool) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case SecretProviderVault:
		return SecretProviderVault
	case SecretProviderFile:
		return SecretProviderFile
	default:
		if vaultEnabled {
			return SecretProviderVault
		}
		return SecretProviderVault
	}
}

func legacyHotBackend(backend string) string {
	switch backend {
	case BackendOpenSearch:
		return BackendOpenSearch
	default:
		return BackendFile
	}
}

func legacyColdBackend(backend string) string {
	switch backend {
	case BackendClickHouse:
		return BackendClickHouse
	case BackendOpenSearch:
		return BackendOpenSearch
	default:
		return BackendFile
	}
}

func normalizeIdentifier(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	builder := strings.Builder{}
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '_' || r == '-':
			builder.WriteRune(r)
		}
	}
	return strings.Trim(builder.String(), "_-")
}

func normalizeIdentifierWithSlash(value string) string {
	value = strings.Trim(strings.TrimSpace(strings.ToLower(value)), "/")
	if value == "" {
		return ""
	}
	parts := strings.Split(value, "/")
	cleaned := make([]string, 0, len(parts))
	for _, item := range parts {
		token := normalizeIdentifier(item)
		if token != "" {
			cleaned = append(cleaned, token)
		}
	}
	return strings.Join(cleaned, "/")
}

func MaskSecrets(input Settings) Settings {
	out := Normalize(input)
	if strings.TrimSpace(out.ClickHouse.PasswordEnc) != "" {
		out.ClickHouse.Password = MaskedSecretValue
	}
	if EnabledVault(out) && out.Cold.Backend == BackendClickHouse {
		out.ClickHouse.Password = MaskedSecretValue
	}
	if strings.TrimSpace(out.OpenSearch.PasswordEnc) != "" {
		out.OpenSearch.Password = MaskedSecretValue
	}
	if EnabledVault(out) && (out.Hot.Backend == BackendOpenSearch || out.Cold.Backend == BackendOpenSearch) {
		out.OpenSearch.Password = MaskedSecretValue
	}
	if strings.TrimSpace(out.OpenSearch.APIKeyEnc) != "" {
		out.OpenSearch.APIKey = MaskedSecretValue
	}
	if EnabledVault(out) && (out.Hot.Backend == BackendOpenSearch || out.Cold.Backend == BackendOpenSearch) {
		out.OpenSearch.APIKey = MaskedSecretValue
	}
	if strings.TrimSpace(out.Vault.TokenEnc) != "" {
		out.Vault.Token = MaskedSecretValue
	}
	out.ClickHouse.PasswordEnc = ""
	out.OpenSearch.PasswordEnc = ""
	out.OpenSearch.APIKeyEnc = ""
	out.Vault.TokenEnc = ""
	return out
}

func EnabledClickHouse(input Settings) bool {
	normalized := Normalize(input)
	return normalized.Cold.Backend == BackendClickHouse && normalized.ClickHouse.Endpoint != ""
}

func EnabledOpenSearch(input Settings) bool {
	normalized := Normalize(input)
	return (normalized.Hot.Backend == BackendOpenSearch || normalized.Cold.Backend == BackendOpenSearch) && normalized.OpenSearch.Endpoint != ""
}

func EnabledVault(input Settings) bool {
	normalized := Normalize(input)
	return normalized.SecretProvider == SecretProviderVault && normalized.Vault.Enabled && normalized.Vault.Address != ""
}
