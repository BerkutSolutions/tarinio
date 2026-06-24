package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"waf/internal/loggingconfig"
)

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
	if _, err := os.Stat(s.path); err != nil {
		if os.IsNotExist(err) {
			if err := s.ingestLatestLocked(options.RetentionDays); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	if strings.TrimSpace(s.lastIngestError) != "" {
		return errors.New(strings.TrimSpace(s.lastIngestError))
	}
	return nil
}

func (s *requestStreamSource) startBackgroundIngest(interval time.Duration) {
	if interval <= 0 {
		interval = 3 * time.Second
	}
	ticker := time.NewTicker(interval)
	runIngest := func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if err := s.ensureArchiveRootLocked(); err != nil {
			s.lastIngestError = err.Error()
			return
		}
		if err := s.ingestLatestLocked(s.defaultRetention); err != nil {
			s.lastIngestError = err.Error()
			return
		}
		s.lastIngestError = ""
	}
	go func() {
		runIngest()
		for range ticker.C {
			runIngest()
		}
	}()
}

func (s *requestStreamSource) ingestLatestLocked(retentionDays int) error {
	file, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			if s.clickhouse != nil {
				if importErr := s.importArchiveDaysToClickHouseLocked(); importErr != nil {
					s.lastIngestError = importErr.Error()
				}
			}
			if s.opensearch != nil {
				if importErr := s.importArchiveDaysToOpenSearchLocked(); importErr != nil {
					s.lastIngestError = importErr.Error()
				}
			}
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
	writeToOpenSearch := s.opensearch != nil && len(records) > 0 && ((settings.Routing.WriteRequestsToHot && settings.Hot.Backend == loggingconfig.BackendOpenSearch) || (settings.Routing.WriteRequestsToCold && settings.Cold.Backend == loggingconfig.BackendOpenSearch))
	if writeToOpenSearch {
		if err := s.opensearch.insert(records); err != nil {
			s.lastIngestError = err.Error()
		} else {
			s.lastIngestError = ""
		}
	}
	if s.clickhouse != nil && len(records) > 0 && settings.Routing.WriteRequestsToCold && settings.Cold.Backend == loggingconfig.BackendClickHouse {
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
	if s.opensearch != nil {
		if err := s.importArchiveDaysToOpenSearchLocked(); err != nil {
			s.lastIngestError = err.Error()
		}
	}
	if err := s.migrateHotToColdLocked(retentionDays); err != nil {
		s.lastIngestError = err.Error()
	}
	s.pruneArchiveLocked(retentionDays)
	return nil
}
