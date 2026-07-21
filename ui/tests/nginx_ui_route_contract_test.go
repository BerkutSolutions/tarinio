package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNginxContract_UnknownUIPathsReturn404(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "nginx.conf"))
	if err != nil {
		t.Fatalf("read nginx config: %v", err)
	}
	config := string(raw)
	if !strings.Contains(config, "profile)(/.*)?$") {
		t.Fatal("known SPA sections must preserve deep-link routing")
	}
	if !strings.Contains(config, "try_files $uri $uri/ =404;") {
		t.Fatal("unknown UI paths must return a controlled 404 instead of index.html")
	}
	if !strings.Contains(config, "location = /healthz {") || !strings.Contains(config, "return 204;") {
		t.Fatal("UI health endpoint must remain available to the control-plane readiness probe")
	}
}
