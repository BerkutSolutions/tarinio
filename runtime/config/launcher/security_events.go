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
}

const requestIngestBatchLines = 4000

func newRequestStreamSource(path string, maxItems int, archiveRoot string, defaultRetention int) *requestStreamSource {
	if maxItems <= 0 {
		maxItems = 500
	}
	if defaultRetention <= 0 {
		defaultRetention = 30
	}
	return &requestStreamSource{
		path:             path,
		maxItems:         maxItems,
		archiveRoot:      archiveRoot,
		defaultRetention: defaultRetention,
	}
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
		row := map[string]any{
			"stream":      "runtime",
			"ingested_at": item.when.UTC().Format(time.RFC3339Nano),
			"entry": map[string]any{
				"timestamp":     item.when.UTC().Format(time.RFC3339Nano),
				"request_id":    item.requestID,
				"client_ip":     item.ip,
				"country":       item.country,
				"method":        item.method,
				"uri":           item.path,
				"status":        item.status,
				"site":          item.siteID,
				"host":          item.host,
				"upstream_addr": item.upstreamAddr,
				"referer":       item.referer,
				"user_agent":    item.userAgent,
			},
		}
		content, marshalErr := json.Marshal(row)
		if marshalErr != nil {
			continue
		}
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
	s.pruneArchiveLocked(retentionDays)
	return nil
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
	path := filepath.Join(s.archiveRoot, day+".jsonl")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func reverseRequestRows(items []map[string]any) {
	for left, right := 0, len(items)-1; left < right; left, right = left+1, right-1 {
		items[left], items[right] = items[right], items[left]
	}
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
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		item, ok := parseAccessLine(line)
		if !ok {
			continue
		}
		if shouldSkipInternalSite(item.siteID) {
			continue
		}

		second := item.when.UTC().Format("2006-01-02T15:04:05Z")
		burstKey := item.ip + "|" + second
		burstBySecond[burstKey]++
		burstPathKey := item.ip + "|" + item.path + "|" + second
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
				"path":    item.path,
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
		if count < 8 {
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
		if count < 4 {
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
