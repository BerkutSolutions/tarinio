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

func TestDashboardMemoryLabelAndProgressUseSameContainerAggregate(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "app", "static", "js", "pages", "dashboard.series.js"))
	if err != nil {
		t.Fatalf("read dashboard series: %v", err)
	}
	for _, marker := range []string{
		`const usedPercent = clamp(Number(containersOverview?.avg_memory_percent || 0), 0, 100)`,
		`dashboard-system-container-row`,
	} {
		if !strings.Contains(string(body), marker) {
			t.Fatalf("memory widget must render container overview marker %q", marker)
		}
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
