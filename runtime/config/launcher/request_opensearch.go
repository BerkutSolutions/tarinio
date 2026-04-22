package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"waf/internal/loggingconfig"
	"waf/internal/secretcrypto"
	"waf/internal/vaultkv"
)

type requestOpenSearchConfig struct {
	Enabled       bool
	Endpoint      string
	Username      string
	Password      string
	APIKey        string
	RequestsIndex string
}

type requestOpenSearchStore struct {
	settingsPath string
	pepper       string
	client       *http.Client

	mu             sync.Mutex
	lastMappingKey string
}

var errOpenSearchDisabled = errors.New("opensearch logging backend is disabled")

func newRequestOpenSearchStore(settingsPath string, pepper string) *requestOpenSearchStore {
	return &requestOpenSearchStore{
		settingsPath: strings.TrimSpace(settingsPath),
		pepper:       strings.TrimSpace(pepper),
		client:       &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *requestOpenSearchStore) currentConfig() (requestOpenSearchConfig, error) {
	if s == nil || strings.TrimSpace(s.settingsPath) == "" {
		return requestOpenSearchConfig{}, nil
	}
	content, err := os.ReadFile(s.settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return requestOpenSearchConfig{}, nil
		}
		return requestOpenSearchConfig{}, err
	}
	var payload struct {
		Logging loggingconfig.Settings `json:"logging"`
	}
	if err := json.Unmarshal(content, &payload); err != nil {
		return requestOpenSearchConfig{}, err
	}
	logging := loggingconfig.Normalize(payload.Logging)
	if !loggingconfig.EnabledOpenSearch(logging) {
		return requestOpenSearchConfig{}, nil
	}
	password := ""
	apiKey := ""
	if loggingconfig.EnabledVault(logging) {
		client := vaultkv.Client{
			Address:       logging.Vault.Address,
			Token:         mustResolveVaultToken(logging.Vault.TokenEnc, s.pepper),
			Mount:         logging.Vault.Mount,
			PathPrefix:    logging.Vault.PathPrefix,
			TLSSkipVerify: logging.Vault.TLSSkipVerify,
		}
		password, err = client.Get("logging/opensearch", "password")
		if err != nil {
			return requestOpenSearchConfig{}, err
		}
		apiKey, err = client.Get("logging/opensearch", "api_key")
		if err != nil {
			return requestOpenSearchConfig{}, err
		}
	} else {
		password, err = secretcrypto.Decrypt("waf:logging:opensearch:password", logging.OpenSearch.PasswordEnc, s.pepper)
		if err != nil {
			return requestOpenSearchConfig{}, err
		}
		apiKey, err = secretcrypto.Decrypt("waf:logging:opensearch:api_key", logging.OpenSearch.APIKeyEnc, s.pepper)
		if err != nil {
			return requestOpenSearchConfig{}, err
		}
	}
	return requestOpenSearchConfig{
		Enabled:       true,
		Endpoint:      logging.OpenSearch.Endpoint,
		Username:      logging.OpenSearch.Username,
		Password:      password,
		APIKey:        apiKey,
		RequestsIndex: logging.OpenSearch.RequestsIndex,
	}, nil
}

func (s *requestOpenSearchStore) ensureIndex(cfg requestOpenSearchConfig) error {
	key := cfg.Endpoint + "|" + cfg.RequestsIndex + "|" + cfg.Username
	s.mu.Lock()
	if s.lastMappingKey == key {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()
	body := map[string]any{
		"mappings": map[string]any{
			"properties": map[string]any{
				"event_hash":    map[string]any{"type": "keyword"},
				"stream":        map[string]any{"type": "keyword"},
				"ingested_at":   map[string]any{"type": "date"},
				"timestamp":     map[string]any{"type": "date"},
				"request_id":    map[string]any{"type": "keyword"},
				"client_ip":     map[string]any{"type": "ip", "ignore_malformed": true},
				"country":       map[string]any{"type": "keyword"},
				"method":        map[string]any{"type": "keyword"},
				"uri":           map[string]any{"type": "wildcard"},
				"status":        map[string]any{"type": "integer"},
				"site":          map[string]any{"type": "keyword"},
				"host":          map[string]any{"type": "keyword"},
				"upstream_addr": map[string]any{"type": "keyword"},
				"referer":       map[string]any{"type": "wildcard"},
				"user_agent":    map[string]any{"type": "wildcard"},
			},
		},
	}
	if err := s.doRequest(cfg, http.MethodPut, "/"+cfg.RequestsIndex, body, nil); err != nil && !strings.Contains(err.Error(), "resource_already_exists_exception") {
		return err
	}
	s.mu.Lock()
	s.lastMappingKey = key
	s.mu.Unlock()
	return nil
}

func (s *requestOpenSearchStore) insert(records []requestLogRecord) error {
	if s == nil || len(records) == 0 {
		return nil
	}
	cfg, err := s.currentConfig()
	if err != nil || !cfg.Enabled {
		return err
	}
	if err := s.ensureIndex(cfg); err != nil {
		return err
	}
	var body bytes.Buffer
	enc := json.NewEncoder(&body)
	for _, record := range records {
		meta := map[string]any{"index": map[string]any{"_index": cfg.RequestsIndex, "_id": record.EventHash}}
		if err := enc.Encode(meta); err != nil {
			return err
		}
		if err := enc.Encode(record); err != nil {
			return err
		}
	}
	return s.doRawRequest(cfg, http.MethodPost, "/_bulk?refresh=false", &body, func(resp *http.Response) error {
		var payload struct {
			Errors bool `json:"errors"`
		}
		if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&payload); err != nil {
			return err
		}
		if payload.Errors {
			return fmt.Errorf("opensearch bulk insert reported errors")
		}
		return nil
	})
}

func (s *requestOpenSearchStore) latest(options requestQueryOptions) ([]map[string]any, error) {
	cfg, err := s.currentConfig()
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled {
		return nil, errOpenSearchDisabled
	}
	if err := s.ensureIndex(cfg); err != nil {
		return nil, err
	}
	limit := options.Limit
	if limit <= 0 {
		limit = 500
	}
	query := map[string]any{
		"size": limit,
		"from": maxInt(options.Offset, 0),
		"sort": []map[string]any{
			{"timestamp": map[string]any{"order": "desc"}},
		},
		"query": buildOpenSearchRequestQuery(options),
	}
	var payload struct {
		Hits struct {
			Hits []struct {
				Source requestLogRecord `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := s.doRequest(cfg, http.MethodPost, "/"+cfg.RequestsIndex+"/_search", query, &payload); err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(payload.Hits.Hits))
	for _, hit := range payload.Hits.Hits {
		out = append(out, requestRecordToMap(hit.Source))
	}
	return out, nil
}

func (s *requestOpenSearchStore) indexes(options requestQueryOptions, archiveRoot string) (map[string]any, error) {
	cfg, err := s.currentConfig()
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled {
		return nil, errOpenSearchDisabled
	}
	if err := s.ensureIndex(cfg); err != nil {
		return nil, err
	}
	query := map[string]any{
		"size":  0,
		"query": buildOpenSearchRequestQuery(options),
		"aggs": map[string]any{
			"by_day": map[string]any{
				"date_histogram": map[string]any{
					"field":             "timestamp",
					"calendar_interval": "day",
					"order":             map[string]any{"_key": "desc"},
				},
			},
		},
	}
	var payload struct {
		Hits struct {
			Total struct {
				Value int `json:"value"`
			} `json:"total"`
		} `json:"hits"`
		Aggregations struct {
			ByDay struct {
				Buckets []struct {
					KeyAsString string `json:"key_as_string"`
					DocCount    int    `json:"doc_count"`
				} `json:"buckets"`
			} `json:"by_day"`
		} `json:"aggregations"`
	}
	if err := s.doRequest(cfg, http.MethodPost, "/"+cfg.RequestsIndex+"/_search", query, &payload); err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(payload.Aggregations.ByDay.Buckets))
	for _, bucket := range payload.Aggregations.ByDay.Buckets {
		date := bucket.KeyAsString
		if len(date) >= 10 {
			date = date[:10]
		}
		items = append(items, map[string]any{
			"date":         date,
			"file_name":    fmt.Sprintf("opensearch:%s", cfg.RequestsIndex),
			"lines":        bucket.DocCount,
			"size_bytes":   0,
			"updated_at":   bucket.KeyAsString,
			"storage_type": "opensearch",
		})
	}
	state, _ := loadRequestMigrationState(filepath.Join(archiveRoot, requestOpenSearchMigrationStateFile))
	return map[string]any{
		"items":             items,
		"total":             len(items),
		"limit":             options.Limit,
		"offset":            options.Offset,
		"archive_root":      archiveRoot,
		"storage_type":      "opensearch",
		"migration_state":   state,
		"opensearch_target": fmt.Sprintf("%s/%s", cfg.Endpoint, cfg.RequestsIndex),
	}, nil
}

func (s *requestOpenSearchStore) days() ([]string, error) {
	cfg, err := s.currentConfig()
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled {
		return nil, errOpenSearchDisabled
	}
	query := map[string]any{
		"size": 0,
		"aggs": map[string]any{
			"by_day": map[string]any{
				"date_histogram": map[string]any{
					"field":             "timestamp",
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
				} `json:"buckets"`
			} `json:"by_day"`
		} `json:"aggregations"`
	}
	if err := s.doRequest(cfg, http.MethodPost, "/"+cfg.RequestsIndex+"/_search", query, &payload); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(payload.Aggregations.ByDay.Buckets))
	for _, bucket := range payload.Aggregations.ByDay.Buckets {
		day := bucket.KeyAsString
		if len(day) >= 10 {
			day = day[:10]
		}
		if strings.TrimSpace(day) != "" {
			out = append(out, day)
		}
	}
	return out, nil
}

func (s *requestOpenSearchStore) exportDay(day string) ([]requestLogRecord, error) {
	cfg, err := s.currentConfig()
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled {
		return nil, errOpenSearchDisabled
	}
	query := map[string]any{
		"size": 10000,
		"sort": []map[string]any{
			{"timestamp": map[string]any{"order": "asc"}},
		},
		"query": map[string]any{
			"range": map[string]any{
				"timestamp": map[string]any{
					"gte": day + "T00:00:00Z",
					"lt":  day + "T23:59:59Z",
				},
			},
		},
	}
	var payload struct {
		Hits struct {
			Hits []struct {
				Source requestLogRecord `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := s.doRequest(cfg, http.MethodPost, "/"+cfg.RequestsIndex+"/_search", query, &payload); err != nil {
		return nil, err
	}
	out := make([]requestLogRecord, 0, len(payload.Hits.Hits))
	for _, hit := range payload.Hits.Hits {
		out = append(out, hit.Source)
	}
	return out, nil
}

func (s *requestOpenSearchStore) containsDayRecords(day string, expected []requestLogRecord) error {
	if s == nil || len(expected) == 0 {
		return nil
	}
	records, err := s.exportDay(day)
	if err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(records))
	for _, record := range records {
		hash := strings.TrimSpace(record.EventHash)
		if hash == "" {
			continue
		}
		seen[hash] = struct{}{}
	}
	for _, record := range expected {
		hash := strings.TrimSpace(record.EventHash)
		if hash == "" {
			continue
		}
		if _, ok := seen[hash]; !ok {
			return errors.New("opensearch migration validation failed: missing migrated records")
		}
	}
	return nil
}

func (s *requestOpenSearchStore) deleteDay(day string) error {
	cfg, err := s.currentConfig()
	if err != nil || !cfg.Enabled {
		return err
	}
	query := map[string]any{
		"query": map[string]any{
			"range": map[string]any{
				"timestamp": map[string]any{
					"gte": day + "T00:00:00Z",
					"lt":  day + "T23:59:59Z",
				},
			},
		},
	}
	return s.doRequest(cfg, http.MethodPost, "/"+cfg.RequestsIndex+"/_delete_by_query", query, nil)
}

func (s *requestOpenSearchStore) doRequest(cfg requestOpenSearchConfig, method string, path string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(payload)
	}
	return s.doRawRequest(cfg, method, path, reader, func(resp *http.Response) error {
		if out == nil {
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
			return nil
		}
		return json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(out)
	})
}

func (s *requestOpenSearchStore) doRawRequest(cfg requestOpenSearchConfig, method string, path string, body io.Reader, handle func(*http.Response) error) error {
	req, err := http.NewRequest(method, strings.TrimRight(cfg.Endpoint, "/")+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(cfg.APIKey) != "" {
		req.Header.Set("Authorization", "ApiKey "+strings.TrimSpace(cfg.APIKey))
	} else if strings.TrimSpace(cfg.Username) != "" {
		req.SetBasicAuth(cfg.Username, cfg.Password)
	}
	client := s.client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		content, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		return fmt.Errorf("opensearch request failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(content)))
	}
	if handle != nil {
		return handle(resp)
	}
	return nil
}

func buildOpenSearchRequestQuery(options requestQueryOptions) map[string]any {
	filters := make([]map[string]any, 0, 3)
	if options.RetentionDays > 0 && options.Day == "" && options.Since.IsZero() {
		filters = append(filters, map[string]any{
			"range": map[string]any{
				"timestamp": map[string]any{
					"gte": fmt.Sprintf("now-%dd/d", options.RetentionDays),
				},
			},
		})
	}
	if !options.Since.IsZero() {
		filters = append(filters, map[string]any{
			"range": map[string]any{
				"timestamp": map[string]any{
					"gte": options.Since.UTC().Format(time.RFC3339Nano),
				},
			},
		})
	}
	if strings.TrimSpace(options.Day) != "" {
		filters = append(filters, map[string]any{
			"range": map[string]any{
				"timestamp": map[string]any{
					"gte": options.Day + "T00:00:00Z",
					"lt":  options.Day + "T23:59:59Z",
				},
			},
		})
	}
	if len(filters) == 0 {
		return map[string]any{"match_all": map[string]any{}}
	}
	return map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}
}
