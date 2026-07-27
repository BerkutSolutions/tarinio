package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeoTimeWindowRerenderPreservesTheWholeDraft(t *testing.T) {
	for _, name := range []string{"sites.detail-list-bindings.js", "sites.detail-events-search-lists.js"} {
		content, err := os.ReadFile(filepath.Join("..", "app", "static", "js", "pages", name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		source := strings.ReplaceAll(string(content), "\r\n", "\n")
		addStart := strings.Index(source, `container.querySelector("[data-geo-tw-add]")`)
		removeStart := strings.Index(source, `container.querySelectorAll("[data-geo-tw-remove]")`)
		if addStart < 0 || removeStart <= addStart {
			t.Fatalf("%s: geo time-window handlers are missing", name)
		}
		addBlock := source[addStart:removeStart]
		if syncAt, mutationAt := strings.Index(addBlock, "syncStateDraftFromForm();"), strings.Index(addBlock, "state.draft.geo_time_windows"); syncAt < 0 || mutationAt < 0 || syncAt > mutationAt {
			t.Fatalf("%s: add must capture checkbox and form state before rerender", name)
		}
		removeBlock := source[removeStart:]
		if syncAt, readAt := strings.Index(removeBlock, "syncStateDraftFromForm();"), strings.Index(removeBlock, "readGeoTimeWindowDraftRows(container)"); syncAt < 0 || readAt < 0 || syncAt > readAt {
			t.Fatalf("%s: remove must capture checkbox and form state before rerender", name)
		}
	}
}

func TestDashboardChartUsesPersistentPointerInteractions(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "app", "static", "js", "pages", "dashboard.series.js"))
	if err != nil {
		t.Fatalf("read dashboard series: %v", err)
	}
	source := string(content)
	for _, marker := range []string{
		`import { clamp, formatNumber } from "./dashboard.layout-core.js"`,
		`pointer-events="all" data-chart-overlay="true"`,
		`bodyNode.__wafChartPointer = { clientX: event.clientX, clientY: event.clientY }`,
		`bodyNode.removeEventListener("pointermove", previousHandlers.show)`,
		`bodyNode.addEventListener("pointermove", show)`,
		`window.requestAnimationFrame(() =>`,
		`bodyNode.matches(":hover")`,
	} {
		if !strings.Contains(source, marker) {
			t.Fatalf("dashboard chart missing pointer persistence marker %q", marker)
		}
	}
}

func TestAntiDDoSDetailKeepsCountryIndicatorWithClientIP(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "app", "static", "js", "pages", "antiddos.js"))
	if err != nil {
		t.Fatalf("read anti-ddos page: %v", err)
	}
	source := string(content)
	marker := `countryFlagEmoji(detailsValue(details, "country", "country_code") || item?.country)`
	if strings.Count(source, marker) != 1 {
		t.Fatalf("anti-ddos detail must render exactly one country indicator marker, got %d", strings.Count(source, marker))
	}
}
