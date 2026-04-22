package main

import (
	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"waf/internal/loggingconfig"
	"waf/internal/secretcrypto"
)

func TestRequestStreamMigratesOldOpenSearchDataToClickHouse(t *testing.T) {
	pepper := "pepper-for-tests"
	oldDay := time.Now().UTC().AddDate(0, -3, 0).Format("2006-01-02")
	oldStamp := oldDay + "T11:12:13Z"

	clickhouse := newFakeClickHouse()
	clickhouseServer := httptest.NewServer(clickhouse)
	defer clickhouseServer.Close()

	opensearch := newFakeOpenSearch([]requestLogRecord{
		{
			EventHash:    "old-hash",
			Stream:       "runtime",
			IngestedAt:   oldStamp,
			Timestamp:    oldStamp,
			RequestID:    "req-old",
			ClientIP:     "1.1.1.1",
			Method:       "GET",
			URI:          "/three-months-old",
			Status:       200,
			Site:         "legacy",
			Host:         "legacy.example.com",
			UpstreamAddr: "10.0.0.1:80",
		},
	})
	opensearchServer := httptest.NewServer(opensearch)
	defer opensearchServer.Close()

	root := t.TempDir()
	settingsPath := filepath.Join(root, "runtime_settings.json")
	writeRuntimeSettingsFixture(t, settingsPath, pepper, loggingconfig.Settings{
		Backend: loggingconfig.BackendOpenSearch,
		Hot: loggingconfig.HotSettings{
			Backend: loggingconfig.BackendOpenSearch,
		},
		Cold: loggingconfig.ColdSettings{
			Backend: loggingconfig.BackendClickHouse,
		},
		Retention: loggingconfig.RetentionSettings{
			HotDays:  loggingconfig.MaxHotDays,
			ColdDays: loggingconfig.MaxColdDays,
		},
		Routing: loggingconfig.RoutingSettings{
			WriteRequestsToHot: true,
			KeepLocalFallback:  true,
		},
		OpenSearch: loggingconfig.OpenSearchSettings{
			Endpoint:      opensearchServer.URL,
			RequestsIndex: "waf-requests",
		},
		ClickHouse: loggingconfig.ClickHouseSettings{
			Endpoint: clickhouseServer.URL,
			Database: "waf_logs",
			Table:    "request_logs",
		},
	})

	source := newRequestStreamSource(
		filepath.Join(root, "missing-access.log"),
		100,
		filepath.Join(root, "archive"),
		loggingconfig.MaxHotDays,
		withRequestOpenSearch(settingsPath, pepper),
		withRequestClickHouse(settingsPath, pepper),
	)

	if err := source.probe(url.Values{}); err != nil {
		t.Fatalf("probe failed: %v", err)
	}

	if got := clickhouse.recordCount(); got != 1 {
		t.Fatalf("expected migrated row in clickhouse, got %d", got)
	}
	if got := opensearch.recordCount(); got != 0 {
		t.Fatalf("expected old row to be deleted from opensearch after migration, got %d", got)
	}

	query := url.Values{}
	query.Set("day", oldDay)
	query.Set("retention_days", "14")
	items, err := source.latest(query)
	if err != nil {
		t.Fatalf("latest failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one historical row from clickhouse, got %d", len(items))
	}
	entry, _ := items[0]["entry"].(map[string]any)
	if got := asString(entry["request_id"]); got != "req-old" {
		t.Fatalf("unexpected request id after migration: %s", got)
	}
}

func TestRequestBackendsResolveSecretsFromVault(t *testing.T) {
	pepper := "pepper-for-tests"
	var (
		mu      sync.Mutex
		secrets = map[string]map[string]string{
			"logging/clickhouse": {"password": "ch-secret"},
			"logging/opensearch": {"password": "os-secret", "api_key": "os-api"},
		}
	)
	vaultServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		name := strings.TrimPrefix(r.URL.Path, "/v1/secret/data/tarinio/")
		switch r.Method {
		case http.MethodGet:
			payload := map[string]any{
				"data": map[string]any{
					"data": secrets[name],
				},
			}
			_ = json.NewEncoder(w).Encode(payload)
		case http.MethodPost:
			var body struct {
				Data map[string]string `json:"data"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			secrets[name] = body.Data
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":{}}`))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer vaultServer.Close()

	root := t.TempDir()
	settingsPath := filepath.Join(root, "runtime_settings.json")
	tokenEnc, err := secretcrypto.Encrypt("waf:logging:vault:token", "vault-token", pepper)
	if err != nil {
		t.Fatalf("encrypt vault token: %v", err)
	}
	writeRuntimeSettingsFixture(t, settingsPath, pepper, loggingconfig.Settings{
		Backend: loggingconfig.BackendOpenSearch,
		Hot: loggingconfig.HotSettings{Backend: loggingconfig.BackendOpenSearch},
		Cold: loggingconfig.ColdSettings{Backend: loggingconfig.BackendClickHouse},
		SecretProvider: loggingconfig.SecretProviderVault,
		Vault: loggingconfig.VaultSettings{
			Enabled:    true,
			Address:    vaultServer.URL,
			TokenEnc:   tokenEnc,
			Mount:      "secret",
			PathPrefix: "tarinio",
		},
		OpenSearch: loggingconfig.OpenSearchSettings{
			Endpoint:      "http://opensearch:9200",
			Username:      "admin",
			RequestsIndex: "waf-requests",
		},
		ClickHouse: loggingconfig.ClickHouseSettings{
			Endpoint: "http://clickhouse:8123",
			Database: "waf_logs",
			Table:    "request_logs",
			Username: "waf",
		},
	})

	clickhouseStore := newRequestClickHouseStore(settingsPath, pepper)
	clickhouseCfg, err := clickhouseStore.currentConfig()
	if err != nil {
		t.Fatalf("clickhouse config: %v", err)
	}
	if clickhouseCfg.Password != "ch-secret" {
		t.Fatalf("unexpected clickhouse password from vault: %q", clickhouseCfg.Password)
	}

	opensearchStore := newRequestOpenSearchStore(settingsPath, pepper)
	opensearchCfg, err := opensearchStore.currentConfig()
	if err != nil {
		t.Fatalf("opensearch config: %v", err)
	}
	if opensearchCfg.Password != "os-secret" || opensearchCfg.APIKey != "os-api" {
		t.Fatalf("unexpected opensearch secrets from vault: password=%q api=%q", opensearchCfg.Password, opensearchCfg.APIKey)
	}
}

func writeRuntimeSettingsFixture(t *testing.T, path string, _ string, logging loggingconfig.Settings) {
	t.Helper()
	logging = loggingconfig.Normalize(logging)
	payload := map[string]any{
		"logging": logging,
	}
	content, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal settings fixture: %v", err)
	}
	if err := os.WriteFile(path, append(content, '\n'), 0o644); err != nil {
		t.Fatalf("write settings fixture: %v", err)
	}
}

type fakeClickHouse struct {
	mu      sync.Mutex
	records []requestLogRecord
}

func newFakeClickHouse() *fakeClickHouse {
	return &fakeClickHouse{records: make([]requestLogRecord, 0)}
}

func (f *fakeClickHouse) recordCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.records)
}

func (f *fakeClickHouse) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	switch {
	case strings.HasPrefix(query, "CREATE DATABASE"), strings.HasPrefix(query, "CREATE TABLE"), strings.HasPrefix(query, "ALTER TABLE"):
		if strings.Contains(query, "DELETE WHERE") {
			day := extractSingleQuotedValue(query)
			f.deleteDay(day)
		}
		w.WriteHeader(http.StatusOK)
	case strings.HasPrefix(query, "INSERT INTO"):
		f.insertBody(r.Body)
		w.WriteHeader(http.StatusOK)
	case strings.HasPrefix(query, "SELECT count() AS total"):
		total := f.uniqueDayCount()
		_, _ = w.Write([]byte(`{"total":` + strconvString(total) + `}` + "\n"))
	case strings.Contains(query, "GROUP BY date ORDER BY date DESC"):
		for _, line := range f.groupedDaysJSON() {
			_, _ = w.Write([]byte(line + "\n"))
		}
	case strings.HasPrefix(query, "SELECT event_hash"):
		for _, line := range f.recordsJSON(query) {
			_, _ = w.Write([]byte(line + "\n"))
		}
	default:
		w.WriteHeader(http.StatusOK)
	}
}

func (f *fakeClickHouse) insertBody(body io.ReadCloser) {
	defer body.Close()
	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var record requestLogRecord
		if err := json.Unmarshal([]byte(line), &record); err == nil {
			f.mu.Lock()
			f.records = append(f.records, record)
			f.mu.Unlock()
		}
	}
}

func (f *fakeClickHouse) deleteDay(day string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	filtered := f.records[:0]
	for _, record := range f.records {
		if !strings.HasPrefix(record.Timestamp, day) {
			filtered = append(filtered, record)
		}
	}
	f.records = append([]requestLogRecord{}, filtered...)
}

func (f *fakeClickHouse) uniqueDayCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	seen := map[string]struct{}{}
	for _, record := range f.records {
		if len(record.Timestamp) >= 10 {
			seen[record.Timestamp[:10]] = struct{}{}
		}
	}
	return len(seen)
}

func (f *fakeClickHouse) groupedDaysJSON() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	type item struct {
		Date      string `json:"date"`
		Lines     int    `json:"lines"`
		UpdatedAt string `json:"updated_at"`
	}
	byDay := map[string]*item{}
	for _, record := range f.records {
		day := record.Timestamp[:10]
		current := byDay[day]
		if current == nil {
			current = &item{Date: day, UpdatedAt: record.IngestedAt}
			byDay[day] = current
		}
		current.Lines++
		if record.IngestedAt > current.UpdatedAt {
			current.UpdatedAt = record.IngestedAt
		}
	}
	keys := make([]string, 0, len(byDay))
	for day := range byDay {
		keys = append(keys, day)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))
	out := make([]string, 0, len(keys))
	for _, day := range keys {
		line, _ := json.Marshal(byDay[day])
		out = append(out, string(line))
	}
	return out
}

func (f *fakeClickHouse) recordsJSON(query string) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	day := extractSingleQuotedValue(query)
	out := make([]requestLogRecord, 0, len(f.records))
	for _, record := range f.records {
		if day != "" && !strings.HasPrefix(record.Timestamp, day) {
			continue
		}
		out = append(out, record)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Timestamp > out[j].Timestamp })
	lines := make([]string, 0, len(out))
	for _, record := range out {
		line, _ := json.Marshal(record)
		lines = append(lines, string(line))
	}
	return lines
}

type fakeOpenSearch struct {
	mu      sync.Mutex
	records []requestLogRecord
}

func newFakeOpenSearch(records []requestLogRecord) *fakeOpenSearch {
	return &fakeOpenSearch{records: append([]requestLogRecord{}, records...)}
}

func (f *fakeOpenSearch) recordCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.records)
}

func (f *fakeOpenSearch) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodPut:
		w.WriteHeader(http.StatusOK)
	case strings.HasSuffix(r.URL.Path, "/_search"):
		f.handleSearch(w, r)
	case strings.HasSuffix(r.URL.Path, "/_delete_by_query"):
		f.handleDeleteByQuery(w, r)
	case strings.HasPrefix(r.URL.Path, "/_bulk"):
		f.handleBulk(w, r)
	default:
		w.WriteHeader(http.StatusOK)
	}
}

func (f *fakeOpenSearch) handleBulk(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	scanner := bufio.NewScanner(r.Body)
	lines := make([]string, 0)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	for i := 1; i < len(lines); i += 2 {
		var record requestLogRecord
		if err := json.Unmarshal([]byte(lines[i]), &record); err == nil {
			f.mu.Lock()
			f.records = append(f.records, record)
			f.mu.Unlock()
		}
	}
	_, _ = w.Write([]byte(`{"errors":false}`))
}

func (f *fakeOpenSearch) handleDeleteByQuery(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)
	day := rangeDayFromBody(body)
	f.mu.Lock()
	filtered := f.records[:0]
	for _, record := range f.records {
		if day != "" && strings.HasPrefix(record.Timestamp, day) {
			continue
		}
		filtered = append(filtered, record)
	}
	f.records = append([]requestLogRecord{}, filtered...)
	f.mu.Unlock()
	_, _ = w.Write([]byte(`{"deleted":1}`))
}

func (f *fakeOpenSearch) handleSearch(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)

	if _, hasAggs := body["aggs"]; hasAggs {
		type bucket struct {
			KeyAsString string `json:"key_as_string"`
			DocCount    int    `json:"doc_count"`
		}
		byDay := map[string]int{}
		f.mu.Lock()
		for _, record := range f.records {
			byDay[record.Timestamp[:10]]++
		}
		f.mu.Unlock()
		keys := make([]string, 0, len(byDay))
		for day := range byDay {
			keys = append(keys, day)
		}
		sort.Sort(sort.Reverse(sort.StringSlice(keys)))
		buckets := make([]bucket, 0, len(keys))
		for _, day := range keys {
			buckets = append(buckets, bucket{
				KeyAsString: day + "T00:00:00.000Z",
				DocCount:    byDay[day],
			})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"hits": map[string]any{
				"total": map[string]any{"value": len(f.records)},
			},
			"aggregations": map[string]any{
				"by_day": map[string]any{"buckets": buckets},
			},
		})
		return
	}

	day := rangeDayFromBody(body)
	f.mu.Lock()
	records := make([]requestLogRecord, 0, len(f.records))
	for _, record := range f.records {
		if day != "" && !strings.HasPrefix(record.Timestamp, day) {
			continue
		}
		records = append(records, record)
	}
	f.mu.Unlock()
	sort.Slice(records, func(i, j int) bool { return records[i].Timestamp > records[j].Timestamp })
	hits := make([]map[string]any, 0, len(records))
	for _, record := range records {
		hits = append(hits, map[string]any{"_source": record})
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"hits": map[string]any{
			"hits": hits,
		},
	})
}

func rangeDayFromBody(body map[string]any) string {
	query, _ := body["query"].(map[string]any)
	if query == nil {
		return ""
	}
	if boolQuery, _ := query["bool"].(map[string]any); boolQuery != nil {
		if filters, _ := boolQuery["filter"].([]any); filters != nil {
			for _, item := range filters {
				if typed, _ := item.(map[string]any); typed != nil {
					if day := rangeDayFromBody(map[string]any{"query": typed}); day != "" {
						return day
					}
				}
			}
		}
	}
	rangeNode, _ := query["range"].(map[string]any)
	if rangeNode == nil {
		return ""
	}
	timestampNode, _ := rangeNode["timestamp"].(map[string]any)
	if timestampNode == nil {
		return ""
	}
	gte := strings.TrimSpace(asString(timestampNode["gte"]))
	if len(gte) >= 10 {
		return gte[:10]
	}
	return ""
}

func extractSingleQuotedValue(query string) string {
	re := regexp.MustCompile(`'([^']+)'`)
	matches := re.FindStringSubmatch(query)
	if len(matches) == 2 {
		return matches[1]
	}
	return ""
}

func strconvString(value int) string { return strconv.Itoa(value) }
