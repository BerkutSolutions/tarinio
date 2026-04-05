package tests

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"unicode/utf8"
)

var (
	textExtensions = map[string]struct{}{
		".md":         {},
		".go":         {},
		".js":         {},
		".json":       {},
		".html":       {},
		".css":        {},
		".yml":        {},
		".yaml":       {},
		".tmpl":       {},
		".conf":       {},
		".txt":        {},
		".ps1":        {},
		".sh":         {},
		".sql":        {},
		".toml":       {},
		".mod":        {},
		".sum":        {},
		".dockerfile": {},
	}
	textBasenames = map[string]struct{}{
		"Dockerfile":          {},
		"docker-compose.yml":  {},
		"docker-compose.yaml": {},
		"README.md":           {},
		"README.en.md":        {},
		"CHANGELOG.md":        {},
		".env":                {},
		".env.example":        {},
		"Makefile":            {},
	}
	bomSensitiveExtensions = map[string]struct{}{
		".go":   {},
		".js":   {},
		".json": {},
		".html": {},
		".css":  {},
		".yml":  {},
		".yaml": {},
		".tmpl": {},
		".conf": {},
		".sh":   {},
		".ps1":  {},
	}
	skipDirs = map[string]struct{}{
		".git":         {},
		".work":        {},
		"node_modules": {},
		"bin":          {},
	}
)

func TestRepositoryTextFilesEncoding(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	var checked int
	var broken []string

	err := filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if _, ok := skipDirs[info.Name()]; ok {
				return filepath.SkipDir
			}
			return nil
		}

		if !isTextFile(path) {
			return nil
		}

		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			broken = append(broken, rel(repoRoot, path)+": read error: "+readErr.Error())
			return nil
		}
		checked++

		if hasUTF8BOM(raw) && isBOMSensitive(path) {
			broken = append(broken, rel(repoRoot, path)+": contains UTF-8 BOM")
		}
		if !utf8.Valid(raw) {
			broken = append(broken, rel(repoRoot, path)+": invalid UTF-8")
			return nil
		}

		content := string(raw)
		if !strings.Contains(filepath.ToSlash(path), "/ui/tests/") && (strings.ContainsRune(content, '\uFFFD') || hasMojibakeMarker(content)) {
			broken = append(broken, rel(repoRoot, path)+": mojibake/replacement marker found")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk repository: %v", err)
	}

	if checked == 0 {
		t.Fatalf("repository encoding test checked 0 files")
	}
	if len(broken) > 0 {
		sort.Strings(broken)
		t.Fatalf("repository encoding validation failed (%d files checked): %v", checked, sample(broken))
	}
}

func isTextFile(path string) bool {
	base := filepath.Base(path)
	if _, ok := textBasenames[base]; ok {
		return true
	}
	ext := strings.ToLower(filepath.Ext(base))
	_, ok := textExtensions[ext]
	return ok
}

func hasUTF8BOM(raw []byte) bool {
	return len(raw) >= 3 && raw[0] == 0xEF && raw[1] == 0xBB && raw[2] == 0xBF
}

func isBOMSensitive(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if _, ok := bomSensitiveExtensions[ext]; ok {
		return true
	}
	base := filepath.Base(path)
	if base == "Dockerfile" || base == "docker-compose.yml" || base == "docker-compose.yaml" {
		return true
	}
	return false
}

func rel(root, path string) string {
	r, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(r)
}