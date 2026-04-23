package tests

import (
	"encoding/json"
	"os/exec"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

var appVersionPattern = regexp.MustCompile(`var AppVersion = "([^"]+)"`)

type packageJSON struct {
	Version string `json:"version"`
}

type packageLockJSON struct {
	Version  string                     `json:"version"`
	Packages map[string]packageLockNode `json:"packages"`
}

type packageLockNode struct {
	Version  string `json:"version"`
	Resolved string `json:"resolved"`
}

func TestReleaseDocsAndLockfileConsistency(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	appVersion := mustReadAppVersion(t, repoRoot)

	changelog := mustReadFile(t, filepath.Join(repoRoot, "CHANGELOG.md"))
	expectedChangelogHeader := "## [" + appVersion + "] - "
	if !strings.Contains(changelog, expectedChangelogHeader) {
		t.Fatalf("CHANGELOG.md must contain release header %q", expectedChangelogHeader)
	}

	pkg := mustReadPackageJSON(t, filepath.Join(repoRoot, "package.json"))
	if pkg.Version != appVersion {
		t.Fatalf("package.json version mismatch: got %q want %q", pkg.Version, appVersion)
	}

	lock := mustReadPackageLockJSON(t, filepath.Join(repoRoot, "package-lock.json"))
	if lock.Version != appVersion {
		t.Fatalf("package-lock.json top-level version mismatch: got %q want %q", lock.Version, appVersion)
	}

	rootPkg, ok := lock.Packages[""]
	if !ok {
		t.Fatalf("package-lock.json is missing root package entry")
	}
	if rootPkg.Version != appVersion {
		t.Fatalf("package-lock.json root package version mismatch: got %q want %q", rootPkg.Version, appVersion)
	}

	validateNpmLockfileInstall(t, repoRoot)

	requiredDocs := []string{
		filepath.Join(repoRoot, "docs", "eng", "enterprise-identity.md"),
		filepath.Join(repoRoot, "docs", "eng", "evidence-and-releases.md"),
		filepath.Join(repoRoot, "docs", "eng", "logging-architecture.md"),
		filepath.Join(repoRoot, "docs", "eng", "migration-compatibility.md"),
		filepath.Join(repoRoot, "docs", "ru", "enterprise-identity.md"),
		filepath.Join(repoRoot, "docs", "ru", "evidence-and-releases.md"),
		filepath.Join(repoRoot, "docs", "ru", "logging-architecture.md"),
		filepath.Join(repoRoot, "docs", "ru", "migration-compatibility.md"),
	}
	for _, path := range requiredDocs {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("required enterprise documentation file is missing: %s (%v)", path, err)
		}
	}

	uiNginx := mustReadFile(t, filepath.Join(repoRoot, "ui", "nginx.conf"))
	if strings.Contains(uiNginx, "try_files /var/lib/waf/request-archive/requests.jsonl") {
		t.Fatalf("ui/nginx.conf must not serve /api/requests from legacy request-archive files")
	}

	loggingDocs := []string{
		mustReadFile(t, filepath.Join(repoRoot, "docs", "eng", "logging-architecture.md")),
		mustReadFile(t, filepath.Join(repoRoot, "docs", "eng", "migration-compatibility.md")),
		mustReadFile(t, filepath.Join(repoRoot, "docs", "ru", "logging-architecture.md")),
		mustReadFile(t, filepath.Join(repoRoot, "docs", "ru", "migration-compatibility.md")),
	}
	for _, content := range loggingDocs {
		if strings.Contains(content, "default profile now includes `OpenSearch`, `ClickHouse`, and `Vault`") {
			t.Fatalf("logging and migration docs must not describe ClickHouse as part of the default 2.0.10 profile")
		}
	}

	i18nFiles := []string{
		filepath.Join(repoRoot, "ui", "app", "static", "i18n", "en.json"),
		filepath.Join(repoRoot, "ui", "app", "static", "i18n", "ru.json"),
		filepath.Join(repoRoot, "ui", "app", "static", "i18n", "de.json"),
		filepath.Join(repoRoot, "ui", "app", "static", "i18n", "sr.json"),
		filepath.Join(repoRoot, "ui", "app", "static", "i18n", "zh.json"),
	}
	for _, path := range i18nFiles {
		content := mustReadFile(t, path)
		if strings.Contains(content, "2.0.5") {
			t.Fatalf("i18n file still contains stale 2.0.5 version marker: %s", path)
		}
	}
}

func validateNpmLockfileInstall(t *testing.T, repoRoot string) {
	t.Helper()

	nodeCommand := "node"
	npmCommand := "npm"
	if runtime.GOOS == "windows" {
		nodeCommand = "node.exe"
		npmCommand = "npm.cmd"
	}

	lockfileCheck := exec.Command(nodeCommand, filepath.Join("scripts", "check-docs-lockfile.js"))
	lockfileCheck.Dir = repoRoot
	lockfileOutput, lockfileErr := lockfileCheck.CombinedOutput()
	if lockfileErr != nil {
		t.Fatalf("package-lock.json failed static lockfile validation: %v\n%s", lockfileErr, strings.TrimSpace(string(lockfileOutput)))
	}

	cmd := exec.Command(npmCommand, "ci", "--ignore-scripts", "--dry-run")
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("package-lock.json failed npm ci dry-run validation: %v\n%s", err, strings.TrimSpace(string(output)))
	}
}

func mustReadAppVersion(t *testing.T, repoRoot string) string {
	t.Helper()
	raw := mustReadFile(t, filepath.Join(repoRoot, "control-plane", "internal", "appmeta", "meta.go"))
	match := appVersionPattern.FindStringSubmatch(raw)
	if len(match) != 2 {
		t.Fatalf("failed to parse AppVersion from meta.go")
	}
	return match[1]
}

func mustReadPackageJSON(t *testing.T, path string) packageJSON {
	t.Helper()
	var out packageJSON
	raw := mustReadFile(t, path)
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return out
}

func mustReadPackageLockJSON(t *testing.T, path string) packageLockJSON {
	t.Helper()
	var out packageLockJSON
	raw := mustReadFile(t, path)
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return out
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(raw)
}

