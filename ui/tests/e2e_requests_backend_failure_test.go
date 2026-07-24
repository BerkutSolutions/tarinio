//go:build e2e

package tests

import (
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestE2ERequestsBackendFailureIsVisible proves that a real unavailable
// runtime request backend is surfaced by the control-plane instead of being
// converted into a successful empty response.
func TestE2ERequestsBackendFailureIsVisible(t *testing.T) {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	if baseURL == "" {
		t.Fatal("WAF_E2E_BASE_URL is required")
	}
	runtimeContainer := strings.TrimSpace(os.Getenv("WAF_E2E_RUNTIME_CONTAINER"))
	if runtimeContainer == "" {
		t.Fatal("WAF_E2E_RUNTIME_CONTAINER is required")
	}
	client, requestBaseURL, hostOverride := newE2EClientAndBase(t, baseURL)
	loginE2EUser(t, client, requestBaseURL, hostOverride)

	initial := requestE2EJSON(t, client, http.MethodGet, requestBaseURL+"/api/requests?limit=1", hostOverride, nil)
	_ = initial.Body.Close()
	if initial.StatusCode != http.StatusOK {
		t.Fatalf("initial request read status=%d, want 200", initial.StatusCode)
	}

	if output, err := exec.Command("docker", "pause", runtimeContainer).CombinedOutput(); err != nil {
		t.Fatalf("pause runtime container: %v: %s", err, output)
	}
	t.Cleanup(func() {
		output, err := exec.Command("docker", "unpause", runtimeContainer).CombinedOutput()
		if err != nil {
			t.Errorf("unpause runtime container: %v: %s", err, output)
			return
		}
		deadline := time.Now().Add(30 * time.Second)
		for time.Now().Before(deadline) {
			response := requestE2EJSON(t, client, http.MethodGet, requestBaseURL+"/api/requests?limit=1", hostOverride, nil)
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
		t.Error("request backend did not recover after runtime unpause")
	})

	failed := requestE2EJSON(t, client, http.MethodGet, requestBaseURL+"/api/requests?limit=1", hostOverride, nil)
	_ = failed.Body.Close()
	if failed.StatusCode != http.StatusBadGateway {
		t.Fatalf("unavailable request backend status=%d, want 502", failed.StatusCode)
	}
}
