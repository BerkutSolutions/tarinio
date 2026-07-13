package main

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

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
		items = append(items, requestArchiveIndex{
			Date:      day,
			FileName:  filepath.Base(path),
			SizeBytes: info.Size(),
			Lines:     0,
			UpdatedAt: info.ModTime().UTC().Format(time.RFC3339),
		})
	}

	return map[string]any{
		"items":             items,
		"total":             total,
		"limit":             limit,
		"offset":            offset,
		"archive_root":      s.archiveRoot,
		"storage_type":      "archive",
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
	if s.opensearch != nil {
		if err := s.opensearch.deleteDay(day); err != nil {
			s.lastIngestError = err.Error()
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
	for _, name := range []string{
		requestOpenSearchMigrationStateFile,
		requestClickHouseMigrationStateFile,
		requestHotToColdMigrationStateFile,
	} {
		statePath := filepath.Join(s.archiveRoot, name)
		state, err := loadRequestMigrationState(statePath)
		if err != nil {
			continue
		}
		delete(state.ImportedDays, day)
		_ = saveRequestMigrationState(statePath, state)
	}
	return nil
}
