package tests

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDocsSecurityBenchmarkPackContract(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", ".."))

	docs := []string{
		filepath.Join(repoRoot, "docs", "eng", "security-benchmark-pack", "README.md"),
		filepath.Join(repoRoot, "docs", "ru", "security-benchmark-pack", "README.md"),
	}

	requiredMarkers := []string{
		"false_positive_rate",
		"p95",
		"p99",
		"cpu",
		"memory",
		"release-manifest",
		"signature",
		"sbom",
		"provenance",
		"Pass/Fail Criteria",
	}

	for _, path := range docs {
		content := mustReadFile(t, path)
		for _, marker := range requiredMarkers {
			if !strings.Contains(content, marker) {
				t.Fatalf("security benchmark pack doc %s must contain marker %q", path, marker)
			}
		}
	}

	engIndex := mustReadFile(t, filepath.Join(repoRoot, "docs", "eng", "index.md"))
	if !strings.Contains(engIndex, "security-benchmark-pack/README.md") {
		t.Fatalf("docs/eng/index.md must link security benchmark pack")
	}
}
