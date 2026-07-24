package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServicesCompactFormGridCanShrinkOnNarrowScreens(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "app", "static", "waf.css"))
	if err != nil {
		t.Fatalf("read waf.css: %v", err)
	}
	content := string(body)
	marker := ".waf-service-compact-section .waf-form-grid {\n    grid-template-columns: minmax(0, 1fr);"
	if !strings.Contains(content, marker) {
		t.Fatal("mobile Services form grid must override the 240px compact-section minimum")
	}
	if count := strings.Count(content, `repeat(auto-fit, minmax(min(220px, 100%), 1fr))`); count < 2 {
		t.Fatalf("Geo and ModSecurity grids must shrink responsively, got %d adaptive rules", count)
	}
}
