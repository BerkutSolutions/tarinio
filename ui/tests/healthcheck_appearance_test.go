package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHealthcheckAppearanceUsesLiveChecksForEveryVariant(t *testing.T) {
	files := []string{
		filepath.Join("..", "app", "healthcheck.html"),
		filepath.Join("..", "app", "static", "js", "healthcheck.js"),
		filepath.Join("..", "app", "static", "js", "healthcheck-appearance.js"),
		filepath.Join("..", "app", "static", "healthcheck-appearance.css"),
		filepath.Join("..", "app", "static", "healthcheck-appearance-fixes.css"),
		filepath.Join("..", "app", "static", "js", "pages", "settings.js"),
	}
	content := make([]string, 0, len(files))
	for _, file := range files {
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		content = append(content, string(body))
	}
	joined := strings.Join(content, "\n")
	for _, marker := range []string{
		`id="healthcheck-app"`,
		`applyHealthcheckAppearance`,
		`/api/public/healthcheck-appearance`,
		`variant-1`, `variant-2`, `variant-3`, `variant-4`, `variant-5`,
		`/healthcheck?appearance=${theme}`,
		`await runChecks()`, `await loadErrorIssues()`, `await loadCompat()`,
		`hc-page-actions`, `hc-console`, `appearance !== "variant-5"`,
		`lines.slice(-12).join("\n")`,
		`grid-template-columns:230px minmax(0,1fr)`, `Hard width containment`,
		`contain:inline-size`, `hc-errors .hc-panel`,
	} {
		if !strings.Contains(joined, marker) {
			t.Fatalf("expected healthcheck appearance contract marker %q", marker)
		}
	}
}
