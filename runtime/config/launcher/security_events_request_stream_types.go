package main

import (
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

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

const requestOpenSearchMigrationStateFile = ".opensearch-migration-state.json"
const requestClickHouseMigrationStateFile = ".clickhouse-migration-state.json"
const requestHotToColdMigrationStateFile = ".hot-to-cold-migration-state.json"

const requestIngestBatchLines = 4000
const maxRequestQueryItems = 1000
const maxRequestQueryOffset = 100000
const maxRequestHistoryDays = 31

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
	Limit                 int
	Offset                int
	Since                 time.Time
	Day                   string
	TimezoneOffsetMinutes int
	RetentionDays         int
	Probe                 bool
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
	if limit > maxRequestQueryItems {
		limit = maxRequestQueryItems
	}
	offset := 0
	if raw := strings.TrimSpace(values.Get("offset")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			offset = parsed
		}
	}
	if offset > maxRequestQueryOffset {
		offset = maxRequestQueryOffset
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
	timezoneOffsetMinutes := 0
	if raw := strings.TrimSpace(values.Get("tz_offset_minutes")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			timezoneOffsetMinutes = normalizeTimezoneOffsetMinutes(parsed)
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
		Limit:                 limit,
		Offset:                offset,
		Since:                 since,
		Day:                   day,
		TimezoneOffsetMinutes: timezoneOffsetMinutes,
		RetentionDays:         retentionDays,
		Probe:                 probe,
	}
}
