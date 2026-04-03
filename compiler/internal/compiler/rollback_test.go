package compiler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRollbackRunner_RollsBackToKnownGoodRevision(t *testing.T) {
	root := t.TempDir()
	for _, revisionID := range []string{"rev-good", "rev-bad"} {
		candidatePath := filepath.Join(root, "candidates", revisionID)
		if err := os.MkdirAll(candidatePath, 0o755); err != nil {
			t.Fatalf("mkdir failed: %v", err)
		}
		if err := os.WriteFile(filepath.Join(candidatePath, "manifest.json"), []byte("{}\n"), 0o644); err != nil {
			t.Fatalf("write manifest failed: %v", err)
		}
	}

	runner := RollbackRunner{
		Activator: AtomicActivator{Root: root},
	}
	result, err := runner.Rollback("rev-bad", &ActivePointer{
		RevisionID:    "rev-good",
		CandidatePath: filepath.Join(root, "candidates", "rev-good"),
	})
	if err != nil {
		t.Fatalf("rollback failed: %v", err)
	}
	if !result.Succeeded {
		t.Fatal("expected rollback success")
	}
	if result.RolledBackTo != "rev-good" {
		t.Fatalf("unexpected rollback target: %s", result.RolledBackTo)
	}
	if result.ActivationPointer == nil || result.ActivationPointer.RevisionID != "rev-good" {
		t.Fatal("expected activation pointer to target known-good revision")
	}
}

func TestRollbackRunner_RejectsSameFailedAndKnownGoodRevision(t *testing.T) {
	runner := RollbackRunner{
		Activator: AtomicActivator{Root: t.TempDir()},
	}
	_, err := runner.Rollback("rev-001", &ActivePointer{
		RevisionID:    "rev-001",
		CandidatePath: filepath.Join("candidates", "rev-001"),
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestRollbackRunner_PropagatesActivationFailure(t *testing.T) {
	runner := RollbackRunner{
		Activator: AtomicActivator{Root: t.TempDir()},
	}
	_, err := runner.Rollback("rev-bad", &ActivePointer{
		RevisionID:    "rev-good",
		CandidatePath: filepath.Join("candidates", "rev-good"),
	})
	if err == nil {
		t.Fatal("expected activation failure")
	}
}
