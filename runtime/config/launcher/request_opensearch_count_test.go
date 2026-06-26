package main

import (
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"waf/internal/loggingconfig"
)

func TestRequestOpenSearchCountTracksTotalsAboveTenThousand(t *testing.T) {
	records := make([]requestLogRecord, 0, 12050)
	base := time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 12050; i++ {
		stamp := base.Add(-time.Duration(i) * time.Second).Format(time.RFC3339)
		records = append(records, requestLogRecord{
			EventHash:  "hash-" + strconvString(i),
			Timestamp:  stamp,
			IngestedAt: stamp,
			ClientIP:   "203.0.113.10",
			Method:     "GET",
			URI:        "/checkout",
			Status:     200,
			Site:       "site-a",
			Host:       "shop.example.com",
		})
	}

	server := httptest.NewServer(newFakeOpenSearch(records))
	defer server.Close()

	root := t.TempDir()
	settingsPath := filepath.Join(root, "runtime_settings.json")
	writeRuntimeSettingsFixture(t, settingsPath, "", loggingconfig.Settings{
		Backend: loggingconfig.BackendOpenSearch,
		Hot: loggingconfig.HotSettings{
			Backend: loggingconfig.BackendOpenSearch,
		},
		Cold: loggingconfig.ColdSettings{
			Backend: loggingconfig.BackendOpenSearch,
		},
		OpenSearch: loggingconfig.OpenSearchSettings{
			Endpoint:      server.URL,
			RequestsIndex: "waf-requests",
		},
	})

	store := newRequestOpenSearchStore(settingsPath, "")
	count, err := store.count(requestQueryOptions{})
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != len(records) {
		t.Fatalf("expected count=%d, got %d", len(records), count)
	}
}
