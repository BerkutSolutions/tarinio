package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
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

	"waf/internal/loggingconfig"
	"waf/internal/secretcrypto"
	"waf/internal/vaultkv"
)

type requestLogRecord struct {
	EventHash    string `json:"event_hash"`
	Stream       string `json:"stream"`
	IngestedAt   string `json:"ingested_at"`
	Timestamp    string `json:"timestamp"`
	RequestID    string `json:"request_id"`
	ClientIP     string `json:"client_ip"`
	Country      string `json:"country"`
	City         string `json:"city"`
	Method       string `json:"method"`
	URI          string `json:"uri"`
	Status       int    `json:"status"`
	Site         string `json:"site"`
	Host         string `json:"host"`
	UpstreamAddr string `json:"upstream_addr"`
	Referer      string `json:"referer"`
	UserAgent    string `json:"user_agent"`
}

type requestClickHouseConfig struct {
	Enabled          bool
	Endpoint         string
	Database         string
	Username         string
	Password         string
	Table            string
	MigrationEnabled bool
}

type requestClickHouseStore struct {
	settingsPath string
	pepper       string
	client       *http.Client

	mu            sync.Mutex
	lastSchemaKey string
}

type requestMigrationState struct {
	ImportedDays map[string]string `json:"imported_days,omitempty"`
}

var errClickHouseDisabled = errors.New("clickhouse logging backend is disabled")

func newRequestClickHouseStore(settingsPath string, pepper string) *requestClickHouseStore {
	return &requestClickHouseStore{
		settingsPath: strings.TrimSpace(settingsPath),
		pepper:       strings.TrimSpace(pepper),
		client:       &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *requestClickHouseStore) currentConfig() (requestClickHouseConfig, error) {
	if s == nil || strings.TrimSpace(s.settingsPath) == "" {
		return requestClickHouseConfig{}, nil
	}
	content, err := os.ReadFile(s.settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return requestClickHouseConfig{}, nil
		}
		return requestClickHouseConfig{}, err
	}
	var payload struct {
		Logging loggingconfig.Settings `json:"logging"`
	}
	if err := json.Unmarshal(content, &payload); err != nil {
		return requestClickHouseConfig{}, err
	}
	logging := loggingconfig.Normalize(payload.Logging)
	if !loggingconfig.EnabledClickHouse(logging) {
		return requestClickHouseConfig{}, nil
	}
	password := ""
	if loggingconfig.EnabledVault(logging) {
		client := vaultkv.Client{
			Address:       logging.Vault.Address,
			Token:         mustResolveVaultToken(logging.Vault.TokenEnc, s.pepper),
			Mount:         logging.Vault.Mount,
			PathPrefix:    logging.Vault.PathPrefix,
			TLSSkipVerify: logging.Vault.TLSSkipVerify,
		}
		password, err = client.Get("logging/clickhouse", "password")
		if err != nil {
			return requestClickHouseConfig{}, err
		}
	} else {
		password, err = secretcrypto.Decrypt("waf:logging:clickhouse", logging.ClickHouse.PasswordEnc, s.pepper)
		if err != nil {
			return requestClickHouseConfig{}, err
		}
	}
	return requestClickHouseConfig{
		Enabled:          true,
		Endpoint:         logging.ClickHouse.Endpoint,
		Database:         logging.ClickHouse.Database,
		Username:         logging.ClickHouse.Username,
		Password:         password,
		Table:            logging.ClickHouse.Table,
		MigrationEnabled: logging.ClickHouse.MigrationEnabled,
	}, nil
}

func mustResolveVaultToken(valueEnc string, pepper string) string {
	token, err := secretcrypto.Decrypt("waf:logging:vault:token", valueEnc, pepper)
	if err != nil {
		return ""
	}
	return token
}

func (s *requestClickHouseStore) insert(records []requestLogRecord) error {
	if s == nil || len(records) == 0 {
		return nil
	}
	cfg, err := s.currentConfig()
	if err != nil || !cfg.Enabled {
		return err
	}
	if err := s.ensureSchema(cfg); err != nil {
		return err
	}
	body := bytes.Buffer{}
	encoder := json.NewEncoder(&body)
	for _, record := range records {
		if err := encoder.Encode(record); err != nil {
			return err
		}
	}
	return s.doQuery(cfg, fmt.Sprintf("INSERT INTO %s.%s FORMAT JSONEachRow", cfg.Database, cfg.Table), &body)
}

func (s *requestClickHouseStore) latest(options requestQueryOptions) ([]map[string]any, error) {
	cfg, err := s.currentConfig()
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled {
		return nil, errClickHouseDisabled
	}
	if err := s.ensureSchema(cfg); err != nil {
		return nil, err
	}
	limit := options.Limit
	if limit <= 0 {
		limit = 500
	}
	query := strings.Builder{}
	query.WriteString("SELECT event_hash, stream, ingested_at, timestamp, request_id, client_ip, country, city, method, uri, status, site, host, upstream_addr, referer, user_agent ")
	query.WriteString(fmt.Sprintf("FROM %s.%s FINAL WHERE 1=1", cfg.Database, cfg.Table))
	if options.RetentionDays > 0 && options.Day == "" && options.Since.IsZero() {
		query.WriteString(fmt.Sprintf(" AND timestamp >= now64(9) - INTERVAL %d DAY", options.RetentionDays))
	}
	if !options.Since.IsZero() {
		query.WriteString(fmt.Sprintf(" AND timestamp >= parseDateTime64BestEffort('%s', 9, 'UTC')", escapeSQLString(options.Since.UTC().Format(time.RFC3339Nano))))
	}
	if strings.TrimSpace(options.Day) != "" {
		query.WriteString(fmt.Sprintf(" AND toDate(timestamp) = toDate('%s')", escapeSQLString(options.Day)))
	}
	query.WriteString(" ORDER BY timestamp DESC")
	query.WriteString(fmt.Sprintf(" LIMIT %d OFFSET %d FORMAT JSONEachRow", limit, maxInt(options.Offset, 0)))

	resp, err := s.query(cfg, query.String())
	if err != nil {
		return nil, err
	}
	defer resp.Close()
	rows := make([]map[string]any, 0, limit)
	scanner := bufio.NewScanner(resp)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var record requestLogRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			continue
		}
		rows = append(rows, requestRecordToMap(record))
	}
	return rows, scanner.Err()
}

func (s *requestClickHouseStore) indexes(options requestQueryOptions, archiveRoot string) (map[string]any, error) {
	cfg, err := s.currentConfig()
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled {
		return nil, errClickHouseDisabled
	}
	if err := s.ensureSchema(cfg); err != nil {
		return nil, err
	}
	limit := options.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	offset := maxInt(options.Offset, 0)
	baseWhere := "WHERE 1=1"
	totalResp, err := s.query(cfg, fmt.Sprintf("SELECT count() AS total FROM (SELECT toDate(timestamp) AS day FROM %s.%s FINAL %s GROUP BY day) FORMAT JSONEachRow", cfg.Database, cfg.Table, baseWhere))
	if err != nil {
		return nil, err
	}
	total := 0
	func() {
		defer totalResp.Close()
		scanner := bufio.NewScanner(totalResp)
		if scanner.Scan() {
			var row struct {
				Total int `json:"total"`
			}
			_ = json.Unmarshal(scanner.Bytes(), &row)
			total = row.Total
		}
	}()
	query := fmt.Sprintf(
		"SELECT toString(toDate(timestamp)) AS date, count() AS lines, max(ingested_at) AS updated_at FROM %s.%s FINAL %s GROUP BY date ORDER BY date DESC LIMIT %d OFFSET %d FORMAT JSONEachRow",
		cfg.Database,
		cfg.Table,
		baseWhere,
		limit,
		offset,
	)
	resp, err := s.query(cfg, query)
	if err != nil {
		return nil, err
	}
	defer resp.Close()
	type row struct {
		Date      string `json:"date"`
		Lines     int    `json:"lines"`
		UpdatedAt string `json:"updated_at"`
	}
	items := make([]map[string]any, 0, limit)
	scanner := bufio.NewScanner(resp)
	for scanner.Scan() {
		var item row
		if err := json.Unmarshal(scanner.Bytes(), &item); err != nil {
			continue
		}
		items = append(items, map[string]any{
			"date":         item.Date,
			"file_name":    fmt.Sprintf("clickhouse:%s.%s", cfg.Database, cfg.Table),
			"lines":        item.Lines,
			"size_bytes":   0,
			"updated_at":   item.UpdatedAt,
			"storage_type": "clickhouse",
		})
	}
	state, _ := loadRequestMigrationState(filepath.Join(archiveRoot, ".clickhouse-migration-state.json"))
	return map[string]any{
		"items":             items,
		"total":             total,
		"limit":             limit,
		"offset":            offset,
		"archive_root":      archiveRoot,
		"storage_type":      "clickhouse",
		"migration_state":   state,
		"clickhouse_target": fmt.Sprintf("%s/%s.%s", cfg.Endpoint, cfg.Database, cfg.Table),
	}, scanner.Err()
}

func (s *requestClickHouseStore) deleteDay(day string) error {
	cfg, err := s.currentConfig()
	if err != nil || !cfg.Enabled {
		return err
	}
	if err := s.ensureSchema(cfg); err != nil {
		return err
	}
	return s.doQuery(cfg, fmt.Sprintf("ALTER TABLE %s.%s DELETE WHERE toDate(timestamp) = toDate('%s')", cfg.Database, cfg.Table, escapeSQLString(day)), nil)
}

func (s *requestClickHouseStore) exportDay(day string) ([]requestLogRecord, error) {
	cfg, err := s.currentConfig()
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled {
		return nil, errClickHouseDisabled
	}
	if err := s.ensureSchema(cfg); err != nil {
		return nil, err
	}
	query := strings.Builder{}
	query.WriteString("SELECT event_hash, stream, ingested_at, timestamp, request_id, client_ip, country, city, method, uri, status, site, host, upstream_addr, referer, user_agent ")
	query.WriteString(fmt.Sprintf("FROM %s.%s FINAL WHERE toDate(timestamp) = toDate('%s') ORDER BY timestamp ASC FORMAT JSONEachRow", cfg.Database, cfg.Table, escapeSQLString(day)))
	resp, err := s.query(cfg, query.String())
	if err != nil {
		return nil, err
	}
	defer resp.Close()
	out := make([]requestLogRecord, 0, 256)
	scanner := bufio.NewScanner(resp)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var record requestLogRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			continue
		}
		out = append(out, record)
	}
	return out, scanner.Err()
}

func (s *requestClickHouseStore) ensureSchema(cfg requestClickHouseConfig) error {
	key := cfg.Endpoint + "|" + cfg.Database + "|" + cfg.Table + "|" + cfg.Username
	s.mu.Lock()
	if s.lastSchemaKey == key {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()
	if err := s.doQuery(cfg, fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", cfg.Database), nil); err != nil {
		return err
	}
	createTable := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.%s (
event_hash String,
stream LowCardinality(String),
ingested_at DateTime64(9, 'UTC'),
timestamp DateTime64(9, 'UTC'),
request_id String,
client_ip String,
country LowCardinality(String),
city String,
method LowCardinality(String),
uri String,
status UInt16,
site LowCardinality(String),
host String,
upstream_addr String,
referer String,
user_agent String
) ENGINE = ReplacingMergeTree()
PARTITION BY toYYYYMMDD(timestamp)
ORDER BY (timestamp, event_hash)`, cfg.Database, cfg.Table)
	if err := s.doQuery(cfg, createTable, nil); err != nil {
		return err
	}
	if err := s.doQuery(cfg, fmt.Sprintf("ALTER TABLE %s.%s ADD COLUMN IF NOT EXISTS city String AFTER country", cfg.Database, cfg.Table), nil); err != nil {
		return err
	}
	s.mu.Lock()
	s.lastSchemaKey = key
	s.mu.Unlock()
	return nil
}

func (s *requestClickHouseStore) query(cfg requestClickHouseConfig, query string) (io.ReadCloser, error) {
	req, err := s.newRequest(cfg, query, nil)
	if err != nil {
		return nil, err
	}
	client := s.client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		return nil, fmt.Errorf("clickhouse query failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return resp.Body, nil
}

func (s *requestClickHouseStore) doQuery(cfg requestClickHouseConfig, query string, body io.Reader) error {
	req, err := s.newRequest(cfg, query, body)
	if err != nil {
		return err
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
		return fmt.Errorf("clickhouse query failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(content)))
	}
	return nil
}

func (s *requestClickHouseStore) newRequest(cfg requestClickHouseConfig, query string, body io.Reader) (*http.Request, error) {
	endpoint := strings.TrimRight(strings.TrimSpace(cfg.Endpoint), "/")
	target, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	values := target.Query()
	values.Set("query", query)
	target.RawQuery = values.Encode()
	req, err := http.NewRequest(http.MethodPost, target.String(), body)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.Username) != "" {
		req.SetBasicAuth(cfg.Username, cfg.Password)
	}
	return req, nil
}

func newRequestLogRecord(item parsedAccess) requestLogRecord {
	record := requestLogRecord{
		Stream:       "runtime",
		IngestedAt:   item.when.UTC().Format(time.RFC3339Nano),
		Timestamp:    item.when.UTC().Format(time.RFC3339Nano),
		RequestID:    item.requestID,
		ClientIP:     item.ip,
		Country:      item.country,
		City:         item.city,
		Method:       item.method,
		URI:          item.path,
		Status:       item.status,
		Site:         item.siteID,
		Host:         item.host,
		UpstreamAddr: item.upstreamAddr,
		Referer:      item.referer,
		UserAgent:    item.userAgent,
	}
	record.EventHash = requestRecordHash(record)
	return record
}

func requestRecordHash(record requestLogRecord) string {
	parts := []string{
		record.Stream,
		record.Timestamp,
		record.RequestID,
		record.ClientIP,
		record.Country,
		record.City,
		record.Method,
		record.URI,
		strconv.Itoa(record.Status),
		record.Site,
		record.Host,
		record.UpstreamAddr,
		record.Referer,
		record.UserAgent,
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])
}

func requestRecordToMap(record requestLogRecord) map[string]any {
	return map[string]any{
		"stream":      record.Stream,
		"ingested_at": record.IngestedAt,
		"entry": map[string]any{
			"timestamp":     record.Timestamp,
			"request_id":    record.RequestID,
			"client_ip":     record.ClientIP,
			"country":       record.Country,
			"city":          record.City,
			"method":        record.Method,
			"uri":           record.URI,
			"status":        record.Status,
			"site":          record.Site,
			"host":          record.Host,
			"upstream_addr": record.UpstreamAddr,
			"referer":       record.Referer,
			"user_agent":    record.UserAgent,
		},
	}
}

func loadRequestLogRecordFromArchiveLine(line string) (requestLogRecord, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return requestLogRecord{}, false
	}
	var row struct {
		Stream     string         `json:"stream"`
		IngestedAt string         `json:"ingested_at"`
		Entry      map[string]any `json:"entry"`
	}
	if err := json.Unmarshal([]byte(line), &row); err != nil {
		return requestLogRecord{}, false
	}
	record := requestLogRecord{
		Stream:       strings.TrimSpace(row.Stream),
		IngestedAt:   strings.TrimSpace(row.IngestedAt),
		Timestamp:    strings.TrimSpace(asString(row.Entry["timestamp"])),
		RequestID:    strings.TrimSpace(asString(row.Entry["request_id"])),
		ClientIP:     strings.TrimSpace(asString(row.Entry["client_ip"])),
		Country:      strings.TrimSpace(asString(row.Entry["country"])),
		City:         strings.TrimSpace(asString(row.Entry["city"])),
		Method:       strings.TrimSpace(asString(row.Entry["method"])),
		URI:          strings.TrimSpace(asString(row.Entry["uri"])),
		Status:       parseIntValue(row.Entry["status"]),
		Site:         strings.TrimSpace(asString(row.Entry["site"])),
		Host:         strings.TrimSpace(asString(row.Entry["host"])),
		UpstreamAddr: strings.TrimSpace(asString(row.Entry["upstream_addr"])),
		Referer:      strings.TrimSpace(asString(row.Entry["referer"])),
		UserAgent:    strings.TrimSpace(asString(row.Entry["user_agent"])),
	}
	if record.Stream == "" {
		record.Stream = "runtime"
	}
	if record.IngestedAt == "" {
		record.IngestedAt = record.Timestamp
	}
	record.EventHash = requestRecordHash(record)
	return record, record.Timestamp != ""
}

func loadRequestMigrationState(path string) (requestMigrationState, error) {
	state := requestMigrationState{ImportedDays: map[string]string{}}
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return state, err
	}
	if err := json.Unmarshal(content, &state); err != nil {
		return requestMigrationState{ImportedDays: map[string]string{}}, err
	}
	if state.ImportedDays == nil {
		state.ImportedDays = map[string]string{}
	}
	return state, nil
}

func saveRequestMigrationState(path string, state requestMigrationState) error {
	if state.ImportedDays == nil {
		state.ImportedDays = map[string]string{}
	}
	content, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}

func parseIntValue(value any) int {
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	case int64:
		return int(typed)
	case json.Number:
		parsed, _ := typed.Int64()
		return int(parsed)
	case string:
		parsed, _ := strconv.Atoi(strings.TrimSpace(typed))
		return parsed
	default:
		return 0
	}
}

func escapeSQLString(value string) string {
	return strings.ReplaceAll(strings.TrimSpace(value), "'", "''")
}

func maxInt(value int, fallback int) int {
	if value < fallback {
		return fallback
	}
	return value
}
