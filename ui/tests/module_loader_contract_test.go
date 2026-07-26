package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUIContract_DynamicModulesUseBoundedBackoff(t *testing.T) {
	app, err := os.ReadFile(filepath.Join("..", "app", "static", "js", "app.js"))
	if err != nil {
		t.Fatalf("read app.js: %v", err)
	}
	source := string(app)
	for _, marker := range []string{
		"const PAGE_MODULE_RETRY_DELAYS_MS = [250, 750];",
		"window.setTimeout(resolve, PAGE_MODULE_RETRY_DELAYS_MS[index])",
	} {
		if !strings.Contains(source, marker) {
			t.Fatalf("app.js missing module-load resilience marker %q", marker)
		}
	}

	index, err := os.ReadFile(filepath.Join("..", "app", "index.html"))
	if err != nil {
		t.Fatalf("read index.html: %v", err)
	}
	if !strings.Contains(string(index), "app.js?v=20260726-module-backoff-1") {
		t.Fatal("index.html must invalidate the cached module loader")
	}
}
