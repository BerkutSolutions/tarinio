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

func TestDashboardPickerAllowsPersistingInitiallyHiddenWidgets(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "app", "static", "js", "pages", "dashboard.layout-core.js"))
	if err != nil {
		t.Fatalf("read dashboard layout: %v", err)
	}
	content := string(body)
	if !strings.Contains(content, `const allowed = new Set(WIDGETS.map((widget) => widget.id))`) {
		t.Fatal("visible-widget persistence must accept every registered widget, including initially hidden widgets")
	}
}

func TestDashboardMemoryLabelAndProgressUseSameClampedPercent(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "app", "static", "js", "pages", "dashboard.series.js"))
	if err != nil {
		t.Fatalf("read dashboard series: %v", err)
	}
	if !strings.Contains(string(body), `const usedPercent = clamp(Number(system.memory_used_percent || 0), 0, 100)`) {
		t.Fatal("memory label and progress must share a clamped percentage")
	}
}

func TestServicesListRenderersGateMutationControlsByWritePermission(t *testing.T) {
	for _, name := range []string{"sites.list-view.js", "sites.view-io.js"} {
		body, err := os.ReadFile(filepath.Join("..", "app", "static", "js", "pages", name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		content := string(body)
		for _, marker := range []string{`permissions.has("sites.write")`, "canWrite ?", "services-delete-selected", "data-toggle-site"} {
			if !strings.Contains(content, marker) {
				t.Fatalf("%s missing read-only mutation gate marker %q", name, marker)
			}
		}
	}
}
