package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func releaseEvidencePython(t *testing.T) string {
	t.Helper()
	for _, command := range []string{"python3", "python"} {
		if _, err := exec.LookPath(command); err == nil {
			return command
		}
	}
	t.Skip("python3 or python is required for release evidence tests")
	return ""
}

func TestReleaseEvidenceScript_RequiresSuccessfulE2EAndDAST(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	work := t.TempDir()
	e2eDir := filepath.Join(work, "e2e", "security-invariants")
	dastDir := filepath.Join(work, "dast", "baseline")
	if err := os.MkdirAll(e2eDir, 0o755); err != nil {
		t.Fatalf("create e2e directory: %v", err)
	}
	if err := os.MkdirAll(dastDir, 0o755); err != nil {
		t.Fatalf("create dast directory: %v", err)
	}
	writeReleaseEvidenceFixture(t, filepath.Join(e2eDir, "e2e-evidence.json"), `{"status":"passed","summary":{"pass":2,"fail":0,"skip":0}}`)
	writeReleaseEvidenceFixture(t, filepath.Join(dastDir, "dast-evidence.json"), `{"status":"passed","threshold":"High","counts":{"High":0,"Critical":0},"blocking_alerts":[]}`)

	output := filepath.Join(work, "out")
	python := releaseEvidencePython(t)
	command := releaseEvidenceCommand(python, root, e2eDir, dastDir, output)
	if got, err := command.CombinedOutput(); err != nil {
		t.Fatalf("generate valid release evidence: %v: %s", err, got)
	}
	if _, err := os.Stat(filepath.Join(output, "release-evidence.md")); err != nil {
		t.Fatalf("release summary missing: %v", err)
	}
	summary, err := os.ReadFile(filepath.Join(output, "release-evidence-summary.md"))
	if err != nil {
		t.Fatalf("release evidence summary missing: %v", err)
	}
	if !strings.Contains(string(summary), "| E2E suite | Passed | Failed | Skipped |") ||
		!strings.Contains(string(summary), "| DAST suite | High | Critical | Status |") {
		t.Fatalf("release evidence summary does not contain verification tables: %s", summary)
	}

	writeReleaseEvidenceFixture(t, filepath.Join(dastDir, "dast-evidence.json"), `{"status":"failed","counts":{"High":1,"Critical":0},"blocking_alerts":[{"name":"fixture"}]}`)
	if got, err := releaseEvidenceCommand(python, root, e2eDir, dastDir, filepath.Join(work, "blocked")).CombinedOutput(); err == nil || !strings.Contains(string(got), "blocking findings") {
		t.Fatalf("expected High DAST finding to block release evidence, err=%v output=%s", err, got)
	}
}

func writeReleaseEvidenceFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write fixture %s: %v", path, err)
	}
}

func releaseEvidenceCommand(python, root, e2eDir, dastDir, output string) *exec.Cmd {
	command := exec.Command(python, "scripts/write-release-evidence.py",
		"--e2e-root", filepath.Dir(e2eDir),
		"--dast-root", filepath.Dir(dastDir),
		"--output-dir", output,
		"--version", "1.5.6",
		"--commit", "fixture-commit")
	command.Dir = root
	return command
}
