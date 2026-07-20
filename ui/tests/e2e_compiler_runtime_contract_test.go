package tests

import (
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestE2ECompilerRuntimeAuthContract proves one complete configuration path:
// API profile -> compiled revision artifact -> active runtime file -> HTTP auth.
func TestE2ECompilerRuntimeAuthContract(t *testing.T) {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	runtimeURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_RUNTIME_URL")), "/")
	if baseURL == "" || runtimeURL == "" {
		t.Skip("WAF_E2E_BASE_URL and WAF_E2E_RUNTIME_URL are required")
	}
	client, requestBaseURL, hostOverride := newE2EClientAndBase(t, baseURL)
	loginE2EUser(t, client, requestBaseURL, hostOverride)
	const siteID = "e2e-contract-auth"
	const upstreamID = siteID + "-upstream"
	const host = siteID + ".test"
	const username = "contract-user"
	const password = "contract-password"
	for _, endpoint := range []string{"/api/sites/" + siteID + "?auto_apply=false", "/api/upstreams/" + upstreamID + "?auto_apply=false"} {
		resp := requestE2EJSON(t, client, http.MethodDelete, requestBaseURL+endpoint, hostOverride, nil)
		_ = resp.Body.Close()
	}
	t.Cleanup(func() {
		for _, endpoint := range []string{"/api/sites/" + siteID + "?auto_apply=false", "/api/upstreams/" + upstreamID + "?auto_apply=false"} {
			resp := requestE2EJSON(t, client, http.MethodDelete, requestBaseURL+endpoint, hostOverride, nil)
			_ = resp.Body.Close()
		}
		e2eCompileAndApply(t, client, requestBaseURL, hostOverride)
	})
	createE2EModSecuritySite(t, client, requestBaseURL, hostOverride, siteID, upstreamID, host)
	profile := e2eGetProfile(t, client, requestBaseURL, hostOverride, siteID)
	httpBehavior := mapGetOrCreate(profile, "http_behavior")
	httpBehavior["allowed_methods"] = []string{"GET", "POST", "HEAD", "OPTIONS"}
	auth := mapGetOrCreate(profile, "security_auth_basic")
	auth["use_auth_basic"] = true
	auth["auth_mode"] = "basic"
	auth["auth_order"] = "auth_first"
	auth["auth_basic_location"] = "sitewide"
	auth["auth_basic_user"] = username
	auth["auth_basic_password"] = password
	auth["users"] = []map[string]any{{"username": username, "password": password, "enabled": true}}
	resp := e2ePutProfile(t, client, requestBaseURL, hostOverride, siteID, profile)
	e2eAssertModSecurityProfile(t, resp, profile)
	revisionID := e2eCompileAndApply(t, client, requestBaseURL, hostOverride)
	if revisionID == "" {
		t.Fatal("compile/apply returned an empty revision ID")
	}

	artifactPath := "nginx/easy-locations/" + siteID + ".conf"
	controlPlaneContainer := strings.TrimSpace(os.Getenv("WAF_E2E_CONTROL_PLANE_CONTAINER"))
	if controlPlaneContainer == "" {
		controlPlaneContainer = "waf-e2e-control-plane"
	}
	artifact, err := exec.Command("docker", "exec", controlPlaneContainer, "cat", "/var/lib/waf/candidates/"+revisionID+"/"+artifactPath).CombinedOutput()
	if err != nil {
		t.Fatalf("read compiled revision artifact: %v: %s", err, artifact)
	}
	for _, directive := range []string{"auth_basic", "auth_basic_user_file"} {
		if !strings.Contains(string(artifact), directive) {
			t.Fatalf("compiled artifact missing %s", directive)
		}
	}
	runtimeContainer := strings.TrimSpace(os.Getenv("WAF_E2E_RUNTIME_CONTAINER"))
	if runtimeContainer == "" {
		runtimeContainer = "waf-e2e-runtime"
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
		t.Fatalf("read active runtime artifact: %v: %s", err, runtime)
	}
	if string(runtime) != string(artifact) {
		t.Fatal("active runtime artifact differs from applied revision artifact")
	}
	waitForStatus := func(password string, want int) {
		t.Helper()
		deadline := time.Now().Add(30 * time.Second)
		status := 0
		for time.Now().Before(deadline) {
			response := e2eBasicVerify(t, runtimeURL, host, username, password)
			status = response.StatusCode
			_ = response.Body.Close()
			if status == want {
				return
			}
			time.Sleep(250 * time.Millisecond)
		}
		t.Fatalf("Basic Auth invalid_password=%t: status=%d, want %d", password == "invalid-password", status, want)
	}
	waitForStatus("invalid-password", http.StatusUnauthorized)
	waitForStatus(password, http.StatusNoContent)
	t.Logf("contract revision=%s runtime-artifact=matched HTTP=401,204", revisionID)
}
