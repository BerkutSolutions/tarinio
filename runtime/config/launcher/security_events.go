package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"waf/internal/loggingconfig"
)

var accessLogPattern = regexp.MustCompile(`^(\S+) \S+ \S+ \[([^\]]+)\] "([A-Z]+) ([^"]*?) HTTP/[^"]*" (\d{3}) (\S+) "([^"]*)" "([^"]*)" "([^"]*)"$`)

type securityEvent struct {
	Type            string         `json:"type"`
	Severity        string         `json:"severity"`
	SiteID          string         `json:"site_id,omitempty"`
	SourceComponent string         `json:"source_component"`
	OccurredAt      string         `json:"occurred_at"`
	Summary         string         `json:"summary"`
	Details         map[string]any `json:"details,omitempty"`
}

type securityEventSource struct {
	mu     sync.Mutex
	path   string
	offset int64
}

const (
	defaultBurstRequestsPerSecondThreshold     = 25
	defaultBurstPathRequestsPerSecondThreshold = 10
)

func newSecurityEventSource(path string) *securityEventSource {
	return &securityEventSource{path: path}
}

func normalizeSiteID(value string) string {
	site := strings.ToLower(strings.TrimSpace(value))
	site = strings.ReplaceAll(site, "_", "-")
	return site
}

func shouldSkipInternalSite(siteID string) bool {
	switch normalizeSiteID(siteID) {
	case "control-plane-access", "control-plane", "ui":
		return true
	default:
		return false
	}
}

type requestStreamSource struct {
	mu                  sync.Mutex
	path                string
	maxItems            int
	archiveRoot         string
	defaultRetention    int
	lastIngestError     string
	lastProcessedOffset int64
	settingsPath        string
	opensearch          *requestOpenSearchStore
	clickhouse          *requestClickHouseStore
}

const requestIngestBatchLines = 4000

type requestStreamOption func(*requestStreamSource)

func withRequestClickHouse(settingsPath string, pepper string) requestStreamOption {
	return func(source *requestStreamSource) {
		source.settingsPath = strings.TrimSpace(settingsPath)
		source.clickhouse = newRequestClickHouseStore(settingsPath, pepper)
	}
}

func withRequestOpenSearch(settingsPath string, pepper string) requestStreamOption {
	return func(source *requestStreamSource) {
		source.settingsPath = strings.TrimSpace(settingsPath)
		source.opensearch = newRequestOpenSearchStore(settingsPath, pepper)
	}
}

func newRequestStreamSource(path string, maxItems int, archiveRoot string, defaultRetention int, options ...requestStreamOption) *requestStreamSource {
	if maxItems <= 0 {
		maxItems = 500
	}
	if defaultRetention <= 0 {
		defaultRetention = 30
	}
	source := &requestStreamSource{
		path:             path,
		maxItems:         maxItems,
		archiveRoot:      archiveRoot,
		defaultRetention: defaultRetention,
	}
	for _, option := range options {
		if option != nil {
			option(source)
		}
	}
	return source
}

type requestQueryOptions struct {
	Limit         int
	Offset        int
	Since         time.Time
	Day           string
	RetentionDays int
	Probe         bool
}

func parseRequestQueryOptions(values url.Values, maxItems int, defaultRetention int) requestQueryOptions {
	limit := maxItems
	if raw := strings.TrimSpace(values.Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if limit <= 0 {
		limit = 500
	}
	offset := 0
	if raw := strings.TrimSpace(values.Get("offset")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			offset = parsed
		}
	}
	since := time.Time{}
	if raw := strings.TrimSpace(values.Get("since")); raw != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, raw); err == nil {
			since = parsed.UTC()
		} else if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
			since = parsed.UTC()
		}
	}
	day := strings.TrimSpace(values.Get("day"))
	if day != "" {
		if _, err := time.Parse("2006-01-02", day); err != nil {
			day = ""
		}
	}
	retentionDays := defaultRetention
	if raw := strings.TrimSpace(values.Get("retention_days")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			retentionDays = parsed
		}
	}
	if retentionDays <= 0 {
		retentionDays = defaultRetention
	}
	probe := false
	if raw := strings.TrimSpace(values.Get("probe")); raw != "" {
		switch strings.ToLower(raw) {
		case "1", "true", "yes", "on":
			probe = true
		}
	}
	return requestQueryOptions{
		Limit:         limit,
		Offset:        offset,
		Since:         since,
		Day:           day,
		RetentionDays: retentionDays,
		Probe:         probe,
	}
}

func (s *requestStreamSource) latest(query url.Values) ([]map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	options := parseRequestQueryOptions(query, s.maxItems, s.defaultRetention)

	if err := s.ensureArchiveRootLocked(); err != nil {
		return nil, err
	}

	if err := s.ingestLatestLocked(options.RetentionDays); err != nil {
		return nil, err
	}
	if options.Probe {
		return []map[string]any{}, nil
	}
	items, handled, err := s.latestFromBackendsLocked(options)
	if handled {
		return items, err
	}
	return s.loadArchiveRowsLocked(options)
}

func (s *requestStreamSource) probe(query url.Values) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	options := parseRequestQueryOptions(query, s.maxItems, s.defaultRetention)
	if err := s.ensureArchiveRootLocked(); err != nil {
		return err
	}
	return s.ingestLatestLocked(options.RetentionDays)
}

func (s *requestStreamSource) startBackgroundIngest(interval time.Duration) {
	if interval <= 0 {
		interval = 3 * time.Second
	}
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			s.mu.Lock()
			if err := s.ensureArchiveRootLocked(); err != nil {
				s.lastIngestError = err.Error()
				s.mu.Unlock()
				continue
			}
			if err := s.ingestLatestLocked(s.defaultRetention); err != nil {
				s.lastIngestError = err.Error()
			} else {
				s.lastIngestError = ""
			}
			s.mu.Unlock()
		}
	}()
}

func (s *requestStreamSource) ingestLatestLocked(retentionDays int) error {
	file, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			if migrateErr := s.migrateHotToColdLocked(retentionDays); migrateErr != nil {
				s.lastIngestError = migrateErr.Error()
			}
			s.pruneArchiveLocked(retentionDays)
			return nil
		}
		return err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return err
	}
	if s.lastProcessedOffset > stat.Size() {
		s.lastProcessedOffset = 0
	}
	if _, err := file.Seek(s.lastProcessedOffset, 0); err != nil {
		return err
	}

	reader := bufio.NewReaderSize(file, 64*1024)
	rowsByDay := map[string][][]byte{}
	records := make([]requestLogRecord, 0, requestIngestBatchLines)
	nextOffset := s.lastProcessedOffset
	processed := 0
	for processed < requestIngestBatchLines {
		chunk, readErr := reader.ReadBytes('\n')
		if len(chunk) == 0 && errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return readErr
		}
		if len(chunk) == 0 {
			break
		}
		nextOffset += int64(len(chunk))
		processed++
		line := strings.TrimSpace(string(chunk))
		if line == "" {
			if errors.Is(readErr, io.EOF) {
				break
			}
			continue
		}
		item, ok := parseAccessLine(line)
		if !ok {
			if errors.Is(readErr, io.EOF) {
				break
			}
			continue
		}
		if shouldSkipInternalManagementRequest(item) {
			if errors.Is(readErr, io.EOF) {
				break
			}
			continue
		}
		record := newRequestLogRecord(item)
		row := requestRecordToMap(record)
		content, marshalErr := json.Marshal(row)
		if marshalErr != nil {
			continue
		}
		records = append(records, record)
		day := item.when.UTC().Format("2006-01-02")
		rowsByDay[day] = append(rowsByDay[day], append(content, '\n'))
		if errors.Is(readErr, io.EOF) {
			break
		}
	}
	s.lastProcessedOffset = nextOffset

	for day, batch := range rowsByDay {
		path := filepath.Join(s.archiveRoot, day+".jsonl")
		handle, openErr := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if openErr != nil {
			continue
		}
		for _, row := range batch {
			_, _ = handle.Write(row)
		}
		_ = handle.Close()
	}
	settings := s.loadLoggingSettingsLocked()
	if s.opensearch != nil && len(records) > 0 && settings.Routing.WriteRequestsToHot {
		if err := s.opensearch.insert(records); err != nil {
			s.lastIngestError = err.Error()
		} else {
			s.lastIngestError = ""
		}
	}
	if s.clickhouse != nil && len(records) > 0 && settings.Routing.WriteRequestsToCold {
		if err := s.clickhouse.insert(records); err != nil {
			s.lastIngestError = err.Error()
		} else {
			s.lastIngestError = ""
		}
	}
	if s.clickhouse != nil {
		if err := s.importArchiveDaysToClickHouseLocked(); err != nil {
			s.lastIngestError = err.Error()
		}
	}
	if err := s.migrateHotToColdLocked(retentionDays); err != nil {
		s.lastIngestError = err.Error()
	}
	s.pruneArchiveLocked(retentionDays)
	return nil
}

func (s *requestStreamSource) loadLoggingSettingsLocked() loggingconfig.Settings {
	if strings.TrimSpace(s.settingsPath) == "" {
		return loggingconfig.Normalize(loggingconfig.Settings{
			Backend: loggingconfig.BackendOpenSearch,
			Hot:     loggingconfig.HotSettings{Backend: loggingconfig.BackendOpenSearch},
			Cold:    loggingconfig.ColdSettings{Backend: loggingconfig.BackendClickHouse},
			Retention: loggingconfig.RetentionSettings{
				HotDays:  loggingconfig.DefaultHotDays,
				ColdDays: loggingconfig.DefaultColdDays,
			},
			Routing: loggingconfig.RoutingSettings{
				WriteRequestsToHot: true,
				KeepLocalFallback:  true,
			},
		})
	}
	content, err := os.ReadFile(s.settingsPath)
	if err != nil {
		return loggingconfig.Normalize(loggingconfig.Settings{
			Backend: loggingconfig.BackendOpenSearch,
			Hot:     loggingconfig.HotSettings{Backend: loggingconfig.BackendOpenSearch},
			Cold:    loggingconfig.ColdSettings{Backend: loggingconfig.BackendClickHouse},
			Retention: loggingconfig.RetentionSettings{
				HotDays:  loggingconfig.DefaultHotDays,
				ColdDays: loggingconfig.DefaultColdDays,
			},
			Routing: loggingconfig.RoutingSettings{
				WriteRequestsToHot: true,
				KeepLocalFallback:  true,
			},
		})
	}
	var payload struct {
		Logging loggingconfig.Settings `json:"logging"`
	}
	if err := json.Unmarshal(content, &payload); err != nil {
		return loggingconfig.Normalize(loggingconfig.Settings{
			Backend: loggingconfig.BackendOpenSearch,
			Hot:     loggingconfig.HotSettings{Backend: loggingconfig.BackendOpenSearch},
			Cold:    loggingconfig.ColdSettings{Backend: loggingconfig.BackendClickHouse},
			Retention: loggingconfig.RetentionSettings{
				HotDays:  loggingconfig.DefaultHotDays,
				ColdDays: loggingconfig.DefaultColdDays,
			},
			Routing: loggingconfig.RoutingSettings{
				WriteRequestsToHot: true,
				KeepLocalFallback:  true,
			},
		})
	}
	return loggingconfig.Normalize(payload.Logging)
}

func (s *requestStreamSource) latestFromBackendsLocked(options requestQueryOptions) ([]map[string]any, bool, error) {
	combined := make([]map[string]any, 0)
	seen := map[string]struct{}{}
	hadBackend := false

	if s.opensearch != nil {
		items, err := s.opensearch.latest(options)
		if err == nil {
			hadBackend = true
			appendRequestRowsDedup(&combined, seen, items)
		} else if !errors.Is(err, errOpenSearchDisabled) {
			s.lastIngestError = err.Error()
		}
	}
	if s.clickhouse != nil {
		items, err := s.clickhouse.latest(options)
		if err == nil {
			hadBackend = true
			appendRequestRowsDedup(&combined, seen, items)
		} else if !errors.Is(err, errClickHouseDisabled) {
			s.lastIngestError = err.Error()
		}
	}
	if !hadBackend {
		return nil, false, nil
	}
	sort.Slice(combined, func(i, j int) bool {
		left := requestRowTimestamp(combined[i])
		right := requestRowTimestamp(combined[j])
		return left.After(right)
	})
	if options.Offset > 0 || (options.Limit > 0 && len(combined) > options.Limit) {
		start := maxInt(options.Offset, 0)
		if start > len(combined) {
			start = len(combined)
		}
		end := len(combined)
		if options.Limit > 0 && start+options.Limit < end {
			end = start + options.Limit
		}
		combined = combined[start:end]
	}
	return combined, true, nil
}

func (s *requestStreamSource) ensureArchiveRootLocked() error {
	if err := os.MkdirAll(s.archiveRoot, 0o755); err == nil {
		return nil
	}
	fallback := filepath.Join(os.TempDir(), "waf-requests-archive")
	if mkErr := os.MkdirAll(fallback, 0o755); mkErr != nil {
		return mkErr
	}
	s.archiveRoot = fallback
	return nil
}

func (s *requestStreamSource) pruneArchiveLocked(retentionDays int) {
	if retentionDays <= 0 {
		retentionDays = 30
	}
	entries, err := os.ReadDir(s.archiveRoot)
	if err != nil {
		return
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if !strings.HasSuffix(name, ".jsonl") {
			continue
		}
		day := strings.TrimSuffix(name, ".jsonl")
		parsed, parseErr := time.Parse("2006-01-02", day)
		if parseErr != nil {
			continue
		}
		if parsed.Before(cutoff) {
			_ = os.Remove(filepath.Join(s.archiveRoot, name))
		}
	}
}

func (s *requestStreamSource) listArchiveDaysLocked(targetDay string) ([]string, error) {
	if strings.TrimSpace(targetDay) != "" {
		return []string{targetDay}, nil
	}
	entries, err := os.ReadDir(s.archiveRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	days := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.TrimSpace(entry.Name())
		if !strings.HasSuffix(name, ".jsonl") {
			continue
		}
		day := strings.TrimSuffix(name, ".jsonl")
		if _, err := time.Parse("2006-01-02", day); err != nil {
			continue
		}
		days = append(days, day)
	}
	sort.Slice(days, func(i, j int) bool { return days[i] > days[j] })
	return days, nil
}

func (s *requestStreamSource) loadArchiveRowsLocked(options requestQueryOptions) ([]map[string]any, error) {
	days, err := s.listArchiveDaysLocked(options.Day)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, options.Limit)
	skip := options.Offset

	for _, day := range days {
		if len(out) >= options.Limit {
			break
		}
		content, readErr := os.ReadFile(filepath.Join(s.archiveRoot, day+".jsonl"))
		if readErr != nil {
			if os.IsNotExist(readErr) {
				continue
			}
			return nil, readErr
		}
		lines := strings.Split(string(content), "\n")
		for i := len(lines) - 1; i >= 0; i-- {
			line := strings.TrimSpace(lines[i])
			if line == "" {
				continue
			}
			var row map[string]any
			if err := json.Unmarshal([]byte(line), &row); err != nil {
				continue
			}
			if !options.Since.IsZero() {
				entry, _ := row["entry"].(map[string]any)
				ts := ""
				if entry != nil {
					ts = strings.TrimSpace(asString(entry["timestamp"]))
				}
				if ts == "" {
					ts = strings.TrimSpace(asString(row["ingested_at"]))
				}
				if ts != "" {
					parsed, err := time.Parse(time.RFC3339Nano, ts)
					if err != nil {
						parsed, err = time.Parse(time.RFC3339, ts)
					}
					if err == nil && parsed.UTC().Before(options.Since) {
						continue
					}
				}
			}
			if skip > 0 {
				skip--
				continue
			}
			out = append(out, row)
			if len(out) >= options.Limit {
				break
			}
		}
	}
	reverseRequestRows(out)
	return out, nil
}

type requestArchiveIndex struct {
	Date      string `json:"date"`
	FileName  string `json:"file_name"`
	SizeBytes int64  `json:"size_bytes"`
	Lines     int    `json:"lines"`
	UpdatedAt string `json:"updated_at"`
}

func (s *requestStreamSource) indexes(query url.Values) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureArchiveRootLocked(); err != nil {
		return nil, err
	}
	if err := s.ingestLatestLocked(s.defaultRetention); err != nil {
		s.lastIngestError = err.Error()
	}

	limit := 10
	offset := 0
	if raw := strings.TrimSpace(query.Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if raw := strings.TrimSpace(query.Get("offset")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 0 {
			offset = parsed
		}
	}
	if limit > 50 {
		limit = 50
	}

	options := requestQueryOptions{Limit: limit, Offset: offset, RetentionDays: s.defaultRetention}
	if payload, ok := s.indexesFromBackendsLocked(options); ok {
		payload["last_ingest_error"] = s.lastIngestError
		return payload, nil
	}
	days, err := s.listArchiveDaysLocked("")
	if err != nil {
		return nil, err
	}
	total := len(days)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}

	items := make([]requestArchiveIndex, 0, end-offset)
	for _, day := range days[offset:end] {
		path := filepath.Join(s.archiveRoot, day+".jsonl")
		info, statErr := os.Stat(path)
		if statErr != nil {
			continue
		}
		lines := 0
		file, openErr := os.Open(path)
		if openErr == nil {
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				if strings.TrimSpace(scanner.Text()) != "" {
					lines++
				}
			}
			_ = file.Close()
		}
		items = append(items, requestArchiveIndex{
			Date:      day,
			FileName:  filepath.Base(path),
			SizeBytes: info.Size(),
			Lines:     lines,
			UpdatedAt: info.ModTime().UTC().Format(time.RFC3339),
		})
	}

	return map[string]any{
		"items":             items,
		"total":             total,
		"limit":             limit,
		"offset":            offset,
		"archive_root":      s.archiveRoot,
		"last_ingest_error": s.lastIngestError,
	}, nil
}

func (s *requestStreamSource) deleteIndex(query url.Values) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureArchiveRootLocked(); err != nil {
		return err
	}
	day := strings.TrimSpace(query.Get("date"))
	if day == "" {
		return errors.New("date is required")
	}
	if _, err := time.Parse("2006-01-02", day); err != nil {
		return errors.New("date must be in YYYY-MM-DD format")
	}
	if s.clickhouse != nil {
		if s.opensearch != nil {
			if err := s.opensearch.deleteDay(day); err != nil {
				s.lastIngestError = err.Error()
			}
		}
	}
	if s.clickhouse != nil {
		if err := s.clickhouse.deleteDay(day); err != nil {
			s.lastIngestError = err.Error()
		}
	}
	path := filepath.Join(s.archiveRoot, day+".jsonl")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	statePath := filepath.Join(s.archiveRoot, ".clickhouse-migration-state.json")
	state, err := loadRequestMigrationState(statePath)
	if err == nil {
		delete(state.ImportedDays, day)
		_ = saveRequestMigrationState(statePath, state)
	}
	return nil
}

func (s *requestStreamSource) importArchiveDaysToClickHouseLocked() error {
	if s.clickhouse == nil {
		return nil
	}
	cfg, err := s.clickhouse.currentConfig()
	if err != nil || !cfg.Enabled || !cfg.MigrationEnabled {
		return err
	}
	statePath := filepath.Join(s.archiveRoot, ".clickhouse-migration-state.json")
	state, err := loadRequestMigrationState(statePath)
	if err != nil {
		return err
	}
	days, err := s.listArchiveDaysLocked("")
	if err != nil {
		return err
	}
	for _, day := range days {
		if _, ok := state.ImportedDays[day]; ok {
			continue
		}
		content, err := os.ReadFile(filepath.Join(s.archiveRoot, day+".jsonl"))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		lines := strings.Split(string(content), "\n")
		records := make([]requestLogRecord, 0, len(lines))
		for _, line := range lines {
			record, ok := loadRequestLogRecordFromArchiveLine(line)
			if !ok {
				continue
			}
			records = append(records, record)
		}
		if len(records) == 0 {
			state.ImportedDays[day] = time.Now().UTC().Format(time.RFC3339)
			continue
		}
		if err := s.clickhouse.insert(records); err != nil {
			return err
		}
		state.ImportedDays[day] = time.Now().UTC().Format(time.RFC3339)
	}
	return saveRequestMigrationState(statePath, state)
}

func (s *requestStreamSource) migrateHotToColdLocked(retentionDays int) error {
	if s.opensearch == nil || s.clickhouse == nil {
		return nil
	}
	settings := s.loadLoggingSettingsLocked()
	if settings.Cold.Backend != loggingconfig.BackendClickHouse || settings.Hot.Backend != loggingconfig.BackendOpenSearch {
		return nil
	}
	if retentionDays <= 0 {
		retentionDays = settings.Retention.HotDays
	}
	if retentionDays <= 0 {
		retentionDays = loggingconfig.DefaultHotDays
	}
	statePath := filepath.Join(s.archiveRoot, ".hot-to-cold-migration-state.json")
	state, err := loadRequestMigrationState(statePath)
	if err != nil {
		return err
	}
	days, err := s.opensearch.days()
	if err != nil {
		if errors.Is(err, errOpenSearchDisabled) {
			return nil
		}
		return err
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)
	for _, day := range days {
		if _, ok := state.ImportedDays[day]; ok {
			continue
		}
		parsedDay, err := time.Parse("2006-01-02", day)
		if err != nil || !parsedDay.Before(cutoff) {
			continue
		}
		records, err := s.opensearch.exportDay(day)
		if err != nil {
			return err
		}
		if len(records) > 0 {
			if err := s.clickhouse.insert(records); err != nil {
				return err
			}
		}
		if err := s.opensearch.deleteDay(day); err != nil {
			return err
		}
		state.ImportedDays[day] = time.Now().UTC().Format(time.RFC3339)
	}
	return saveRequestMigrationState(statePath, state)
}

func (s *requestStreamSource) indexesFromBackendsLocked(options requestQueryOptions) (map[string]any, bool) {
	mergedItems := make([]map[string]any, 0)
	hadBackend := false
	seen := map[string]struct{}{}
	total := 0

	if s.opensearch != nil {
		payload, err := s.opensearch.indexes(requestQueryOptions{Limit: 500, Offset: 0}, s.archiveRoot)
		if err == nil {
			hadBackend = true
			total += parseMapInt(payload, "total")
			for _, item := range parseIndexItems(payload) {
				key := indexItemKey(item)
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				mergedItems = append(mergedItems, item)
			}
		} else if !errors.Is(err, errOpenSearchDisabled) {
			s.lastIngestError = err.Error()
		}
	}
	if s.clickhouse != nil {
		payload, err := s.clickhouse.indexes(requestQueryOptions{Limit: 500, Offset: 0}, s.archiveRoot)
		if err == nil {
			hadBackend = true
			total += parseMapInt(payload, "total")
			for _, item := range parseIndexItems(payload) {
				key := indexItemKey(item)
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				mergedItems = append(mergedItems, item)
			}
		} else if !errors.Is(err, errClickHouseDisabled) {
			s.lastIngestError = err.Error()
		}
	}
	if !hadBackend {
		return nil, false
	}
	sort.Slice(mergedItems, func(i, j int) bool {
		return strings.TrimSpace(asString(mergedItems[i]["date"])) > strings.TrimSpace(asString(mergedItems[j]["date"]))
	})
	start := maxInt(options.Offset, 0)
	if start > len(mergedItems) {
		start = len(mergedItems)
	}
	end := len(mergedItems)
	if options.Limit > 0 && start+options.Limit < end {
		end = start + options.Limit
	}
	return map[string]any{
		"items":        mergedItems[start:end],
		"total":        len(mergedItems),
		"limit":        options.Limit,
		"offset":       options.Offset,
		"archive_root": s.archiveRoot,
		"storage_type": "tiered",
	}, true
}

func reverseRequestRows(items []map[string]any) {
	for left, right := 0, len(items)-1; left < right; left, right = left+1, right-1 {
		items[left], items[right] = items[right], items[left]
	}
}

func appendRequestRowsDedup(target *[]map[string]any, seen map[string]struct{}, items []map[string]any) {
	for _, item := range items {
		key := requestRowKey(item)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		*target = append(*target, item)
	}
}

func requestRowKey(item map[string]any) string {
	entry, _ := item["entry"].(map[string]any)
	return strings.Join([]string{
		strings.TrimSpace(asString(item["stream"])),
		strings.TrimSpace(asString(item["ingested_at"])),
		strings.TrimSpace(asString(entry["timestamp"])),
		strings.TrimSpace(asString(entry["request_id"])),
		strings.TrimSpace(asString(entry["client_ip"])),
		strings.TrimSpace(asString(entry["uri"])),
		strconv.Itoa(parseIntValue(entry["status"])),
	}, "|")
}

func requestRowTimestamp(item map[string]any) time.Time {
	entry, _ := item["entry"].(map[string]any)
	for _, raw := range []string{
		strings.TrimSpace(asString(entry["timestamp"])),
		strings.TrimSpace(asString(item["ingested_at"])),
	} {
		if raw == "" {
			continue
		}
		if parsed, err := time.Parse(time.RFC3339Nano, raw); err == nil {
			return parsed.UTC()
		}
		if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
			return parsed.UTC()
		}
	}
	return time.Time{}
}

func parseIndexItems(payload map[string]any) []map[string]any {
	raw, _ := payload["items"].([]map[string]any)
	if raw != nil {
		return raw
	}
	itemsRaw, _ := payload["items"].([]any)
	out := make([]map[string]any, 0, len(itemsRaw))
	for _, item := range itemsRaw {
		typed, _ := item.(map[string]any)
		if typed != nil {
			out = append(out, typed)
		}
	}
	return out
}

func parseMapInt(payload map[string]any, key string) int {
	if payload == nil {
		return 0
	}
	return parseIntValue(payload[key])
}

func indexItemKey(item map[string]any) string {
	return strings.Join([]string{
		strings.TrimSpace(asString(item["date"])),
		strings.TrimSpace(asString(item["storage_type"])),
		strings.TrimSpace(asString(item["file_name"])),
	}, "|")
}

func asString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []byte:
		return string(typed)
	default:
		return ""
	}
}

func (s *securityEventSource) next() ([]securityEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if s.offset > stat.Size() {
		s.offset = 0
	}
	if _, err := file.Seek(s.offset, 0); err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	out := make([]securityEvent, 0, 32)
	burstBySecond := make(map[string]int)
	burstBySecondAndPath := make(map[string]int)
	burstMeta := make(map[string]map[string]string)
	burstThreshold := burstThresholdFromEnv("WAF_SECURITY_EVENT_BURST_RPS_THRESHOLD", defaultBurstRequestsPerSecondThreshold)
	burstPathThreshold := burstThresholdFromEnv("WAF_SECURITY_EVENT_BURST_PATH_RPS_THRESHOLD", defaultBurstPathRequestsPerSecondThreshold)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		item, ok := parseAccessLine(line)
		if !ok {
			continue
		}
		if !shouldTrackRequestBurst(item) {
			continue
		}

		second := item.when.UTC().Format("2006-01-02T15:04:05Z")
		scopeKey := burstScopeKey(item)
		burstKey := item.ip + "|" + scopeKey + "|" + second
		burstBySecond[burstKey]++
		burstPath := normalizeBurstPath(item.path)
		burstPathKey := item.ip + "|" + scopeKey + "|" + burstPath + "|" + second
		burstBySecondAndPath[burstPathKey]++
		if _, exists := burstMeta[burstKey]; !exists {
			burstMeta[burstKey] = map[string]string{
				"ip":      item.ip,
				"ts":      second,
				"site_id": item.siteID,
				"host":    item.host,
				"country": item.country,
			}
		}
		if _, exists := burstMeta[burstPathKey]; !exists {
			burstMeta[burstPathKey] = map[string]string{
				"ip":      item.ip,
				"path":    burstPath,
				"ts":      second,
				"site_id": item.siteID,
				"host":    item.host,
				"country": item.country,
			}
		}

		switch item.status {
		case 429:
			out = append(out, securityEvent{
				Type:            "security_rate_limit",
				Severity:        "warning",
				SiteID:          item.siteID,
				SourceComponent: "runtime-nginx",
				OccurredAt:      item.when.UTC().Format(time.RFC3339Nano),
				Summary:         "rate limit triggered",
				Details: map[string]any{
					"status":     item.status,
					"method":     item.method,
					"path":       item.path,
					"client_ip":  item.ip,
					"country":    item.country,
					"host":       item.host,
					"referer":    item.referer,
					"user_agent": item.userAgent,
				},
			})
		case 403, 444:
			out = append(out, securityEvent{
				Type:            "security_access",
				Severity:        "warning",
				SiteID:          item.siteID,
				SourceComponent: "runtime-nginx",
				OccurredAt:      item.when.UTC().Format(time.RFC3339Nano),
				Summary:         "access blocked",
				Details: map[string]any{
					"status":     item.status,
					"method":     item.method,
					"path":       item.path,
					"client_ip":  item.ip,
					"country":    item.country,
					"host":       item.host,
					"referer":    item.referer,
					"user_agent": item.userAgent,
				},
			})
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if pos, err := file.Seek(0, 1); err == nil {
		s.offset = pos
	}

	for key, count := range burstBySecond {
		if count < burstThreshold {
			continue
		}
		meta := burstMeta[key]
		out = append(out, securityEvent{
			Type:            "security_waf",
			Severity:        "warning",
			SiteID:          sanitizeSiteID(meta["site_id"]),
			SourceComponent: "runtime-nginx",
			OccurredAt:      meta["ts"],
			Summary:         "request burst detected (not blocked)",
			Details: map[string]any{
				"client_ip":       meta["ip"],
				"requests_second": count,
				"host":            meta["host"],
				"country":         meta["country"],
				"blocked":         false,
			},
		})
	}
	for key, count := range burstBySecondAndPath {
		if count < burstPathThreshold {
			continue
		}
		meta := burstMeta[key]
		out = append(out, securityEvent{
			Type:            "security_waf",
			Severity:        "warning",
			SiteID:          sanitizeSiteID(meta["site_id"]),
			SourceComponent: "runtime-nginx",
			OccurredAt:      meta["ts"],
			Summary:         "request burst detected on path (not blocked)",
			Details: map[string]any{
				"client_ip":         meta["ip"],
				"path":              meta["path"],
				"path_requests_sec": count,
				"host":              meta["host"],
				"country":           meta["country"],
				"blocked":           false,
			},
		})
	}

	return out, nil
}

func (s *securityEventSource) probe() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	_, err = file.Stat()
	return err
}

type parsedAccess struct {
	requestID    string
	ip           string
	siteID       string
	host         string
	country      string
	method       string
	path         string
	status       int
	upstreamAddr string
	referer      string
	userAgent    string
	when         time.Time
}

func parseAccessLine(line string) (parsedAccess, bool) {
	if strings.HasPrefix(line, "{") {
		var item struct {
			Timestamp    string `json:"timestamp"`
			RequestID    string `json:"request_id"`
			ClientIP     string `json:"client_ip"`
			Country      string `json:"country"`
			Method       string `json:"method"`
			URI          string `json:"uri"`
			Status       int    `json:"status"`
			Referer      string `json:"referer"`
			UserAgent    string `json:"user_agent"`
			Site         string `json:"site"`
			Host         string `json:"host"`
			UpstreamAddr string `json:"upstream_addr"`
		}
		if err := json.Unmarshal([]byte(line), &item); err == nil {
			ip := strings.TrimSpace(item.ClientIP)
			if ip == "" || item.Status <= 0 {
				return parsedAccess{}, false
			}
			when, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(item.Timestamp))
			if err != nil {
				when, err = time.Parse(time.RFC3339, strings.TrimSpace(item.Timestamp))
				if err != nil {
					return parsedAccess{}, false
				}
			}
			return parsedAccess{
				requestID:    strings.TrimSpace(item.RequestID),
				ip:           ip,
				siteID:       sanitizeSiteID(item.Site),
				host:         strings.ToLower(strings.TrimSpace(item.Host)),
				country:      strings.ToUpper(strings.TrimSpace(item.Country)),
				method:       strings.TrimSpace(item.Method),
				path:         strings.TrimSpace(item.URI),
				status:       item.Status,
				upstreamAddr: strings.TrimSpace(item.UpstreamAddr),
				referer:      strings.TrimSpace(item.Referer),
				userAgent:    strings.TrimSpace(item.UserAgent),
				when:         when.UTC(),
			}, true
		}
	}

	matches := accessLogPattern.FindStringSubmatch(line)
	if len(matches) != 10 {
		return parsedAccess{}, false
	}
	when, err := time.Parse("02/Jan/2006:15:04:05 -0700", matches[2])
	if err != nil {
		return parsedAccess{}, false
	}
	status, err := strconv.Atoi(matches[5])
	if err != nil {
		return parsedAccess{}, false
	}
	return parsedAccess{
		ip:           matches[1],
		siteID:       sanitizeSiteID(matches[9]),
		host:         "",
		country:      "",
		method:       matches[3],
		path:         matches[4],
		status:       status,
		referer:      matches[7],
		userAgent:    matches[8],
		upstreamAddr: "",
		when:         when,
	}, true
}

func sanitizeSiteID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" || value == "-" {
		return ""
	}
	return value
}

func burstThresholdFromEnv(name string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func burstScopeKey(item parsedAccess) string {
	if siteID := sanitizeSiteID(item.siteID); siteID != "" {
		return siteID
	}
	if host := strings.ToLower(strings.TrimSpace(item.host)); host != "" {
		return host
	}
	return "unknown"
}

func normalizeBurstPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" || trimmed == "-" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err == nil && strings.TrimSpace(parsed.Path) != "" {
		trimmed = parsed.Path
	}
	return strings.ToLower(strings.TrimSpace(trimmed))
}

func shouldTrackRequestBurst(item parsedAccess) bool {
	if shouldSkipInternalManagementRequest(item) {
		return false
	}
	if shouldSkipInternalSite(item.siteID) {
		return false
	}
	if item.status == 429 || item.status == 403 || item.status == 444 {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(item.method), "OPTIONS") {
		return false
	}
	return !shouldIgnoreBurstPath(item.path)
}

func shouldSkipInternalManagementRequest(item parsedAccess) bool {
	if shouldSkipInternalSite(item.siteID) {
		return true
	}
	path := normalizeBurstPath(item.path)
	if path == "" || !isInternalManagementPath(path) {
		return false
	}
	host := strings.ToLower(strings.TrimSpace(item.host))
	return host == "" || isInternalManagementHost(host) || sanitizeSiteID(item.siteID) == ""
}

func isInternalManagementHost(host string) bool {
	switch strings.TrimSpace(strings.ToLower(host)) {
	case "localhost", "127.0.0.1", "::1", "control-plane", "ui":
		return true
	default:
		return false
	}
}

func isInternalManagementPath(path string) bool {
	switch {
	case strings.HasPrefix(path, "/api/"),
		strings.HasPrefix(path, "/static/"),
		strings.HasPrefix(path, "/dashboard"),
		strings.HasPrefix(path, "/healthz"),
		strings.HasPrefix(path, "/readyz"),
		strings.HasPrefix(path, "/login"),
		strings.HasPrefix(path, "/logout"),
		strings.HasPrefix(path, "/setup"),
		strings.HasPrefix(path, "/onboarding"),
		strings.HasPrefix(path, "/favicon"),
		strings.HasPrefix(path, "/manifest"),
		strings.HasPrefix(path, "/site.webmanifest"):
		return true
	default:
		return false
	}
}

func shouldIgnoreBurstPath(path string) bool {
	normalized := normalizeBurstPath(path)
	if normalized == "" {
		return false
	}
	for _, prefix := range []string{
		"/_static/",
		"/static/",
		"/assets/",
		"/build/",
		"/dist/",
		"/favicon",
		"/robots.txt",
		"/manifest",
		"/site.webmanifest",
		"/browserconfig.xml",
		"/apple-touch-icon",
		"/sitemap",
	} {
		if strings.HasPrefix(normalized, prefix) {
			return true
		}
	}
	for _, suffix := range []string{
		".css",
		".js",
		".mjs",
		".map",
		".png",
		".jpg",
		".jpeg",
		".gif",
		".svg",
		".ico",
		".webp",
		".avif",
		".woff",
		".woff2",
		".ttf",
		".otf",
		".eot",
		".json",
		".xml",
		".txt",
	} {
		if strings.HasSuffix(normalized, suffix) {
			return true
		}
	}
	return false
}
