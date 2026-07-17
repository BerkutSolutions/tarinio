package main

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseRequestQueryOptionsReadsTimezoneOffsetMinutes(t *testing.T) {
	values := url.Values{}
	values.Set("day", "2026-06-25")
	values.Set("tz_offset_minutes", "-180")

	options := parseRequestQueryOptions(values, 100, 14)
	if options.TimezoneOffsetMinutes != -180 {
		t.Fatalf("expected timezone offset -180, got %d", options.TimezoneOffsetMinutes)
	}
}

func TestRequestDayRangeUTCUsesLocalTimezoneOffset(t *testing.T) {
	options := requestQueryOptions{
		Day:                   "2026-06-25",
		TimezoneOffsetMinutes: -180,
	}

	start, end, ok := requestDayRangeUTC(options)
	if !ok {
		t.Fatal("expected day range to be available")
	}
	if got := start.Format(time.RFC3339); got != "2026-06-24T21:00:00Z" {
		t.Fatalf("unexpected range start: %s", got)
	}
	if got := end.Format(time.RFC3339); got != "2026-06-25T21:00:00Z" {
		t.Fatalf("unexpected range end: %s", got)
	}
}

func TestRequestDayArchiveKeysLimitsSinceToRelevantDays(t *testing.T) {
	keys := requestDayArchiveKeys(requestQueryOptions{Since: time.Now().UTC().Add(-24 * time.Hour)})
	if len(keys) == 0 || len(keys) > 2 {
		t.Fatalf("expected one or two archive days for a 24-hour range, got %#v", keys)
	}
}

func TestRequestQueryOptionsClampRemotePagination(t *testing.T) {
	options := parseRequestQueryOptions(url.Values{"limit": {"999999"}, "offset": {"999999999"}}, 100, 14)
	if options.Limit != maxRequestQueryItems || options.Offset != maxRequestQueryOffset {
		t.Fatalf("unexpected clamps: limit=%d offset=%d", options.Limit, options.Offset)
	}
}

func TestRequestDayArchiveKeysClampHistoricSince(t *testing.T) {
	keys := requestDayArchiveKeys(requestQueryOptions{Since: time.Now().UTC().AddDate(0, 0, -365)})
	if len(keys) > maxRequestHistoryDays+1 {
		t.Fatalf("historic probe enumerated too many archive days: %d", len(keys))
	}
}

func TestNormalizeRuntimeRouteUsesFiniteUnknownLabel(t *testing.T) {
	if got := normalizeRuntimeRoute("/untrusted/unique/path"); got != "unknown" {
		t.Fatalf("expected finite unknown label, got %q", got)
	}
}

func TestRequestArchiveLatestRespectsLocalDayTimezoneOffset(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(root, "access.log")
	archiveRoot := filepath.Join(root, "requests-archive")
	source := newRequestStreamSource(logPath, 100, archiveRoot, 30)

	lines := []byte(
		"{\"timestamp\":\"2026-06-24T21:30:00Z\",\"request_id\":\"req-local\",\"client_ip\":\"1.1.1.1\",\"method\":\"GET\",\"uri\":\"/early\",\"status\":200,\"site\":\"logs_example_test\",\"host\":\"logs.example.test\"}\n" +
			"{\"timestamp\":\"2026-06-25T21:30:00Z\",\"request_id\":\"req-next-day\",\"client_ip\":\"1.1.1.2\",\"method\":\"GET\",\"uri\":\"/late\",\"status\":200,\"site\":\"ui_example_test\",\"host\":\"ui.example.test\"}\n",
	)
	if err := os.WriteFile(logPath, lines, 0o644); err != nil {
		t.Fatalf("write log fixture: %v", err)
	}

	query := url.Values{}
	query.Set("day", "2026-06-25")
	query.Set("tz_offset_minutes", "-180")
	items, err := source.latest(query)
	if err != nil {
		t.Fatalf("latest failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one row for local day window, got %d", len(items))
	}
	entry, _ := items[0]["entry"].(map[string]any)
	if got := asString(entry["request_id"]); got != "req-local" {
		t.Fatalf("unexpected request id: %s", got)
	}
}
