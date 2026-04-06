package appcompat

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureLegacyDataTransferred_CopiesMissingDirs(t *testing.T) {
	runtimeRoot := t.TempDir()
	legacyRoot := filepath.Join(runtimeRoot, "data", "control-plane")
	if err := os.MkdirAll(filepath.Join(legacyRoot, "sites"), 0o755); err != nil {
		t.Fatalf("mkdir legacy sites: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacyRoot, "sites", "site.json"), []byte(`{"id":"site-a"}`), 0o644); err != nil {
		t.Fatalf("write legacy file: %v", err)
	}

	revisionStoreDir := filepath.Join(runtimeRoot, "control-plane")
	if err := EnsureLegacyDataTransferred(runtimeRoot, revisionStoreDir); err != nil {
		t.Fatalf("ensure transfer: %v", err)
	}
	if _, err := os.Stat(filepath.Join(revisionStoreDir, "sites", "site.json")); err != nil {
		t.Fatalf("expected transferred file, stat error: %v", err)
	}
}

func TestScanLegacyLayout_DetectsPendingTransfer(t *testing.T) {
	runtimeRoot := t.TempDir()
	legacyRoot := filepath.Join(runtimeRoot, "storage", "control-plane")
	if err := os.MkdirAll(filepath.Join(legacyRoot, "tlsconfigs"), 0o755); err != nil {
		t.Fatalf("mkdir legacy tlsconfigs: %v", err)
	}
	revisionStoreDir := filepath.Join(runtimeRoot, "control-plane")
	if err := os.MkdirAll(revisionStoreDir, 0o755); err != nil {
		t.Fatalf("mkdir revision store dir: %v", err)
	}

	scan := ScanLegacyLayout(runtimeRoot, revisionStoreDir)
	if len(scan.PendingTransfers) == 0 {
		t.Fatalf("expected pending transfers, got none")
	}
}
