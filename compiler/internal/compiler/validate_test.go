package compiler

import (
	"strings"
	"testing"
)

func TestValidateRevisionBundle_SucceedsForDeterministicBundle(t *testing.T) {
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

	if err := ValidateRevisionBundle(bundle); err != nil {
		t.Fatalf("expected bundle to validate, got %v", err)
	}
}

func TestValidateRevisionBundle_RequiresManifest(t *testing.T) {
	err := ValidateRevisionBundle(&RevisionBundle{
		Files: []BundleFile{
			{Path: "nginx/nginx.conf", Content: []byte("worker_processes auto;\n")},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "manifest.json is required") {
		t.Fatalf("expected missing manifest error, got %v", err)
	}
}

func TestValidateRevisionBundle_RejectsMissingArtifact(t *testing.T) {
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

	bundle.Files = bundle.Files[1:]
	err = ValidateRevisionBundle(bundle)
	if err == nil || !strings.Contains(err.Error(), "is missing from bundle") {
		t.Fatalf("expected missing artifact error, got %v", err)
	}
}

func TestValidateRevisionBundle_RejectsChecksumMismatch(t *testing.T) {
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

	for i := range bundle.Files {
		if bundle.Files[i].Path == "nginx/nginx.conf" {
			bundle.Files[i].Content = []byte("tampered\n")
		}
	}

	err = ValidateRevisionBundle(bundle)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected checksum mismatch error, got %v", err)
	}
}

func TestValidateRevisionBundle_RejectsBundleChecksumMismatch(t *testing.T) {
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

	for i := range bundle.Files {
		if bundle.Files[i].Path == "manifest.json" {
			bundle.Files[i].Content = []byte(strings.Replace(string(bundle.Files[i].Content), bundle.Manifest.BundleChecksum, "badchecksum", 1))
			break
		}
	}

	err = ValidateRevisionBundle(bundle)
	if err == nil || !strings.Contains(err.Error(), "bundle checksum mismatch") {
		t.Fatalf("expected bundle checksum mismatch error, got %v", err)
	}
}
