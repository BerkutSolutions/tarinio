package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSidebarSVGIconsUseBrowserCache(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "nginx.conf"))
	if err != nil {
		t.Fatalf("read UI nginx config: %v", err)
	}
	config := string(content)
	if !strings.Contains(config, "location ^~ /static/icons/svg/") {
		t.Fatal("sidebar SVG icons need a dedicated cache location")
	}
	if !strings.Contains(config, `Cache-Control "public, max-age=604800"`) {
		t.Fatal("sidebar SVG icons need a week-long browser cache policy")
	}
}
