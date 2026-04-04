package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsVersionGreater(t *testing.T) {
	t.Parallel()

	if !isVersionGreater("4.11.0", "4.10.9") {
		t.Fatal("expected higher patch version to be greater")
	}
	if !isVersionGreater("5.0.0", "4.99.99") {
		t.Fatal("expected higher major version to be greater")
	}
	if isVersionGreater("4.1.0", "4.1.0") {
		t.Fatal("equal versions must not be greater")
	}
	if isVersionGreater("4.0.9", "4.1.0") {
		t.Fatal("lower versions must not be greater")
	}
}

func TestIsValidCRSPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if isValidCRSPath(root) {
		t.Fatal("expected empty directory to be invalid CRS path")
	}
	rulesDir := filepath.Join(root, "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatalf("mkdir rules: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rulesDir, "REQUEST-901-INITIALIZATION.conf"), []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("write rule: %v", err)
	}
	if !isValidCRSPath(root) {
		t.Fatal("expected path with rules/*.conf to be valid CRS path")
	}
}

