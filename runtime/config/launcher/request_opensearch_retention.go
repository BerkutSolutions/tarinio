package main

import (
	"errors"
	"time"

	"waf/internal/loggingconfig"
)

// pruneOpenSearchOldDaysLocked deletes day buckets from the OpenSearch hot index
// that are older than the effective retention horizon. It is the only automatic
// cleanup path when both hot and cold storage point at OpenSearch (no migration
// is scheduled because the migrator only runs for cold=ClickHouse).
//
// The retention horizon is derived from settings:
//   - cold=OpenSearch: all data lives in OpenSearch, horizon = ColdDays.
//   - otherwise:       OpenSearch holds only hot data (cold goes to ClickHouse
//                      or files), horizon = HotDays. This also acts as a safety
//                      net in case migrateHotToColdLocked falls behind.
//
// The hint parameter is consulted only when settings produce a non-positive
// horizon, so a caller can still override behaviour for tests.
func (s *requestStreamSource) pruneOpenSearchOldDaysLocked(hint int) error {
	if s == nil || s.opensearch == nil {
		return nil
	}
	settings := s.loadLoggingSettingsLocked()
	if settings.Hot.Backend != loggingconfig.BackendOpenSearch &&
		settings.Cold.Backend != loggingconfig.BackendOpenSearch {
		return nil
	}

	horizon := 0
	if settings.Cold.Backend == loggingconfig.BackendOpenSearch {
		horizon = settings.Retention.ColdDays
	} else {
		horizon = settings.Retention.HotDays
	}
	if horizon <= 0 {
		horizon = hint
	}
	if horizon <= 0 {
		horizon = loggingconfig.DefaultHotDays
	}

	days, err := s.opensearch.days()
	if err != nil {
		if errors.Is(err, errOpenSearchDisabled) {
			return nil
		}
		return err
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -horizon)
	for _, day := range days {
		parsedDay, parseErr := time.Parse("2006-01-02", day)
		if parseErr != nil {
			continue
		}
		if !parsedDay.Before(cutoff) {
			continue
		}
		if err := s.opensearch.deleteDay(day); err != nil {
			return err
		}
	}
	return nil
}
