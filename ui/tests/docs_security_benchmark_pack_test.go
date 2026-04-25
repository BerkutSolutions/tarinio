package tests

import (
	"os"
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

	for _, path := range docs {
		if _, err := filepath.Abs(path); err != nil {
			t.Fatalf("resolve %s: %v", path, err)
		}
		if _, err := os.Stat(path); err == nil {
			t.Fatalf("security benchmark pack doc %s must be removed", path)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", path, err)
		}
	}

	engIndex := mustReadFile(t, filepath.Join(repoRoot, "docs", "eng", "index.md"))
	if strings.Contains(engIndex, "security-benchmark-pack/README.md") {
		t.Fatalf("docs/eng/index.md must not link removed security benchmark pack docs")
	}
}
