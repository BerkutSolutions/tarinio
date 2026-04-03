package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCandidateStager_StagesBundleIntoDeterministicCandidatePath(t *testing.T) {
	bundle, err := AssembleRevisionBundle(
		RevisionInput{
			ID:        "rev-001",
			Version:   1,
			CreatedAt: "2026-03-31T12:00:00Z",
		},
		[]ArtifactOutput{
			newArtifact("nginx/nginx.conf", ArtifactKindNginxConfig, []byte("worker_processes auto;\n")),
			newArtifact("modsecurity/modsecurity.conf", ArtifactKindModSecurity, []byte("SecRuleEngine Off\n")),
		},
	)
	if err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	root := t.TempDir()
	stager := CandidateStager{Root: root}

	candidatePath, err := stager.Stage(bundle)
	if err != nil {
		t.Fatalf("stage failed: %v", err)
	}

	expected := filepath.Join(root, "candidates", "rev-001")
	if candidatePath != expected {
		t.Fatalf("unexpected candidate path: %s", candidatePath)
	}

	for _, rel := range []string{"manifest.json", filepath.Join("nginx", "nginx.conf"), filepath.Join("modsecurity", "modsecurity.conf")} {
		full := filepath.Join(candidatePath, rel)
		if _, err := os.Stat(full); err != nil {
			t.Fatalf("expected staged file %s: %v", full, err)
		}
	}
}

func TestCandidateStager_ReplacesExistingCandidateContents(t *testing.T) {
	bundle, err := AssembleRevisionBundle(
		RevisionInput{
			ID:        "rev-001",
			Version:   1,
			CreatedAt: "2026-03-31T12:00:00Z",
		},
		[]ArtifactOutput{
			newArtifact("nginx/nginx.conf", ArtifactKindNginxConfig, []byte("worker_processes auto;\n")),
		},
	)
	if err != nil {
		t.Fatalf("assemble failed: %v", err)
	}

	root := t.TempDir()
	candidateRoot := filepath.Join(root, "candidates", "rev-001")
	if err := os.MkdirAll(candidateRoot, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	stalePath := filepath.Join(candidateRoot, "stale.txt")
	if err := os.WriteFile(stalePath, []byte("stale"), 0o644); err != nil {
		t.Fatalf("write stale file failed: %v", err)
	}

	stager := CandidateStager{Root: root}
	stagedPath, err := stager.Stage(bundle)
	if err != nil {
		t.Fatalf("stage failed: %v", err)
	}

	if stagedPath != candidateRoot {
		t.Fatalf("unexpected staged path: %s", stagedPath)
	}
	if _, err := os.Stat(stalePath); !os.IsNotExist(err) {
		t.Fatalf("expected stale file to be removed, got %v", err)
	}
}

func TestCandidateStager_RejectsInvalidBundle(t *testing.T) {
	stager := CandidateStager{Root: t.TempDir()}
	_, err := stager.Stage(&RevisionBundle{
		Revision: RevisionInput{ID: "rev-001"},
		Files: []BundleFile{
			{Path: "nginx/nginx.conf", Content: []byte("worker_processes auto;\n")},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "manifest.json is required") {
		t.Fatalf("expected validation error, got %v", err)
	}
}
