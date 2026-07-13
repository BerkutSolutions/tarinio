package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRequestArchiveLatest_ByDayAndIncremental(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(root, "access.log")
	archiveRoot := filepath.Join(root, "requests-archive")
	source := newRequestStreamSource(logPath, 100, archiveRoot, 30)
	day := time.Now().UTC().AddDate(0, 0, -1)
	prevDay := day.AddDate(0, 0, -1)
	dayStr := day.Format("2006-01-02")
	prevDayStr := prevDay.Format("2006-01-02")

	lines := []string{
		fmt.Sprintf(`{"timestamp":"%sT12:10:00Z","request_id":"req-1","client_ip":"1.1.1.1","method":"GET","uri":"/a","status":200,"site":"site-a","host":"a.example.com"}`, prevDayStr),
		fmt.Sprintf(`{"timestamp":"%sT12:11:00Z","request_id":"req-2","client_ip":"2.2.2.2","method":"POST","uri":"/b","status":403,"site":"site-b","host":"b.example.com"}`, dayStr),
	}
	if err := os.WriteFile(logPath, []byte(lines[0]+"\n"+lines[1]+"\n"), 0o644); err != nil {
		t.Fatalf("write log fixture: %v", err)
	}

	dayQuery := url.Values{}
	dayQuery.Set("day", dayStr)
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

	appendLine := fmt.Sprintf(`{"timestamp":"%sT12:12:00Z","request_id":"req-3","client_ip":"3.3.3.3","method":"GET","uri":"/c","status":200,"site":"site-b","host":"b.example.com"}`, dayStr)
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

	stamp := time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339)
	line := fmt.Sprintf(`{"timestamp":"%s","request_id":"req-1","client_ip":"1.1.1.1","method":"GET","uri":"/catalog","status":200,"site":"shop_example","host":"shop.example","upstream_addr":"172.18.0.6:80"}`, stamp)
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

	uiStamp := time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339)
	realStamp := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
	lines := []string{
		fmt.Sprintf(`{"timestamp":"%s","request_id":"req-ui","client_ip":"127.0.0.1","method":"GET","uri":"/api/requests","status":200,"site":"","host":"localhost"}`, uiStamp),
		fmt.Sprintf(`{"timestamp":"%s","request_id":"req-real","client_ip":"1.1.1.1","method":"GET","uri":"/checkout","status":200,"site":"site-a","host":"shop.example.com","upstream_addr":"172.18.0.7:80"}`, realStamp),
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

func TestRequestArchiveProbeDoesNotIngestSynchronously(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(root, "access.log")
	archiveRoot := filepath.Join(root, "requests-archive")
	source := newRequestStreamSource(logPath, 100, archiveRoot, 30)

	stamp := time.Now().UTC().Format(time.RFC3339)
	line := fmt.Sprintf(`{"timestamp":"%s","request_id":"req-probe","client_ip":"1.1.1.1","method":"GET","uri":"/probe","status":200,"site":"site-a","host":"a.example.com"}`, stamp)
	if err := os.WriteFile(logPath, []byte(line+"\n"), 0o644); err != nil {
		t.Fatalf("write log fixture: %v", err)
	}

	if err := source.probe(url.Values{}); err != nil {
		t.Fatalf("probe failed: %v", err)
	}
	if source.lastProcessedOffset != 0 {
		t.Fatalf("expected probe to avoid log ingest, offset=%d", source.lastProcessedOffset)
	}

	items, err := source.latest(url.Values{})
	if err != nil {
		t.Fatalf("latest failed after probe: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one row after latest ingest, got %d", len(items))
	}
}

func TestRequestArchiveIndexesDoNotIngestSynchronously(t *testing.T) {
	root := t.TempDir()
	logPath := filepath.Join(root, "access.log")
	archiveRoot := filepath.Join(root, "requests-archive")
	source := newRequestStreamSource(logPath, 100, archiveRoot, 30)

	day := time.Now().UTC().Format("2006-01-02")
	line := fmt.Sprintf(`{"timestamp":"%sT12:00:00Z","request_id":"req-index","client_ip":"1.1.1.1","method":"GET","uri":"/index","status":200,"site":"site-a","host":"a.example.com"}`, day)
	if err := os.WriteFile(logPath, []byte(line+"\n"), 0o644); err != nil {
		t.Fatalf("write log fixture: %v", err)
	}

	payload, err := source.indexes(url.Values{})
	if err != nil {
		t.Fatalf("indexes failed: %v", err)
	}
	if total := payload["total"]; total != 0 {
		t.Fatalf("expected no synchronous index ingest, total=%v", total)
	}
	if source.lastProcessedOffset != 0 {
		t.Fatalf("expected indexes to avoid log ingest, offset=%d", source.lastProcessedOffset)
	}

	if _, err := source.latest(url.Values{}); err != nil {
		t.Fatalf("latest failed: %v", err)
	}
	payload, err = source.indexes(url.Values{})
	if err != nil {
		t.Fatalf("indexes after latest failed: %v", err)
	}
	if total := payload["total"]; total != 1 {
		t.Fatalf("expected one index after ingest, total=%v", total)
	}
}
