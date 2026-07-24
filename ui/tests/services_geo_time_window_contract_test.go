package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServicesGeoTimeWindowsReadRenderedRows(t *testing.T) {
	for _, name := range []string{"sites.detail-draft-builder.js", "sites.detail-list-bindings.js", "sites.detail-events-search-lists.js"} {
		body, err := os.ReadFile(filepath.Join("..", "app", "static", "js", "pages", name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if !strings.Contains(string(body), "readGeoTimeWindowDraftRows(container)") {
			t.Fatalf("%s must read current Geo time-window DOM rows", name)
		}
	}
}
