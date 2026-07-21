package tests

import (
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestE2ENegativeSecurityProbes proves that malicious public requests are
// blocked, unauthenticated administration stays unavailable, and a harmless
// request remains usable after the protection revision is active.
func TestE2ENegativeSecurityProbes(t *testing.T) {
	panelURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	runtimeURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_RUNTIME_URL")), "/")
	if panelURL == "" || runtimeURL == "" {
		t.Skip("WAF_E2E_BASE_URL and WAF_E2E_RUNTIME_URL are required")
	}
	canaryURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_DAST_CANARY_URL")), "/")
	if canaryURL == "" {
		canaryURL = "http://127.0.0.1:18083"
	}
	client, baseURL, hostOverride := newE2EClientAndBase(t, panelURL)
	loginE2EUser(t, client, baseURL, hostOverride)
	const siteID = "e2e-negative-probes"
	const upstreamID = siteID + "-upstream"
	const host = siteID + ".test"
	for _, endpoint := range []string{"/api/sites/" + siteID + "?auto_apply=false", "/api/upstreams/" + upstreamID + "?auto_apply=false"} {
		resp := requestE2EJSON(t, client, http.MethodDelete, baseURL+endpoint, hostOverride, nil)
		_ = resp.Body.Close()
	}
	t.Cleanup(func() {
		for _, endpoint := range []string{"/api/sites/" + siteID + "?auto_apply=false", "/api/upstreams/" + upstreamID + "?auto_apply=false"} {
			resp := requestE2EJSON(t, client, http.MethodDelete, baseURL+endpoint, hostOverride, nil)
			_ = resp.Body.Close()
		}
		// The nightly job tears down the complete disposable stack. Avoid a
		// second compile while the just-restarted control-plane is still
		// reconciling its dependencies; it adds no protection value here.
		if os.Getenv("WAF_E2E_RESILIENCE") == "1" {
			return
		}
		e2eCompileAndApply(t, client, baseURL, hostOverride)
	})
	createE2EModSecuritySite(t, client, baseURL, hostOverride, siteID, upstreamID, host)
	e2eSetUpstreamTarget(t, client, baseURL, hostOverride, upstreamID, "dast-canary", 8080)
	profile := e2eGetProfile(t, client, baseURL, hostOverride, siteID)
	front := mapGetOrCreate(profile, "front_service")
	front["security_mode"] = "block"
	antibot := mapGetOrCreate(profile, "security_antibot")
	antibot["antibot_challenge"] = "no"
	modsec := mapGetOrCreate(profile, "security_modsecurity")
	modsec["use_modsecurity"] = true
	modsec["use_modsecurity_crs_plugins"] = false
	modsec["use_modsecurity_custom_configuration"] = true
	modsec["custom_configuration"] = map[string]any{
		"path":    "modsec/e2e-negative-probes.conf",
		"content": `SecRule ARGS "@rx (?i:(?:union[[:space:]]+select|<script|\.\./\.\./|\$\())" "id:100021,phase:2,deny,status:403,log,msg:'DAST attack matrix'"`,
	}
	updated := e2ePutProfile(t, client, baseURL, hostOverride, siteID, profile)
	e2eAssertModSecurityProfile(t, updated, profile)
	if revisionID := e2eCompileAndApply(t, client, baseURL, hostOverride); revisionID == "" {
		t.Fatal("negative probe compile/apply returned an empty revision")
	}
	time.Sleep(2 * time.Second)

	publicRequest := func(path string) (int, error) {
		req, err := http.NewRequest(http.MethodGet, runtimeURL+path, nil)
		if err != nil {
			return 0, err
		}
		req.Host = host
		resp, err := (&http.Client{Timeout: 10 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}).Do(req)
		if err != nil {
			return 0, err
		}
		_, _ = io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return resp.StatusCode, nil
	}
	resetCanary := func(t *testing.T) {
		t.Helper()
		resp, err := http.Post(canaryURL+"/__dast/reset", "application/json", nil)
		if err != nil {
			t.Fatalf("reset DAST canary: %v", err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("reset DAST canary: status=%d", resp.StatusCode)
		}
	}
	canaryCount := func(t *testing.T) int {
		t.Helper()
		resp, err := http.Get(canaryURL + "/__dast/records")
		if err != nil {
			t.Fatalf("read DAST canary: %v", err)
		}
		defer resp.Body.Close()
		var payload struct {
			Count int `json:"count"`
		}
		if resp.StatusCode != http.StatusOK || json.NewDecoder(resp.Body).Decode(&payload) != nil {
			t.Fatalf("read DAST canary: status=%d", resp.StatusCode)
		}
		return payload.Count
	}

	t.Run("public_attacks_are_blocked", func(t *testing.T) {
		for name, values := range map[string]url.Values{
			"sqli":      {"id": []string{"1 UNION SELECT username FROM users"}},
			"xss":       {"q": []string{"<script>alert(1)</script>"}},
			"traversal": {"path": []string{"../../etc/passwd"}},
			"command":   {"cmd": []string{"$(id)"}},
		} {
			t.Run(name, func(t *testing.T) {
				resetCanary(t)
				path := "/waf/" + name + "-probe"
				status, err := publicRequest(path + "?" + values.Encode())
				if err != nil {
					t.Fatal(err)
				}
				if status != http.StatusForbidden {
					t.Fatalf("%s payload: status=%d, want 403", name, status)
				}
				if got := canaryCount(t); got != 0 {
					t.Fatalf("%s payload reached upstream canary %d times", name, got)
				}
				e2eWaitForRequestTelemetry(t, client, baseURL, hostOverride, path, http.StatusForbidden, "modsecurity", "security")
				t.Logf("%s: HTTP 403, upstream canary=0, telemetry reason=modsecurity", name)
			})
		}
	})
	t.Run("unauthenticated_admin_is_rejected", func(t *testing.T) {
		resp, err := http.Get(panelURL + "/api/administration/users")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusForbidden {
			t.Fatalf("anonymous administration request: status=%d, want 401/403", resp.StatusCode)
		}
	})
	t.Run("legitimate_request_is_not_blocked", func(t *testing.T) {
		resetCanary(t)
		status, err := publicRequest("/legitimate?" + url.Values{"q": []string{"health check"}}.Encode())
		if err != nil {
			t.Fatal(err)
		}
		if status == http.StatusForbidden {
			t.Fatal("legitimate public request was blocked")
		}
		if got := canaryCount(t); got != 1 {
			t.Fatalf("legitimate request reached upstream canary %d times, want 1", got)
		}
		t.Log("legitimate request reached upstream canary exactly once")
	})
	if os.Getenv("WAF_E2E_RESILIENCE") == "1" {
		composeFile := strings.TrimSpace(os.Getenv("WAF_E2E_COMPOSE_FILE"))
		if composeFile == "" {
			t.Fatal("WAF_E2E_COMPOSE_FILE is required for resilience checks")
		}
		t.Run("runtime_restart_preserves_active_protection", func(t *testing.T) {
			e2eComposeControl(t, composeFile, "restart", "runtime")
			e2eWaitForPublicStatus(t, publicRequest, "/sqli-probe?id=1", http.StatusForbidden)
		})
		t.Run("control_plane_vault_postgres_outage_does_not_allow_attack", func(t *testing.T) {
			e2eComposeControl(t, composeFile, "stop", "control-plane", "vault", "postgres")
			t.Cleanup(func() {
				e2eComposeControl(t, composeFile, "start", "postgres", "vault", "control-plane")
				e2eWaitForControlPlane(t, panelURL)
			})
			e2eWaitForPublicStatus(t, publicRequest, "/xss-probe?q=%3Cscript%3E", http.StatusForbidden)
		})
	}
}

func e2eSetUpstreamTarget(t *testing.T, client *http.Client, baseURL, hostOverride, upstreamID, host string, port int) {
	t.Helper()
	response := requestE2EJSON(t, client, http.MethodPut, baseURL+"/api/upstreams/"+upstreamID+"?auto_apply=false", hostOverride, map[string]any{
		"id": upstreamID, "site_id": "e2e-negative-probes", "name": upstreamID, "scheme": "http", "host": host, "port": port, "base_path": "/", "pass_host_header": false,
	})
	body := mustReadBody(t, response.Body)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("configure DAST canary upstream: status=%d body=%s", response.StatusCode, body)
	}
}

func e2eComposeControl(t *testing.T, composeFile string, args ...string) {
	t.Helper()
	command := append([]string{"compose", "-f", filepath.Clean(composeFile)}, args...)
	out, err := exec.Command("docker", command...).CombinedOutput()
	if err != nil {
		t.Fatalf("docker %s: %v: %s", strings.Join(command, " "), err, out)
	}
}

func e2eWaitForPublicStatus(t *testing.T, request func(string) (int, error), path string, want int) {
	t.Helper()
	deadline := time.Now().Add(35 * time.Second)
	for time.Now().Before(deadline) {
		status, err := request(path)
		if err == nil && status == want {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("runtime %s did not preserve status %d", path, want)
}

func e2eWaitForControlPlane(t *testing.T, baseURL string) {
	t.Helper()
	deadline := time.Now().Add(35 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL + "/healthz")
		if err == nil && resp.StatusCode == http.StatusOK {
			_ = resp.Body.Close()
			return
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatal("control-plane did not recover after dependency outage")
}
