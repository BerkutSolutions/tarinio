package tests

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// assertE2EArtifactActive proves that the exact candidate compiled by the
// control-plane is the file currently loaded by runtime before HTTP is tested.
func assertE2EArtifactActive(t *testing.T, revisionID, artifactPath string, required ...string) {
	t.Helper()
	controlPlane := strings.TrimSpace(os.Getenv("WAF_E2E_CONTROL_PLANE_CONTAINER"))
	if controlPlane == "" {
		controlPlane = "waf-e2e-control-plane"
	}
	runtimeContainer := strings.TrimSpace(os.Getenv("WAF_E2E_RUNTIME_CONTAINER"))
	if runtimeContainer == "" {
		runtimeContainer = "waf-e2e-runtime"
	}
	candidate, err := exec.Command("docker", "exec", controlPlane, "cat", "/var/lib/waf/candidates/"+revisionID+"/"+artifactPath).CombinedOutput()
	if err != nil {
		t.Fatalf("read compiled artifact %s: %v: %s", artifactPath, err, candidate)
	}
	for _, value := range required {
		if !strings.Contains(string(candidate), value) {
			t.Fatalf("compiled artifact %s misses %q", artifactPath, value)
		}
	}
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		active, currentErr := exec.Command("docker", "exec", runtimeContainer, "cat", "/var/lib/waf/active/current.json").CombinedOutput()
		if currentErr == nil && strings.Contains(string(active), revisionID) {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	runtime, err := exec.Command("docker", "exec", runtimeContainer, "cat", "/etc/waf/current/"+artifactPath).CombinedOutput()
	if err != nil {
		t.Fatalf("read active artifact %s: %v: %s", artifactPath, err, runtime)
	}
	if string(candidate) != string(runtime) {
		t.Fatalf("active runtime artifact differs from revision %s: %s", revisionID, artifactPath)
	}
}
