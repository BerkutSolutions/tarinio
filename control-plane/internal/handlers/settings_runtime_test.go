package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"waf/internal/loggingconfig"
	"waf/internal/secretcrypto"
)

func TestParseLoggingSettingsPreservesMaskedPassword(t *testing.T) {
	pepper := "pepper-for-tests"
	currentSecret, err := secretcrypto.Encrypt("waf:logging:clickhouse", "current-password", pepper)
	if err != nil {
		t.Fatalf("encrypt current secret: %v", err)
	}
	current := loggingconfig.Normalize(loggingconfig.Settings{
		Backend: loggingconfig.BackendClickHouse,
		Cold: loggingconfig.ColdSettings{
			Backend: loggingconfig.BackendClickHouse,
		},
		ClickHouse: loggingconfig.ClickHouseSettings{
			Endpoint:         "http://clickhouse:8123",
			Database:         "waf_logs",
			Table:            "request_logs",
			Username:         "waf",
			PasswordEnc:      currentSecret,
			MigrationEnabled: true,
		},
	})

	next, err := parseLoggingSettings(map[string]any{
		"backend": "clickhouse",
		"cold": map[string]any{
			"backend": "clickhouse",
		},
		"clickhouse": map[string]any{
			"endpoint":          "http://clickhouse:8123/",
			"database":          "waf_logs",
			"table":             "request_logs",
			"username":          "waf",
			"password":          loggingconfig.MaskedSecretValue,
			"migration_enabled": true,
		},
	}, current, pepper)
	if err != nil {
		t.Fatalf("parse logging settings: %v", err)
	}
	if next.ClickHouse.PasswordEnc != currentSecret {
		t.Fatalf("expected encrypted secret to be preserved")
	}
	if next.ClickHouse.Password != "" {
		t.Fatalf("expected plaintext password to be cleared")
	}
	if next.ClickHouse.Endpoint != "http://clickhouse:8123" {
		t.Fatalf("expected endpoint to be normalized, got %q", next.ClickHouse.Endpoint)
	}
}

func TestParseLoggingSettingsEncryptsNewPassword(t *testing.T) {
	pepper := "pepper-for-tests"
	next, err := parseLoggingSettings(map[string]any{
		"backend": "clickhouse",
		"cold": map[string]any{
			"backend": "clickhouse",
		},
		"clickhouse": map[string]any{
			"endpoint":          "http://clickhouse:8123",
			"database":          "waf_logs",
			"table":             "request_logs",
			"username":          "waf",
			"password":          "new-password",
			"migration_enabled": true,
		},
	}, loggingconfig.Settings{}, pepper)
	if err != nil {
		t.Fatalf("parse logging settings: %v", err)
	}
	if next.ClickHouse.Password != "" {
		t.Fatalf("expected plaintext password to be cleared")
	}
	if next.ClickHouse.PasswordEnc == "" {
		t.Fatalf("expected encrypted password to be populated")
	}
	decrypted, err := secretcrypto.Decrypt("waf:logging:clickhouse", next.ClickHouse.PasswordEnc, pepper)
	if err != nil {
		t.Fatalf("decrypt stored secret: %v", err)
	}
	if decrypted != "new-password" {
		t.Fatalf("unexpected decrypted secret: %q", decrypted)
	}
}

func TestParseLoggingSettingsRequiresEndpointForClickHouse(t *testing.T) {
	_, err := parseLoggingSettings(map[string]any{
		"backend": "clickhouse",
		"cold": map[string]any{
			"backend": "clickhouse",
		},
		"clickhouse": map[string]any{
			"database": "waf_logs",
			"table":    "request_logs",
		},
	}, loggingconfig.Settings{}, "pepper-for-tests")
	if err == nil {
		t.Fatalf("expected clickhouse validation error")
	}
}

func TestReconcileLoggingSettingsFromEnvFallsBackToFileWithoutSecret(t *testing.T) {
	t.Setenv("CLICKHOUSE_PASSWORD", "")
	t.Setenv("CLICKHOUSE_USER", "")
	t.Setenv("CLICKHOUSE_DB", "")
	t.Setenv("CLICKHOUSE_ENDPOINT", "")
	runtimeSettingsState.pepper = "pepper-for-tests"

	current := loggingconfig.Normalize(loggingconfig.Settings{
		Backend: loggingconfig.BackendClickHouse,
		Cold: loggingconfig.ColdSettings{
			Backend: loggingconfig.BackendClickHouse,
		},
		ClickHouse: loggingconfig.ClickHouseSettings{
			Endpoint: "http://clickhouse:8123",
			Database: "waf_logs",
			Username: "waf",
			Table:    "request_logs",
		},
	})
	next := reconcileLoggingSettingsFromEnv(current)
	if next.Backend != loggingconfig.BackendFile {
		t.Fatalf("expected backend to fall back to file, got %q", next.Backend)
	}
}

func TestReconcileLoggingSettingsFromEnvEncryptsSecretWhenAvailable(t *testing.T) {
	t.Setenv("CLICKHOUSE_PASSWORD", "clickhouse-secret")
	t.Setenv("CLICKHOUSE_USER", "waf")
	t.Setenv("CLICKHOUSE_DB", "waf_logs")
	t.Setenv("CLICKHOUSE_ENDPOINT", "http://clickhouse:8123")
	runtimeSettingsState.pepper = "pepper-for-tests"

	current := loggingconfig.Normalize(loggingconfig.Settings{
		Backend: loggingconfig.BackendClickHouse,
		Cold: loggingconfig.ColdSettings{
			Backend: loggingconfig.BackendClickHouse,
		},
		ClickHouse: loggingconfig.ClickHouseSettings{
			Endpoint: "http://clickhouse:8123",
			Database: "waf_logs",
			Username: "waf",
			Table:    "request_logs",
		},
	})
	next := reconcileLoggingSettingsFromEnv(current)
	if next.Backend != loggingconfig.BackendClickHouse {
		t.Fatalf("expected clickhouse backend to remain enabled, got %q", next.Backend)
	}
	if next.ClickHouse.PasswordEnc == "" {
		t.Fatalf("expected clickhouse password to be encrypted")
	}
}

func TestParseLoggingSettingsPreservesMaskedOpenSearchSecrets(t *testing.T) {
	pepper := "pepper-for-tests"
	currentPassword, err := secretcrypto.Encrypt("waf:logging:opensearch:password", "current-password", pepper)
	if err != nil {
		t.Fatalf("encrypt current password: %v", err)
	}
	currentAPIKey, err := secretcrypto.Encrypt("waf:logging:opensearch:api_key", "current-api-key", pepper)
	if err != nil {
		t.Fatalf("encrypt current api key: %v", err)
	}
	current := loggingconfig.Normalize(loggingconfig.Settings{
		Hot: loggingconfig.HotSettings{
			Backend: loggingconfig.BackendOpenSearch,
		},
		OpenSearch: loggingconfig.OpenSearchSettings{
			Endpoint:    "http://opensearch:9200",
			IndexPrefix: "waf-hot",
			Username:    "admin",
			PasswordEnc: currentPassword,
			APIKeyEnc:   currentAPIKey,
		},
	})

	next, err := parseLoggingSettings(map[string]any{
		"hot": map[string]any{
			"backend": "opensearch",
		},
		"opensearch": map[string]any{
			"endpoint":     "http://opensearch:9200/",
			"index_prefix": "waf-hot",
			"username":     "admin",
			"password":     loggingconfig.MaskedSecretValue,
			"api_key":      loggingconfig.MaskedSecretValue,
		},
	}, current, pepper)
	if err != nil {
		t.Fatalf("parse logging settings: %v", err)
	}
	if next.OpenSearch.PasswordEnc != currentPassword {
		t.Fatalf("expected encrypted password to be preserved")
	}
	if next.OpenSearch.APIKeyEnc != currentAPIKey {
		t.Fatalf("expected encrypted api key to be preserved")
	}
}

func TestParseLoggingSettingsRequiresVaultAddressWhenEnabled(t *testing.T) {
	_, err := parseLoggingSettings(map[string]any{
		"secret_provider": "vault",
		"vault": map[string]any{
			"enabled": true,
			"token":   "vault-token",
		},
	}, loggingconfig.Settings{}, "pepper-for-tests")
	if err == nil {
		t.Fatalf("expected vault validation error")
	}
}

func TestParseLoggingSettingsStoresSecretsInVault(t *testing.T) {
	var gotPath string
	var gotPayload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		defer r.Body.Close()
		_ = json.NewDecoder(r.Body).Decode(&gotPayload)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{}}`))
	}))
	defer server.Close()

	next, err := parseLoggingSettings(map[string]any{
		"hot": map[string]any{
			"backend": "opensearch",
		},
		"secret_provider": "vault",
		"vault": map[string]any{
			"enabled":     true,
			"address":     server.URL,
			"mount":       "secret",
			"path_prefix": "tarinio",
			"token":       "vault-token",
		},
		"opensearch": map[string]any{
			"endpoint":     "http://opensearch:9200",
			"index_prefix": "waf-hot",
			"username":     "admin",
			"password":     "os-password",
			"api_key":      "os-api-key",
		},
	}, loggingconfig.Settings{}, "pepper-for-tests")
	if err != nil {
		t.Fatalf("parse logging settings: %v", err)
	}
	if gotPath == "" {
		t.Fatalf("expected vault request to be issued")
	}
	if next.OpenSearch.PasswordEnc != "" || next.OpenSearch.APIKeyEnc != "" {
		t.Fatalf("expected local encrypted opensearch secrets to be cleared after vault write")
	}
	if next.SecretProvider != loggingconfig.SecretProviderVault {
		t.Fatalf("expected vault secret provider")
	}
}

func TestParseStorageRetentionCapsHotAndColdIndexWindows(t *testing.T) {
	next, err := parseStorageRetention(map[string]any{
		"hot_index_days":  float64(400),
		"cold_index_days": float64(4000),
	}, StorageRetention{})
	if err != nil {
		t.Fatalf("parse storage retention: %v", err)
	}
	if next.HotIndexDays != loggingconfig.MaxHotDays {
		t.Fatalf("expected hot index days cap %d, got %d", loggingconfig.MaxHotDays, next.HotIndexDays)
	}
	if next.ColdIndexDays != loggingconfig.MaxColdDays {
		t.Fatalf("expected cold index days cap %d, got %d", loggingconfig.MaxColdDays, next.ColdIndexDays)
	}
}

func TestDefaultLoggingSettingsFromEnvReadsVaultTokenFile(t *testing.T) {
	pepper := "pepper-for-tests"
	tokenFile := filepath.Join(t.TempDir(), "vault-token")
	if err := os.WriteFile(tokenFile, []byte("vault-file-token\n"), 0o600); err != nil {
		t.Fatalf("write token file: %v", err)
	}
	t.Setenv("VAULT_ENABLED", "true")
	t.Setenv("VAULT_ADDR", "http://vault:8200")
	t.Setenv("VAULT_MOUNT", "secret")
	t.Setenv("VAULT_PATH_PREFIX", "tarinio")
	t.Setenv("VAULT_TOKEN", "")
	t.Setenv("VAULT_TOKEN_FILE", tokenFile)
	runtimeSettingsState.pepper = pepper

	next := defaultLoggingSettingsFromEnv(loggingconfig.Settings{})
	if next.Vault.TokenEnc == "" {
		t.Fatalf("expected vault token from file to be encrypted into settings")
	}
	decrypted, err := secretcrypto.Decrypt("waf:logging:vault:token", next.Vault.TokenEnc, pepper)
	if err != nil {
		t.Fatalf("decrypt token from file: %v", err)
	}
	if decrypted != "vault-file-token" {
		t.Fatalf("unexpected decrypted token: %q", decrypted)
	}
}
