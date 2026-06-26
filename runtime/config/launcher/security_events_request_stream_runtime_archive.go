package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"waf/internal/loggingconfig"
)

func defaultLocalRequestLoggingSettings() loggingconfig.Settings {
	return loggingconfig.Normalize(loggingconfig.Settings{
		Backend: loggingconfig.BackendFile,
		Hot:     loggingconfig.HotSettings{Backend: loggingconfig.BackendFile},
		Cold:    loggingconfig.ColdSettings{Backend: loggingconfig.BackendFile},
		Retention: loggingconfig.RetentionSettings{
			HotDays:  loggingconfig.DefaultHotDays,
			ColdDays: loggingconfig.DefaultColdDays,
		},
		Routing: loggingconfig.RoutingSettings{
			KeepLocalFallback: true,
		},
	})
}

func (s *requestStreamSource) loadLoggingSettingsLocked() loggingconfig.Settings {
	if strings.TrimSpace(s.settingsPath) == "" {
		return defaultLocalRequestLoggingSettings()
	}
	content, err := os.ReadFile(s.settingsPath)
	if err != nil {
		return defaultLocalRequestLoggingSettings()
	}
	var payload struct {
		Logging loggingconfig.Settings `json:"logging"`
	}
	if err := json.Unmarshal(content, &payload); err != nil {
		return defaultLocalRequestLoggingSettings()
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
	var (
		days []string
		err  error
	)
	dayKeys := requestDayArchiveKeys(options)
	switch len(dayKeys) {
	case 0:
		days, err = s.listArchiveDaysLocked("")
	case 1:
		days, err = s.listArchiveDaysLocked(dayKeys[0])
	default:
		days = append([]string(nil), dayKeys...)
	}
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
			entry, _ := row["entry"].(map[string]any)
			ts := ""
			if entry != nil {
				ts = strings.TrimSpace(asString(entry["timestamp"]))
			}
			if ts == "" {
				ts = strings.TrimSpace(asString(row["ingested_at"]))
			}
			if !requestTimestampMatchesOptions(ts, options) {
				continue
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

func (s *requestStreamSource) countArchiveRowsLocked(options requestQueryOptions) int {
	days, err := s.listArchiveDaysLocked("")
	if err != nil {
		return 0
	}
	total := 0
	for _, day := range days {
		content, readErr := os.ReadFile(filepath.Join(s.archiveRoot, day+".jsonl"))
		if readErr != nil {
			continue
		}
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			var row map[string]any
			if err := json.Unmarshal([]byte(line), &row); err != nil {
				continue
			}
			entry, _ := row["entry"].(map[string]any)
			ts := ""
			if entry != nil {
				ts = strings.TrimSpace(asString(entry["timestamp"]))
			}
			if ts == "" {
				ts = strings.TrimSpace(asString(row["ingested_at"]))
			}
			if requestTimestampMatchesOptions(ts, options) {
				total++
			}
		}
	}
	return total
}
