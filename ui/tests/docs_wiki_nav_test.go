package tests

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestDocsWikiNavCoversAllMarkdownPages(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	docsRoot := filepath.Join(repoRoot, "docs")
	var missing []string

	pluginRoots := map[string]struct{}{
		"eng": {},
		"ru":  {},
	}

	err := filepath.Walk(docsRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}

		relToDocs, err := filepath.Rel(docsRoot, path)
		if err != nil {
			return err
		}
		relToDocs = filepath.ToSlash(relToDocs)

		if isDocusaurusWikiExemptMarkdown(relToDocs) {
			return nil
		}
		top := relToDocs
		if idx := filepath.ToSlash(relToDocs); idx != "" {
			if slash := firstSlash(idx); slash != -1 {
				top = idx[:slash]
			}
		}
		if _, ok := pluginRoots[top]; ok {
			return nil
		}

		missing = append(missing, relToDocs)
		return nil
	})
	if err != nil {
		t.Fatalf("walk docs: %v", err)
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf("docs files are outside configured Docusaurus wiki roots: %v", missing)
	}
}

func isDocusaurusWikiExemptMarkdown(rel string) bool {
	switch rel {
	case "README.md", "CLI_COMMANDS.md", "eng/README.md", "ru/README.md":
		return true
	default:
		if hasPrefixPath(rel, "architecture/") || hasPrefixPath(rel, "operators/") {
			return true
		}
		return false
	}
}

func hasPrefixPath(path string, prefix string) bool {
	if len(path) < len(prefix) {
		return false
	}
	return path[:len(prefix)] == prefix
}

func firstSlash(s string) int {
	for i, ch := range s {
		if ch == '/' {
			return i
		}
	}
	return -1
}
