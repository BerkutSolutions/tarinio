package handlers

import (
	"os"
	"strings"

	"waf/internal/loggingconfig"
	"waf/internal/secretcrypto"
)

func defaultLoggingSettingsFromEnv(current loggingconfig.Settings, security RuntimeSecuritySettings) loggingconfig.Settings {
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
		current.Vault.TLSSkipVerify = security.AllowInsecureVaultTLS && parseEnvBool(os.Getenv("VAULT_TLS_SKIP_VERIFY"))
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
	if !security.AllowInsecureVaultTLS {
		current.Vault.TLSSkipVerify = false
	}
	return loggingconfig.Normalize(current)
}

func reconcileLoggingSettingsFromEnv(current loggingconfig.Settings, security RuntimeSecuritySettings) loggingconfig.Settings {
	current = loggingconfig.Normalize(current)
	clickhousePassword := strings.TrimSpace(os.Getenv("CLICKHOUSE_PASSWORD"))
	opensearchPassword := strings.TrimSpace(os.Getenv("OPENSEARCH_PASSWORD"))
	opensearchAPIKey := strings.TrimSpace(os.Getenv("OPENSEARCH_API_KEY"))
	vaultToken := envSecretValue("VAULT_TOKEN", "VAULT_TOKEN_FILE")
	opensearchConfigured := opensearchPassword != "" || opensearchAPIKey != "" || strings.TrimSpace(current.OpenSearch.PasswordEnc) != "" || strings.TrimSpace(current.OpenSearch.APIKeyEnc) != ""

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
	if current.Hot.Backend == loggingconfig.BackendOpenSearch {
		if !opensearchConfigured {
			current.Hot.Backend = loggingconfig.BackendFile
		} else {
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
		}
	}
	if current.Cold.Backend == loggingconfig.BackendOpenSearch && !opensearchConfigured {
		current.Cold.Backend = loggingconfig.BackendFile
	}
	if current.Cold.Backend == loggingconfig.BackendOpenSearch && opensearchConfigured {
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
	current.Vault.TLSSkipVerify = security.AllowInsecureVaultTLS && (current.Vault.TLSSkipVerify || parseEnvBool(os.Getenv("VAULT_TLS_SKIP_VERIFY")))
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
	if !security.AllowInsecureVaultTLS {
		current.Vault.TLSSkipVerify = false
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
