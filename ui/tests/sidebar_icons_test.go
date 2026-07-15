package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSidebarUsesUploadedSVGAssets(t *testing.T) {
	appJS, err := os.ReadFile(filepath.Join("..", "app", "static", "js", "app.js"))
	if err != nil {
		t.Fatalf("read sidebar script: %v", err)
	}
	content := string(appJS)
	for _, name := range []string{"dashboard", "services", "antiddos", "owasp", "certificates", "requests", "revisions", "journal", "incidents", "bans", "administration", "audit", "settings", "user"} {
		asset := filepath.Join("..", "app", "static", "icons", "svg", name+"-32x32.svg")
		if _, err := os.Stat(asset); err != nil {
			t.Fatalf("missing sidebar SVG %q: %v", asset, err)
		}
		if !strings.Contains(content, `sidebarIcon("`+name+`")`) {
			t.Fatalf("sidebar does not use uploaded SVG %q", name)
		}
	}
}
