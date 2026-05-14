package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"waf/internal/loggingconfig"
)

func (s *requestStreamSource) importArchiveDaysToOpenSearchLocked() error {
	if s.opensearch == nil {
		return nil
	}
	cfg, err := s.opensearch.currentConfig()
	if err != nil {
		return err
	}
	if !cfg.Enabled {
		return nil
	}
	statePath := filepath.Join(s.archiveRoot, requestOpenSearchMigrationStateFile)
	state, err := loadRequestMigrationState(statePath)
	if err != nil {
		return err
	}
	days, err := s.listArchiveDaysLocked("")
	if err != nil {
		return err
	}
	today := time.Now().UTC().Format("2006-01-02")
	for _, day := range days {
		if day >= today {
			continue
		}
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
			if err := os.Remove(filepath.Join(s.archiveRoot, day+".jsonl")); err != nil && !os.IsNotExist(err) {
				return err
			}
			state.ImportedDays[day] = time.Now().UTC().Format(time.RFC3339)
			continue
		}
		if err := s.opensearch.insert(records); err != nil {
			return err
		}
		if err := s.opensearch.containsDayRecords(day, records); err != nil {
			return err
		}
		if err := os.Remove(filepath.Join(s.archiveRoot, day+".jsonl")); err != nil && !os.IsNotExist(err) {
			return err
		}
		state.ImportedDays[day] = time.Now().UTC().Format(time.RFC3339)
	}
	return saveRequestMigrationState(statePath, state)
}

func (s *requestStreamSource) importArchiveDaysToClickHouseLocked() error {
	if s.clickhouse == nil {
		return nil
	}
	cfg, err := s.clickhouse.currentConfig()
	if err != nil || !cfg.Enabled || !cfg.MigrationEnabled {
		return err
	}
	statePath := filepath.Join(s.archiveRoot, requestClickHouseMigrationStateFile)
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
		if err := s.validateClickHouseDayContainsRecords(day, records); err != nil {
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
	statePath := filepath.Join(s.archiveRoot, requestHotToColdMigrationStateFile)
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
			if err := s.validateClickHouseDayContainsRecords(day, records); err != nil {
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

func (s *requestStreamSource) validateClickHouseDayContainsRecords(day string, expected []requestLogRecord) error {
	if s == nil || s.clickhouse == nil || len(expected) == 0 {
		return nil
	}
	records, err := s.clickhouse.exportDay(day)
	if err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(records))
	for _, record := range records {
		if strings.TrimSpace(record.EventHash) == "" {
			continue
		}
		seen[strings.TrimSpace(record.EventHash)] = struct{}{}
	}
	for _, record := range expected {
		hash := strings.TrimSpace(record.EventHash)
		if hash == "" {
			continue
		}
		if _, ok := seen[hash]; !ok {
			return errors.New("clickhouse migration validation failed: missing migrated records")
		}
	}
	return nil
}
