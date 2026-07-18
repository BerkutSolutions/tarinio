package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRequestDashboardSummaryUsesAllFilteredArchiveRows(t *testing.T) {
	root := t.TempDir()
	archiveRoot := filepath.Join(root, "requests-archive")
	if err := os.MkdirAll(archiveRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Hour)
	day := now.Format("2006-01-02")
	lines := []string{
		requestDashboardFixture(now.Add(-time.Hour), "normal", "200", "shop", "shop.example", "/catalog", "203.0.113.10", ""),
		requestDashboardFixture(now, "blocked-a", "403", "shop", "shop.example", "/admin", "203.0.113.20", "DE"),
		requestDashboardFixture(now, "blocked-b", "429", "blog", "blog.example", "/search", "203.0.113.20", "DE"),
		requestDashboardFixture(now, "panel", "403", "ui", "localhost", "/api/dashboard/stats", "127.0.0.1", ""),
	}
	if err := os.WriteFile(filepath.Join(archiveRoot, day+".jsonl"), []byte(joinRequestArchiveLines(lines)), 0o644); err != nil {
		t.Fatal(err)
	}
	source := newRequestStreamSource(filepath.Join(root, "access.log"), 100, archiveRoot, 30)
	summary, err := source.dashboardSummary(url.Values{"since": []string{now.Add(-2 * time.Hour).Format(time.RFC3339)}})
	if err != nil {
		t.Fatalf("dashboard summary: %v", err)
	}
	if summary.RequestsDay != 3 || summary.UniqueIPsDay != 2 {
		t.Fatalf("unexpected request totals: %+v", summary)
	}
	if summary.BlockedDay != 2 || summary.AttacksDay != 2 || summary.UniqueAttackerIPs != 1 {
		t.Fatalf("unexpected attack totals: %+v", summary)
	}
	if got := summary.TopSites[0]; got.Key != "shop" || got.Count != 2 {
		t.Fatalf("top sites must include filtered archive rows, got %+v", summary.TopSites)
	}
	if got := summary.TopAttackerIPs[0]; got.Key != "203.0.113.20" || got.Count != 2 || got.Country != "DE" {
		t.Fatalf("unexpected top attacker IPs: %+v", summary.TopAttackerIPs)
	}
	if got := summary.TopCountries[0]; got.Key != "DE" || got.Count != 2 {
		t.Fatalf("unexpected countries: %+v", summary.TopCountries)
	}
}

func requestDashboardFixture(at time.Time, requestID, status, site, host, uri, ip, country string) string {
	return fmt.Sprintf(`{"ingested_at":"%s","entry":{"timestamp":"%s","request_id":"%s","client_ip":"%s","country":"%s","uri":"%s","status":%s,"site":"%s","host":"%s"}}`, at.Format(time.RFC3339), at.Format(time.RFC3339), requestID, ip, country, uri, status, site, host)
}

func joinRequestArchiveLines(lines []string) string {
	value := ""
	for _, line := range lines {
		value += line + "\n"
	}
	return value
}
