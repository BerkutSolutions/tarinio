package compiler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicActivator_WritesDeterministicActivePointer(t *testing.T) {
	root := t.TempDir()
	candidatePath := filepath.Join(root, "candidates", "rev-001")
	if err := os.MkdirAll(candidatePath, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(candidatePath, "manifest.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write manifest failed: %v", err)
	}

	activator := AtomicActivator{Root: root}
	pointer, err := activator.Activate("rev-001")
	if err != nil {
		t.Fatalf("activate failed: %v", err)
	}

	if pointer.RevisionID != "rev-001" {
		t.Fatalf("unexpected revision id: %s", pointer.RevisionID)
	}

	content, err := os.ReadFile(filepath.Join(root, "active", "current.json"))
	if err != nil {
		t.Fatalf("read active pointer failed: %v", err)
	}

	var reloaded ActivePointer
	if err := json.Unmarshal(content, &reloaded); err != nil {
		t.Fatalf("unmarshal active pointer failed: %v", err)
	}
	if reloaded.RevisionID != "rev-001" {
		t.Fatalf("unexpected stored revision id: %s", reloaded.RevisionID)
	}
	if reloaded.CandidatePath != candidatePath {
		t.Fatalf("unexpected candidate path: %s", reloaded.CandidatePath)
	}
}

func TestAtomicActivator_ReplacesExistingActivePointer(t *testing.T) {
	root := t.TempDir()
	for _, revisionID := range []string{"rev-001", "rev-002"} {
		candidatePath := filepath.Join(root, "candidates", revisionID)
		if err := os.MkdirAll(candidatePath, 0o755); err != nil {
			t.Fatalf("mkdir failed: %v", err)
		}
		if err := os.WriteFile(filepath.Join(candidatePath, "manifest.json"), []byte("{}\n"), 0o644); err != nil {
			t.Fatalf("write manifest failed: %v", err)
		}
	}

	activator := AtomicActivator{Root: root}
	if _, err := activator.Activate("rev-001"); err != nil {
		t.Fatalf("first activation failed: %v", err)
	}
	if _, err := activator.Activate("rev-002"); err != nil {
		t.Fatalf("second activation failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(root, "active", "current.json"))
	if err != nil {
		t.Fatalf("read active pointer failed: %v", err)
	}

	var reloaded ActivePointer
	if err := json.Unmarshal(content, &reloaded); err != nil {
		t.Fatalf("unmarshal active pointer failed: %v", err)
	}
	if reloaded.RevisionID != "rev-002" {
		t.Fatalf("expected rev-002 to be active, got %s", reloaded.RevisionID)
	}
}

func TestAtomicActivator_RejectsMissingCandidate(t *testing.T) {
	activator := AtomicActivator{Root: t.TempDir()}
	if _, err := activator.Activate("rev-001"); err == nil {
		t.Fatal("expected missing candidate error")
	}
}
