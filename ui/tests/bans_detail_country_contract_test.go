package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBanDetailRendersCountryFlagAsTrustedLocalMarkup(t *testing.T) {
	path := filepath.Join("..", "app", "static", "js", "pages", "bans.page-helpers.js")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read ban detail helper: %v", err)
	}
	source := string(content)
	if !strings.Contains(source, `["bans.col.country", countryFlagEmoji(row.country), true]`) {
		t.Fatal("ban detail must mark the local country flag markup as trusted")
	}
	if !strings.Contains(source, `trustedHTML ? String(value || "-") : escapeHtml(String(value || "-"))`) {
		t.Fatal("ban detail must render only trusted country markup without escaping it")
	}
}

func TestBansFilterCoversIPSiteCountryAndModule(t *testing.T) {
	page, err := os.ReadFile(filepath.Join("..", "app", "static", "js", "pages", "bans.js"))
	if err != nil {
		t.Fatalf("read bans page: %v", err)
	}
	rows, err := os.ReadFile(filepath.Join("..", "app", "static", "js", "pages", "bans.page-rows.js"))
	if err != nil {
		t.Fatalf("read bans rows: %v", err)
	}
	if !strings.Contains(string(page), `id="bans-filter"`) || !strings.Contains(string(page), `getFilter:`) {
		t.Fatal("bans page must expose and wire the local filter")
	}
	for _, marker := range []string{"row.ip", "row.country", "row.siteID", "renderModules(row.modules, ctx.t)"} {
		if !strings.Contains(string(rows), marker) {
			t.Fatalf("bans filter must include %s", marker)
		}
	}
}
