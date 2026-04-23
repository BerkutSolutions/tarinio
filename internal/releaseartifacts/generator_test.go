package releaseartifacts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateReleaseArtifacts(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	outputDir := filepath.Join(t.TempDir(), "release-2.0.7")

	result, err := Generate(Options{
		RepoRoot:   repoRoot,
		Version:    "2.0.7",
		CommitSHA:  "deadbeef",
		Tag:        "v2.0.7",
		OutputDir:  outputDir,
		DockerTags: []string{"tarinio:2.0.7", "ghcr.io/berkutsolutions/tarinio:2.0.7"},
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	required := []string{
		result.ManifestPath,
		result.SignaturePath,
		result.SBOMPath,
		result.ProvenancePath,
		result.PublicKeyPath,
		filepath.Join(outputDir, "checksums.txt"),
	}
	for _, path := range required {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected artifact %s: %v", path, err)
		}
	}

	var manifest map[string]any
	if err := decodeJSONFile(result.ManifestPath, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	if got := manifest["format"]; got != "tarinio-release-artifacts/v1" {
		t.Fatalf("manifest format = %v", got)
	}
	if got := manifest["version"]; got != "2.0.7" {
		t.Fatalf("manifest version = %v", got)
	}
	generatedFiles, ok := manifest["generated_files"].([]any)
	if !ok || len(generatedFiles) < 4 {
		t.Fatalf("manifest generated_files missing or too short: %#v", manifest["generated_files"])
	}

	var sbom map[string]any
	if err := decodeJSONFile(result.SBOMPath, &sbom); err != nil {
		t.Fatalf("decode sbom: %v", err)
	}
	if got := sbom["bomFormat"]; got != "CycloneDX" {
		t.Fatalf("sbom bomFormat = %v", got)
	}
}

func decodeJSONFile(path string, target any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}
