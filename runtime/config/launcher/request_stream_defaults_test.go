package main

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"waf/internal/loggingconfig"
)

func TestRequestStreamMissingSettingsStaysOnLocalFallback(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(root, "access.log")
	archiveRoot := filepath.Join(root, "requests-archive")
	settingsPath := filepath.Join(root, "missing-runtime-settings.json")
	line := `{"timestamp":"2026-06-25T12:00:00Z","request_id":"req-local","client_ip":"203.0.113.10","method":"GET","uri":"/checkout","status":200,"site":"site-a","host":"shop.example.com","upstream_addr":"172.18.0.7:80"}`
	if err := os.WriteFile(logPath, []byte(line+"\n"), 0o644); err != nil {
		t.Fatalf("write log fixture: %v", err)
	}

	source := newRequestStreamSource(
		logPath,
		100,
		archiveRoot,
		30,
		withRequestOpenSearch(settingsPath, "pepper-for-tests"),
		withRequestClickHouse(settingsPath, "pepper-for-tests"),
	)

	settings := source.loadLoggingSettingsLocked()
	if settings.Backend != loggingconfig.BackendFile {
		t.Fatalf("expected file backend fallback, got %q", settings.Backend)
	}
	if settings.Routing.WriteRequestsToHot || settings.Routing.WriteRequestsToCold {
		t.Fatalf("expected backend writes to stay disabled without runtime settings, got hot=%v cold=%v", settings.Routing.WriteRequestsToHot, settings.Routing.WriteRequestsToCold)
	}
	if !settings.Routing.KeepLocalFallback {
		t.Fatal("expected local fallback to stay enabled")
	}

	items, err := source.latest(url.Values{})
	if err != nil {
		t.Fatalf("latest failed with missing settings: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one archive-backed row, got %d", len(items))
	}
	entry, _ := items[0]["entry"].(map[string]any)
	if got := asString(entry["request_id"]); got != "req-local" {
		t.Fatalf("unexpected request id: %s", got)
	}
}
