package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitLabCIE2EPortsStayBelowEphemeralRange(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", ".gitlab-ci.yml"))
	if err != nil {
		t.Fatalf("read .gitlab-ci.yml: %v", err)
	}
	source := string(content)
	for _, marker := range []string{
		"E2E_BROWSER_PORT_BASE=$((24000 + E2E_BROWSER_SLOT * 100))",
		"E2E_STACK_PORT_BASE=$((25000 + E2E_STACK_SLOT * 100))",
	} {
		if !strings.Contains(source, marker) {
			t.Fatalf("GitLab CI missing safe E2E port marker %q", marker)
		}
	}
	for _, unsafe := range []string{
		"E2E_BROWSER_PORT_BASE=$((43000",
		"E2E_STACK_PORT_BASE=$((42000",
	} {
		if strings.Contains(source, unsafe) {
			t.Fatalf("GitLab CI uses ephemeral-range E2E ports: %q", unsafe)
		}
	}
}

func TestBrowserE2EUsesItsIsolatedComposeNetwork(t *testing.T) {
	script, err := os.ReadFile(filepath.Join("..", "..", "scripts", "run-e2e-tests.sh"))
	if err != nil {
		t.Fatalf("read run-e2e-tests.sh: %v", err)
	}
	source := string(script)
	for _, marker := range []string{
		`WAF_BROWSER_DOCKER_NETWORK="${E2E_PROJECT}_waf-e2e-net"`,
		`WAF_BROWSER_BASE_URL="https://e2e-management.test"`,
		`--network "$WAF_BROWSER_DOCKER_NETWORK"`,
		`-e WAF_E2E_RUNTIME_URL="$WAF_BROWSER_RUNTIME_URL"`,
	} {
		if !strings.Contains(source, marker) {
			t.Fatalf("browser E2E isolation missing marker %q", marker)
		}
	}
	if strings.Contains(source, "docker run --rm --network host") {
		t.Fatal("browser E2E must not share the runner host network namespace")
	}

	compose, err := os.ReadFile(filepath.Join("..", "..", "deploy", "compose", "e2e", "docker-compose.yml"))
	if err != nil {
		t.Fatalf("read E2E Compose file: %v", err)
	}
	composeSource := strings.ReplaceAll(string(compose), "\r\n", "\n")
	runtimeStart := strings.Index(composeSource, "\n  runtime:\n")
	runtimeEnd := strings.Index(composeSource, "\n  ui:\n")
	if runtimeStart < 0 || runtimeEnd <= runtimeStart {
		t.Fatal("E2E Compose runtime service boundaries are missing")
	}
	if !strings.Contains(composeSource[runtimeStart:runtimeEnd], "- e2e-management.test") {
		t.Fatal("E2E runtime must expose its management hostname inside the isolated Compose network")
	}
}
