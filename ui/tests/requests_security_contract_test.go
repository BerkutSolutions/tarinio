package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRequestsSecurityReasonFilterAndCleanDetailsContract(t *testing.T) {
	files := []string{
		filepath.Join("..", "app", "static", "js", "pages", "requests.js"),
		filepath.Join("..", "app", "static", "js", "pages", "requests.security.js"),
		filepath.Join("..", "app", "static", "i18n", "ru.json"),
		filepath.Join("..", "app", "static", "i18n", "en.json"),
		filepath.Join("..", "app", "static", "i18n", "de.json"),
		filepath.Join("..", "app", "static", "i18n", "sr.json"),
		filepath.Join("..", "app", "static", "i18n", "zh.json"),
	}
	markers := []string{
		`normalizeSecurityReason`,
		`buildRequestDetailsMeta(row, ctx)`,
		`selectedSecurityReason`,
		`requests-filter-security-reason`,
		`requests.filter.securityReason`,
		`requests.detail.securityReason`,
		`buildSecurityDetailSummary(row, ctx)`,
		`normalizedSecurityEventType(row)`,
		`row?.event_type`,
		`requests.securityReason.accessBlocked`,
		`row?.security_reason`,
		`requests.securityReason.modsecurity`,
		`requests.securityReason.rateLimit`,
		`requests.securityReason.threatIntel`,
		`requests.securityReason.scanner`,
		`requests.securityReason.geo`,
		`requests.securityReason.auth`,
		`requests.securityReason.challenge`,
		`buildSecurityBadge(row, ctx, escapeHtml)`,
	}
	contents := make([]string, 0, len(files))
	for _, file := range files {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		contents = append(contents, string(raw))
	}
	for _, marker := range markers {
		found := false
		for _, content := range contents {
			if strings.Contains(content, marker) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing requests compatibility marker %s in %v", marker, files)
		}
	}
	for _, forbidden := range []string{"legacy_row_type_support", "legacyCompatibility", "requests.detail.securitySummary"} {
		for _, content := range contents {
			if strings.Contains(content, forbidden) {
				t.Fatalf("requests UI must not expose %s", forbidden)
			}
		}
	}
}
