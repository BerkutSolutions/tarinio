package main

import (
	"time"
)

const maxTimezoneOffsetMinutes = 14 * 60

func normalizeTimezoneOffsetMinutes(offset int) int {
	if offset > maxTimezoneOffsetMinutes {
		return maxTimezoneOffsetMinutes
	}
	if offset < -maxTimezoneOffsetMinutes {
		return -maxTimezoneOffsetMinutes
	}
	return offset
}

func requestDayRangeUTC(options requestQueryOptions) (time.Time, time.Time, bool) {
	if options.Day == "" {
		return time.Time{}, time.Time{}, false
	}
	parsedDay, err := time.Parse("2006-01-02", options.Day)
	if err != nil {
		return time.Time{}, time.Time{}, false
	}
	offset := time.Duration(normalizeTimezoneOffsetMinutes(options.TimezoneOffsetMinutes)) * time.Minute
	start := parsedDay.Add(offset).UTC()
	end := parsedDay.Add(24 * time.Hour).Add(offset).UTC()
	return start, end, true
}

func requestDayArchiveKeys(options requestQueryOptions) []string {
	start, end, ok := requestDayRangeUTC(options)
	if !ok {
		if options.Day != "" {
			return []string{options.Day}
		}
		if options.Since.IsZero() {
			return nil
		}
		start = time.Date(options.Since.UTC().Year(), options.Since.UTC().Month(), options.Since.UTC().Day(), 0, 0, 0, 0, time.UTC)
		end = time.Now().UTC()
	}
	keys := make([]string, 0, 2)
	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		keys = append(keys, day.Format("2006-01-02"))
	}
	return keys
}

func requestTimestampMatchesOptions(value string, options requestQueryOptions) bool {
	if !options.Since.IsZero() {
		parsed, ok := parseRequestTimestamp(value)
		if ok && parsed.Before(options.Since) {
			return false
		}
	}
	start, end, ok := requestDayRangeUTC(options)
	if !ok {
		return true
	}
	parsed, ok := parseRequestTimestamp(value)
	if !ok {
		return false
	}
	return !parsed.Before(start) && parsed.Before(end)
}

func parseRequestTimestamp(value string) (time.Time, bool) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339, value)
		if err != nil {
			return time.Time{}, false
		}
	}
	return parsed.UTC(), true
}
