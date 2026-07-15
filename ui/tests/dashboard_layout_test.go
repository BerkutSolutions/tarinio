package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDashboardRequestsSeriesUsesNarrowerDefaultWidth(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "app", "static", "js", "pages", "dashboard.layout-core.js"))
	if err != nil {
		t.Fatalf("read dashboard layout: %v", err)
	}
	content := string(body)
	for _, marker := range []string{
		`width: 1040`,
		`REQUESTS_SERIES_PREVIOUS_DEFAULT_WIDTH = 1060`,
		`widget.id === "requests-series"`,
	} {
		if !strings.Contains(content, marker) {
			t.Fatalf("expected dashboard layout marker %q", marker)
		}
	}
}
