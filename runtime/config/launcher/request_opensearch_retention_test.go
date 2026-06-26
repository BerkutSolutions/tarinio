package main

import (
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"waf/internal/loggingconfig"
)

// When both hot and cold are OpenSearch (no ClickHouse migration scheduled),
// pruneOpenSearchOldDaysLocked must still remove indexes older than ColdDays.
func TestRequestStreamPrunesOpenSearchWhenColdIsOpenSearch(t *testing.T) {
	pepper := ""
	now := time.Now().UTC()
	freshStamp := now.AddDate(0, 0, -1).Format("2006-01-02") + "T10:00:00Z"
	staleStamp := now.AddDate(0, 0, -60).Format("2006-01-02") + "T10:00:00Z"

	opensearch := newFakeOpenSearch([]requestLogRecord{
		{
			EventHash: "fresh-hash",
			Stream:    "runtime",
			Timestamp: freshStamp,
			Site:      "waf",
			Host:      "example.com",
		},
		{
			EventHash: "stale-hash",
			Stream:    "runtime",
			Timestamp: staleStamp,
			Site:      "waf",
			Host:      "example.com",
		},
	})
	server := httptest.NewServer(opensearch)
	defer server.Close()

	root := t.TempDir()
	settingsPath := filepath.Join(root, "runtime_settings.json")
	writeRuntimeSettingsFixture(t, settingsPath, pepper, loggingconfig.Settings{
		Backend: loggingconfig.BackendOpenSearch,
		Hot:     loggingconfig.HotSettings{Backend: loggingconfig.BackendOpenSearch},
		Cold:    loggingconfig.ColdSettings{Backend: loggingconfig.BackendOpenSearch},
		Retention: loggingconfig.RetentionSettings{
			HotDays:  14,
			ColdDays: 30,
		},
		Routing: loggingconfig.RoutingSettings{
			WriteRequestsToHot: true,
			KeepLocalFallback:  true,
		},
		OpenSearch: loggingconfig.OpenSearchSettings{
			Endpoint:      server.URL,
			RequestsIndex: "waf-requests",
		},
	})

	source := newRequestStreamSource(
		filepath.Join(root, "missing-access.log"),
		100,
		filepath.Join(root, "archive"),
		14,
		withRequestOpenSearch(settingsPath, pepper),
	)

	if err := source.probe(url.Values{}); err != nil {
		t.Fatalf("probe failed: %v", err)
	}

	if got := opensearch.recordCount(); got != 1 {
		t.Fatalf("expected stale day to be deleted (only fresh row left), got %d records", got)
	}
}

// When cold is ClickHouse and hot is OpenSearch, the prune path falls back to
// HotDays so OpenSearch never holds data past the hot retention horizon, even
// if the migrator missed something.
func TestRequestStreamPrunesOpenSearchByHotDaysWhenColdIsClickHouse(t *testing.T) {
	pepper := ""
	now := time.Now().UTC()
	freshStamp := now.AddDate(0, 0, -1).Format("2006-01-02") + "T10:00:00Z"
	staleStamp := now.AddDate(0, 0, -20).Format("2006-01-02") + "T10:00:00Z"

	clickhouse := newFakeClickHouse()
	clickhouseServer := httptest.NewServer(clickhouse)
	defer clickhouseServer.Close()

	opensearch := newFakeOpenSearch([]requestLogRecord{
		{
			EventHash: "fresh-hash",
			Stream:    "runtime",
			Timestamp: freshStamp,
			Site:      "waf",
			Host:      "example.com",
		},
		{
			EventHash: "stale-hash",
			Stream:    "runtime",
			Timestamp: staleStamp,
			Site:      "waf",
			Host:      "example.com",
		},
	})
	opensearchServer := httptest.NewServer(opensearch)
	defer opensearchServer.Close()

	root := t.TempDir()
	settingsPath := filepath.Join(root, "runtime_settings.json")
	writeRuntimeSettingsFixture(t, settingsPath, pepper, loggingconfig.Settings{
		Backend: loggingconfig.BackendOpenSearch,
		Hot:     loggingconfig.HotSettings{Backend: loggingconfig.BackendOpenSearch},
		Cold:    loggingconfig.ColdSettings{Backend: loggingconfig.BackendClickHouse},
		Retention: loggingconfig.RetentionSettings{
			HotDays:  14,
			ColdDays: 30,
		},
		Routing: loggingconfig.RoutingSettings{
			WriteRequestsToHot: true,
			KeepLocalFallback:  true,
		},
		OpenSearch: loggingconfig.OpenSearchSettings{
			Endpoint:      opensearchServer.URL,
			RequestsIndex: "waf-requests",
		},
		ClickHouse: loggingconfig.ClickHouseSettings{
			Endpoint: clickhouseServer.URL,
			Database: "waf_logs",
			Table:    "request_logs",
		},
	})

	source := newRequestStreamSource(
		filepath.Join(root, "missing-access.log"),
		100,
		filepath.Join(root, "archive"),
		14,
		withRequestOpenSearch(settingsPath, pepper),
		withRequestClickHouse(settingsPath, pepper),
	)

	if err := source.probe(url.Values{}); err != nil {
		t.Fatalf("probe failed: %v", err)
	}

	if got := opensearch.recordCount(); got != 1 {
		t.Fatalf("expected stale opensearch day (older than HotDays=14) to be removed, got %d records", got)
	}
}
