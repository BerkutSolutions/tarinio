package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"waf/internal/loggingconfig"
	"waf/internal/secretcrypto"
	"waf/internal/vaultkv"
)

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
	runtimeSettingsState.mu.RLock()
	security := normalizeRuntimeSecuritySettings(runtimeSettingsState.security)
	runtimeSettingsState.mu.RUnlock()
	return vaultkv.Client{
		Address:       settings.Vault.Address,
		Token:         token,
		Mount:         settings.Vault.Mount,
		PathPrefix:    settings.Vault.PathPrefix,
		TLSSkipVerify: settings.Vault.TLSSkipVerify && security.AllowInsecureVaultTLS,
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
