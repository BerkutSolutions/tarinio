package tests

import (
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestE2ERuntimeReloadReadinessIgnoresUnavailableUpstream ensures apply is
// validated by the local nginx revision marker, not by a protected upstream.
func TestE2ERuntimeReloadReadinessIgnoresUnavailableUpstream(t *testing.T) {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	if baseURL == "" {
		t.Skip("WAF_E2E_BASE_URL not set; skipping runtime reload readiness e2e")
	}
	client, requestBaseURL, hostOverride := newE2EClientAndBase(t, baseURL)
	loginE2EUser(t, client, requestBaseURL, hostOverride)

	siteID := e2eUniqueID(t, "e2e-reload-readiness")
	upstreamID, host := siteID+"-upstream", siteID+".test"
	createE2EModSecuritySite(t, client, requestBaseURL, hostOverride, siteID, upstreamID, host)
	t.Cleanup(func() {
		for _, endpoint := range []string{"/api/sites/" + siteID + "?auto_apply=false", "/api/upstreams/" + upstreamID + "?auto_apply=false"} {
			response := requestE2EJSON(t, client, http.MethodDelete, requestBaseURL+endpoint, hostOverride, nil)
			_ = response.Body.Close()
		}
		_ = e2eCompileAndApply(t, client, requestBaseURL, hostOverride)
	})

	response := requestE2EJSON(t, client, http.MethodPut, requestBaseURL+"/api/upstreams/"+upstreamID+"?auto_apply=false", hostOverride, map[string]any{
		"id": upstreamID, "site_id": siteID, "name": upstreamID, "scheme": "http", "host": "203.0.113.1", "port": 8080, "base_path": "/", "pass_host_header": false,
	})
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("set unavailable upstream: status=%d body=%s", response.StatusCode, body)
	}

	revisionID := e2eCompileAndApply(t, client, requestBaseURL, hostOverride)
	if revisionID == "" {
		t.Fatal("apply with unavailable upstream must still activate the valid nginx revision")
	}
	runtimeContainer := strings.TrimSpace(os.Getenv("WAF_E2E_RUNTIME_CONTAINER"))
	if runtimeContainer == "" {
		runtimeContainer = "waf-e2e-runtime"
	}
	output, err := exec.Command("docker", "exec", runtimeContainer, "wget", "-qSO-", "http://127.0.0.1/__waf_runtime/readiness").CombinedOutput()
	if err != nil || !strings.Contains(string(output), "X-WAF-Runtime-Revision: "+revisionID) {
		t.Fatalf("local runtime readiness must expose applied revision %q: err=%v output=%s", revisionID, err, output)
	}
}
