package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseThreatIntelLine(t *testing.T) {
	target, score, label := parseThreatIntelLine("203.0.113.10,90,scanner")
	if target != "203.0.113.10" || score != 90 || label != "scanner" {
		t.Fatalf("unexpected csv parse result: %q %d %q", target, score, label)
	}

	target, score, label = parseThreatIntelLine("198.51.100.0/24 blocklist")
	if target != "198.51.100.0/24" || score != 100 || label != "blocklist" {
		t.Fatalf("unexpected spaced parse result: %q %d %q", target, score, label)
	}
}

func TestThreatIntelMatcherMatchesIPAndCIDR(t *testing.T) {
	root := t.TempDir()
	feedPath := filepath.Join(root, "feed.txt")
	feedBody := "" +
		"# comment\n" +
		"203.0.113.10,95,scanner\n" +
		"198.51.100.0/24 80 reputation\n"
	if err := os.WriteFile(feedPath, []byte(feedBody), 0o644); err != nil {
		t.Fatalf("write feed: %v", err)
	}

	matcher := &threatIntelMatcher{
		feeds:           []string{feedPath},
		refreshInterval: time.Hour,
		ipHits:          map[string]threatIntelHit{},
		cidrHit:         []threatIntelCIDR{},
	}

	ipHit, ok := matcher.Match("203.0.113.10")
	if !ok {
		t.Fatal("expected exact IP to match")
	}
	if ipHit.Score != 95 || ipHit.Label != "scanner" {
		t.Fatalf("unexpected exact IP hit: %+v", ipHit)
	}

	cidrHit, ok := matcher.Match("198.51.100.77")
	if !ok {
		t.Fatal("expected CIDR to match")
	}
	if cidrHit.Score != 80 || cidrHit.Label != "reputation" {
		t.Fatalf("unexpected cidr hit: %+v", cidrHit)
	}
}
