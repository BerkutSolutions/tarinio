package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServicesWebSocketPatternsAreRegisteredDynamicList(t *testing.T) {
	for _, name := range []string{"sites.geo-lists.js", "sites.geo-list-editors.js"} {
		body, err := os.ReadFile(filepath.Join("..", "app", "static", "js", "pages", name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		content := string(body)
		if strings.Count(content, "ws_block_patterns") < 2 || !strings.Contains(content, "#list-input-ws_block_patterns") {
			t.Fatalf("%s must register WebSocket patterns for list events and settings search", name)
		}
	}
}
