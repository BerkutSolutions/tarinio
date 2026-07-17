package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"waf/internal/loggingconfig"
	"waf/internal/secretcrypto"
	"waf/internal/vaultkv"
)

func parseLoggingSettings(raw map[string]any, current loggingconfig.Settings, pepper string, allowInsecureVaultTLS bool) (loggingconfig.Settings, error) {
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
	if strings.TrimSpace(current.Vault.Address) != "" &&
		strings.TrimSpace(current.Vault.Address) != strings.TrimSpace(incoming.Vault.Address) &&
		strings.TrimSpace(incoming.Vault.Token) == loggingconfig.MaskedSecretValue {
		return loggingconfig.Settings{}, fmt.Errorf("vault token must be supplied when changing vault address")
	}

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
	if incoming.Cold.Backend == loggingconfig.BackendClickHouse {
		endpoint, err := loggingconfig.ValidateClickHouseEndpoint(incoming.ClickHouse.Endpoint, loggingconfig.AllowedClickHouseEndpointsFromEnv())
		if err != nil {
			return loggingconfig.Settings{}, err
		}
		incoming.ClickHouse.Endpoint = endpoint
	}
	if (incoming.Hot.Backend == loggingconfig.BackendOpenSearch || incoming.Cold.Backend == loggingconfig.BackendOpenSearch) && strings.TrimSpace(incoming.OpenSearch.Endpoint) == "" {
		return loggingconfig.Settings{}, fmt.Errorf("opensearch endpoint is required when opensearch storage is enabled")
	}
	if incoming.SecretProvider == loggingconfig.SecretProviderVault && incoming.Vault.Enabled && strings.TrimSpace(incoming.Vault.Address) == "" {
		return loggingconfig.Settings{}, fmt.Errorf("vault address is required when vault secret provider is enabled")
	}
	if incoming.Vault.TLSSkipVerify && !allowInsecureVaultTLS {
		return loggingconfig.Settings{}, fmt.Errorf("vault tls_skip_verify is disabled by security policy")
	}
	if incoming.SecretProvider == loggingconfig.SecretProviderVault && incoming.Vault.Enabled {
		if err := storeLoggingSecretsInVault(incoming); err != nil {
			return loggingconfig.Settings{}, err
		}
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
