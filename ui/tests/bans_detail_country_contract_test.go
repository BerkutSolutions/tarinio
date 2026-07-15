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
