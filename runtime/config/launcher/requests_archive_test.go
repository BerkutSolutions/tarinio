package main

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRequestArchiveLatest_ByDayAndIncremental(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(root, "access.log")
	archiveRoot := filepath.Join(root, "requests-archive")
	source := newRequestStreamSource(logPath, 100, archiveRoot, 30)

	lines := []string{
		`{"timestamp":"2026-04-07T12:10:00Z","request_id":"req-1","client_ip":"1.1.1.1","method":"GET","uri":"/a","status":200,"site":"site-a","host":"a.example.com"}`,
		`{"timestamp":"2026-04-08T12:11:00Z","request_id":"req-2","client_ip":"2.2.2.2","method":"POST","uri":"/b","status":403,"site":"site-b","host":"b.example.com"}`,
	}
	if err := os.WriteFile(logPath, []byte(lines[0]+"\n"+lines[1]+"\n"), 0o644); err != nil {
		t.Fatalf("write log fixture: %v", err)
	}

	dayQuery := url.Values{}
	dayQuery.Set("day", "2026-04-08")
	items, err := source.latest(dayQuery)
	if err != nil {
		t.Fatalf("latest day query failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one row for day filter, got %d", len(items))
	}
	entry, _ := items[0]["entry"].(map[string]any)
	if got := asString(entry["request_id"]); got != "req-2" {
		t.Fatalf("unexpected request id: %s", got)
	}

	appendLine := `{"timestamp":"2026-04-08T12:12:00Z","request_id":"req-3","client_ip":"3.3.3.3","method":"GET","uri":"/c","status":200,"site":"site-b","host":"b.example.com"}`
	file, err := os.OpenFile(logPath, os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		t.Fatalf("open log for append: %v", err)
	}
	_, _ = file.WriteString(appendLine + "\n")
	_ = file.Close()

	items, err = source.latest(dayQuery)
	if err != nil {
		t.Fatalf("latest day query after append failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected two rows for day filter after append, got %d", len(items))
	}
	lastEntry, _ := items[1]["entry"].(map[string]any)
	if got := asString(lastEntry["request_id"]); got != "req-3" {
		t.Fatalf("unexpected appended request id: %s", got)
	}
}

func TestRequestArchivePrunesOldDays(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(root, "access.log")
	archiveRoot := filepath.Join(root, "requests-archive")
	source := newRequestStreamSource(logPath, 100, archiveRoot, 30)

	if err := os.MkdirAll(archiveRoot, 0o755); err != nil {
		t.Fatalf("mkdir archive root: %v", err)
	}
	oldPath := filepath.Join(archiveRoot, "2020-01-01.jsonl")
	if err := os.WriteFile(oldPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write old archive file: %v", err)
	}
	if err := os.WriteFile(logPath, []byte(""), 0o644); err != nil {
		t.Fatalf("write empty log: %v", err)
	}

	query := url.Values{}
	query.Set("retention_days", "1")
	if _, err := source.latest(query); err != nil {
		t.Fatalf("latest with retention failed: %v", err)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("expected old archive file to be removed, got err=%v", err)
	}
}

func TestRequestArchiveFallsBackToTmpWhenPrimaryRootUnavailable(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(root, "access.log")
	blockedPath := filepath.Join(root, "blocked-root")
	if err := os.WriteFile(blockedPath, []byte("blocked"), 0o644); err != nil {
		t.Fatalf("write blocked root fixture: %v", err)
	}
	if err := os.WriteFile(logPath, []byte(""), 0o644); err != nil {
		t.Fatalf("write empty log: %v", err)
	}

	source := newRequestStreamSource(logPath, 100, blockedPath, 30)
	if _, err := source.latest(url.Values{}); err != nil {
		t.Fatalf("latest should fallback to tmp root, got err: %v", err)
	}
	if !strings.Contains(source.archiveRoot, "waf-requests-archive") {
		t.Fatalf("expected fallback archive root, got %s", source.archiveRoot)
	}
}

func TestRequestArchiveDeleteIndex(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(root, "access.log")
	archiveRoot := filepath.Join(root, "requests-archive")
	source := newRequestStreamSource(logPath, 100, archiveRoot, 30)

	if err := os.MkdirAll(archiveRoot, 0o755); err != nil {
		t.Fatalf("mkdir archive root: %v", err)
	}
	target := filepath.Join(archiveRoot, "2026-04-08.jsonl")
	if err := os.WriteFile(target, []byte("{\"entry\":{}}\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	query := url.Values{}
	query.Set("date", "2026-04-08")
	if err := source.deleteIndex(query); err != nil {
		t.Fatalf("delete index failed: %v", err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("expected index file to be removed, got err=%v", err)
	}
}

func TestRequestArchiveFallsBackWhenClickHouseConfigIsMissing(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(root, "access.log")
	archiveRoot := filepath.Join(root, "requests-archive")
	settingsPath := filepath.Join(root, "runtime_settings.json")
	source := newRequestStreamSource(
		logPath,
		100,
		archiveRoot,
		30,
		withRequestClickHouse(settingsPath, "pepper-for-tests"),
	)

	line := `{"timestamp":"2026-04-22T11:09:00Z","request_id":"req-1","client_ip":"1.1.1.1","method":"GET","uri":"/catalog","status":200,"site":"localhost","host":"localhost","upstream_addr":"172.18.0.6:80"}`
	if err := os.WriteFile(logPath, []byte(line+"\n"), 0o644); err != nil {
		t.Fatalf("write log fixture: %v", err)
	}

	items, err := source.latest(url.Values{})
	if err != nil {
		t.Fatalf("latest failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected archive fallback to return one row, got %d", len(items))
	}
	entry, _ := items[0]["entry"].(map[string]any)
	if got := asString(entry["request_id"]); got != "req-1" {
		t.Fatalf("unexpected request id: %s", got)
	}
}

func TestRequestArchiveSkipsInternalManagementTraffic(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(root, "access.log")
	archiveRoot := filepath.Join(root, "requests-archive")
	source := newRequestStreamSource(logPath, 100, archiveRoot, 30)

	lines := []string{
		`{"timestamp":"2026-04-22T11:09:00Z","request_id":"req-ui","client_ip":"127.0.0.1","method":"GET","uri":"/api/requests","status":200,"site":"","host":"localhost"}`,
		`{"timestamp":"2026-04-22T11:10:00Z","request_id":"req-real","client_ip":"1.1.1.1","method":"GET","uri":"/checkout","status":200,"site":"site-a","host":"shop.example.com","upstream_addr":"172.18.0.7:80"}`,
	}
	if err := os.WriteFile(logPath, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("write log fixture: %v", err)
	}

	items, err := source.latest(url.Values{})
	if err != nil {
		t.Fatalf("latest failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one retained row, got %d", len(items))
	}
	entry, _ := items[0]["entry"].(map[string]any)
	if got := asString(entry["request_id"]); got != "req-real" {
		t.Fatalf("unexpected request id: %s", got)
	}
}
