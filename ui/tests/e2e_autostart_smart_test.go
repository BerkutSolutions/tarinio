//go:build e2e

package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestE2EAutoStartSmartRuntime(t *testing.T) {
	if strings.TrimSpace(os.Getenv("WAF_E2E_AUTOSTART_SMART")) != "1" {
		t.Fatal("set WAF_E2E_AUTOSTART_SMART=1 to run auto-start smart e2e")
	}

	repoRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	repoRoot = filepath.Clean(filepath.Join(repoRoot, ".."))
	composeDir := filepath.Join(repoRoot, "deploy", "compose", "auto-start")
	composeFile := filepath.Join(composeDir, "docker-compose.yml")
	if _, err := os.Stat(composeFile); err != nil {
		t.Fatalf("auto-start compose not found: %v", err)
	}
	// The production installer creates .env from .env.example. CI checks out a
	// clean repository, so create the same non-secret local configuration for
	// this isolated stack and remove it after the scenario.
	envFile := filepath.Join(composeDir, ".env")
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		example, readErr := os.ReadFile(filepath.Join(composeDir, ".env.example"))
		if readErr != nil {
			t.Fatalf("read auto-start .env.example: %v", readErr)
		}
		if writeErr := os.WriteFile(envFile, example, 0o600); writeErr != nil {
			t.Fatalf("create auto-start .env for isolated e2e stack: %v", writeErr)
		}
		t.Cleanup(func() { _ = os.Remove(envFile) })
	} else if err != nil {
		t.Fatalf("inspect auto-start .env: %v", err)
	}

	// This scenario is part of the full E2E suite, whose primary stack already
	// owns its runtime ports. Keep the auto-start deployment isolated from both
	// that stack and any developer-owned auto-start project.
	uiPort := firstNonEmptyAutoStart(strings.TrimSpace(os.Getenv("WAF_E2E_AUTOSTART_UI_PORT")), "19683")
	runtimeHTTPPort := firstNonEmptyAutoStart(strings.TrimSpace(os.Getenv("WAF_E2E_AUTOSTART_RUNTIME_HTTP_PORT")), "11681")
	runtimeHTTPSPort := firstNonEmptyAutoStart(strings.TrimSpace(os.Getenv("WAF_E2E_AUTOSTART_RUNTIME_HTTPS_PORT")), "24444")
	project := firstNonEmptyAutoStart(strings.TrimSpace(os.Getenv("WAF_E2E_AUTOSTART_PROJECT")), "strict-autostart-e2e")
	previousRuntimeContainer, runtimeContainerSet := os.LookupEnv("WAF_E2E_RUNTIME_CONTAINER")
	if err := os.Setenv("WAF_E2E_RUNTIME_CONTAINER", project+"-runtime-1"); err != nil {
		t.Fatalf("set auto-start runtime container: %v", err)
	}
	t.Cleanup(func() {
		if runtimeContainerSet {
			_ = os.Setenv("WAF_E2E_RUNTIME_CONTAINER", previousRuntimeContainer)
		} else {
			_ = os.Unsetenv("WAF_E2E_RUNTIME_CONTAINER")
		}
	})
	autoStartEnv := []string{
		"COMPOSE_PROJECT_NAME=" + project,
		"WAF_UI_HTTP_PORT=" + uiPort,
		"WAF_RUNTIME_HTTP_PORT=" + runtimeHTTPPort,
		"WAF_RUNTIME_HTTPS_PORT=" + runtimeHTTPSPort,
	}
	runCmdSoft(composeDir, autoStartEnv, "docker", "compose", "-f", composeFile, "down", "--volumes", "--remove-orphans")
	t.Log("starting clean auto-start compose stack")
	runCmd(t, composeDir, autoStartEnv, "docker", "compose", "-f", composeFile, "up", "-d", "--build")
	t.Cleanup(func() {
		runCmdSoft(composeDir, autoStartEnv, "docker", "compose", "-f", composeFile, "down", "--volumes", "--remove-orphans")
	})

	baseURL := firstNonEmptyAutoStart(strings.TrimSpace(os.Getenv("WAF_E2E_AUTOSTART_BASE_URL")), "http://127.0.0.1:"+uiPort)
	client, requestBaseURL, requestHostOverride := newE2EClientAndBase(t, baseURL)
	previousUsername, usernameSet := os.LookupEnv("WAF_E2E_USERNAME")
	previousPassword, passwordSet := os.LookupEnv("WAF_E2E_PASSWORD")
	_ = os.Setenv("WAF_E2E_USERNAME", "admin")
	_ = os.Setenv("WAF_E2E_PASSWORD", "admin")
	t.Cleanup(func() {
		if usernameSet {
			_ = os.Setenv("WAF_E2E_USERNAME", previousUsername)
		} else {
			_ = os.Unsetenv("WAF_E2E_USERNAME")
		}
		if passwordSet {
			_ = os.Setenv("WAF_E2E_PASSWORD", previousPassword)
		} else {
			_ = os.Unsetenv("WAF_E2E_PASSWORD")
		}
	})
	t.Log("waiting for auto-start management login")
	loginE2EUserWithRetry(t, client, requestBaseURL, requestHostOverride)
	t.Log("auto-start management login succeeded")
	edgeClient, _, _ := newE2EClientAndBase(t, "http://127.0.0.1:"+runtimeHTTPPort)

	siteID := "autotest-site"
	siteHost := "autotest.localhost"
	upstreamID := "autotest-upstream"

	t.Run("ServiceCRUD", func(t *testing.T) {
		t.Log("checking site and upstream CRUD")
		createSiteResp := postJSON(t, client, requestBaseURL+"/api/sites?auto_apply=false", requestHostOverride, map[string]any{
			"id":                  siteID,
			"primary_host":        siteHost,
			"enabled":             true,
			"listen_http":         true,
			"listen_https":        false,
			"use_easy_config":     true,
			"default_upstream_id": upstreamID,
		})
		assertStatusIn(t, createSiteResp, "create site", http.StatusCreated, http.StatusOK)

		createUpstreamResp := postJSON(t, client, requestBaseURL+"/api/upstreams?auto_apply=false", requestHostOverride, map[string]any{
			"id":      upstreamID,
			"site_id": siteID,
			"host":    "127.0.0.1",
			"port":    18080,
			"scheme":  "http",
		})
		assertStatusIn(t, createUpstreamResp, "create upstream", http.StatusCreated, http.StatusOK)

		updateSiteResp := requestJSON(t, client, http.MethodPut, requestBaseURL+"/api/sites/"+siteID+"?auto_apply=false", requestHostOverride, map[string]any{
			"id":           siteID,
			"primary_host": siteHost,
			"enabled":      true,
			"description":  "autotest",
		})
		assertStatusOK(t, updateSiteResp, "update site")
		profile := e2eGetEasyProfile(t, client, requestBaseURL, requestHostOverride, siteID)
		if profile == nil {
			t.Fatal("read auto-start site profile for L7 enforcement")
		}
		limits, ok := profile["security_behavior_and_limits"].(map[string]any)
		if !ok {
			t.Fatal("auto-start site profile has no security behavior settings")
		}
		limits["use_limit_req"] = true
		limits["limit_req_url"] = "/"
		limits["limit_req_rate"] = "1r/s"
		limits["use_bad_behavior"] = false
		profile["security_behavior_and_limits"] = limits
		profile["site_id"] = siteID
		profileResp := postJSON(t, client, requestBaseURL+"/api/easy-site-profiles/"+siteID+"?auto_apply=false", requestHostOverride, profile)
		assertStatusIn(t, profileResp, "configure site L7 rate limit", http.StatusOK, http.StatusCreated)
		e2eCompileAndApply(t, client, requestBaseURL, requestHostOverride)
	})

	t.Run("AntiDDoSAndErrors", func(t *testing.T) {
		t.Log("checking real Anti-DDoS L7 enforcement on the compiled auto-start runtime")
		getSettings := getWithAuth(t, client, requestBaseURL+"/api/anti-ddos/settings", requestHostOverride)
		assertStatusOK(t, getSettings, "get anti-ddos settings")

		updateSettings := requestJSON(t, client, http.MethodPut, requestBaseURL+"/api/anti-ddos/settings", requestHostOverride, map[string]any{
			"use_l4_guard": false, "chain_mode": "auto", "conn_limit": 200, "rate_per_second": 100, "rate_burst": 200,
			"ports": []int{80, 443}, "target": "REJECT", "enforce_l7_rate_limit": true,
			"l7_requests_per_second": 1, "l7_burst": 1, "l7_status_code": 429, "model_enabled": true, "model_poll_interval_sec": 5,
		})
		assertStatusOK(t, updateSettings, "update anti-ddos settings")
		e2eCompileAndApply(t, client, requestBaseURL, requestHostOverride)

		edgeURL := "http://127.0.0.1:" + runtimeHTTPPort + "/"
		var gotRate bool
		for i := 0; i < 8; i++ {
			resp := getWithHost(t, edgeClient, edgeURL, siteHost)
			if resp.StatusCode == http.StatusTooManyRequests {
				gotRate = true
			}
			_ = resp.Body.Close()
			if gotRate {
				break
			}
			time.Sleep(150 * time.Millisecond)
		}
		if !gotRate {
			t.Fatalf("compiled runtime never enforced configured L7 rate limit (expected HTTP 429)")
		}
	})

	t.Run("UIAndModules", func(t *testing.T) {
		t.Log("checking UI routes and JavaScript modules")
		uiPages := []string{"/dashboard", "/sites", "/requests", "/events", "/bans", "/settings", "/administration", "/healthcheck"}
		for _, page := range uiPages {
			resp := getWithAuth(t, client, requestBaseURL+page, requestHostOverride)
			assertStatusOK(t, resp, "open "+page)
		}
		modules := []string{
			"dashboard.page-renderers.js", "dashboard.page-interactions.js", "dashboard.detail-shared.js",
			"sites.js", "requests.js", "events.js", "bans.js", "settings.js", "administration.js",
		}
		for _, mod := range modules {
			resp := getWithAuthRetry429(t, client, requestBaseURL+"/static/js/pages/"+mod, requestHostOverride, 5)
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("module %s status=%d body=%s", mod, resp.StatusCode, mustReadBody(t, resp.Body))
			}
			_ = resp.Body.Close()
		}
	})

	t.Run("CleanupService", func(t *testing.T) {
		deleteUpstream := requestJSON(t, client, http.MethodDelete, requestBaseURL+"/api/upstreams/"+upstreamID+"?auto_apply=false", requestHostOverride, nil)
		if deleteUpstream.StatusCode != http.StatusOK && deleteUpstream.StatusCode != http.StatusNoContent && deleteUpstream.StatusCode != http.StatusNotFound {
			t.Fatalf("delete upstream failed: status=%d body=%s", deleteUpstream.StatusCode, mustReadBody(t, deleteUpstream.Body))
		}
		_ = deleteUpstream.Body.Close()

		deleteSite := requestJSON(t, client, http.MethodDelete, requestBaseURL+"/api/sites/"+siteID+"?auto_apply=false", requestHostOverride, nil)
		if deleteSite.StatusCode != http.StatusOK && deleteSite.StatusCode != http.StatusNoContent && deleteSite.StatusCode != http.StatusNotFound {
			t.Fatalf("delete site failed: status=%d body=%s", deleteSite.StatusCode, mustReadBody(t, deleteSite.Body))
		}
		_ = deleteSite.Body.Close()
	})
}

func firstNonEmptyAutoStart(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func runCmd(t *testing.T, dir string, extraEnv []string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), extraEnv...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, string(out))
	}
}

func runCmdSoft(dir string, extraEnv []string, name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), extraEnv...)
	_, _ = cmd.CombinedOutput()
}

func requestJSON(t *testing.T, client *http.Client, method string, endpoint string, hostOverride string, payload any) *http.Response {
	t.Helper()
	var body []byte
	var err error
	if payload != nil {
		body, err = json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal %s %s payload: %v", method, endpoint, err)
		}
	}
	req, err := http.NewRequest(method, endpoint, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create %s %s request: %v", method, endpoint, err)
	}
	req.Header.Set("Accept", "application/json,text/html,*/*")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if hostOverride != "" {
		req.Host = hostOverride
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s failed: %v", method, endpoint, err)
	}
	return resp
}

func getWithHost(t *testing.T, client *http.Client, endpoint string, hostHeader string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		t.Fatalf("create request %s: %v", endpoint, err)
	}
	req.Host = hostHeader
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed %s: %v", endpoint, err)
	}
	return resp
}

func autoUnbanLoopback(t *testing.T, client *http.Client, baseURL string, hostOverride string, siteID string) {
	t.Helper()
	resp := postJSON(t, client, fmt.Sprintf("%s/api/sites/%s/unban", baseURL, siteID), hostOverride, map[string]any{"ip": "127.0.0.1"})
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		t.Fatalf("auto-unban failed: status=%d body=%s", resp.StatusCode, mustReadBody(t, resp.Body))
	}
	_ = resp.Body.Close()
}

func assertStatusOK(t *testing.T, resp *http.Response, action string) {
	t.Helper()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%s failed: status=%d body=%s", action, resp.StatusCode, mustReadBody(t, resp.Body))
	}
	_ = resp.Body.Close()
}

func assertStatusIn(t *testing.T, resp *http.Response, action string, allowed ...int) {
	t.Helper()
	for _, status := range allowed {
		if resp.StatusCode == status {
			_ = resp.Body.Close()
			return
		}
	}
	t.Fatalf("%s failed: status=%d body=%s", action, resp.StatusCode, mustReadBody(t, resp.Body))
}

func loginE2EUserWithRetry(t *testing.T, client *http.Client, requestBaseURL, requestHostOverride string) {
	t.Helper()
	if err := waitForHTTP(client, requestBaseURL+"/login", requestHostOverride, 90*time.Second); err != nil {
		t.Fatalf("ui is not ready: %v", err)
	}
	challengeURI := normalizeChallengeURI(strings.TrimSpace(os.Getenv("WAF_E2E_ANTIBOT_CHALLENGE_URI")))
	ensureManagementLoginAccess(t, client, requestBaseURL, requestHostOverride, challengeURI)
	username := strings.TrimSpace(os.Getenv("WAF_E2E_USERNAME"))
	if username == "" {
		username = "admin"
	}
	password := strings.TrimSpace(os.Getenv("WAF_E2E_PASSWORD"))
	if password == "" {
		password = "admin"
	}

	deadline := time.Now().Add(90 * time.Second)
	for {
		payload, err := json.Marshal(map[string]any{
			"username": username,
			"password": password,
		})
		if err != nil {
			t.Fatalf("marshal auto-start login: %v", err)
		}
		req, err := http.NewRequest(http.MethodPost, requestBaseURL+"/api/auth/login", bytes.NewReader(payload))
		if err != nil {
			t.Fatalf("create auto-start login request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		if requestHostOverride != "" {
			req.Host = requestHostOverride
		}
		resp, err := client.Do(req)
		if err != nil {
			if time.Now().After(deadline) {
				t.Fatalf("auto-start login did not become available: %v", err)
			}
			time.Sleep(2 * time.Second)
			continue
		}
		if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
			_ = resp.Body.Close()
			ensureManagementLoginAccess(t, client, requestBaseURL, requestHostOverride, challengeURI)
			time.Sleep(2 * time.Second)
			continue
		}
		if resp.StatusCode == http.StatusOK {
			_ = resp.Body.Close()
			return
		}
		body := mustReadBody(t, resp.Body)
		if time.Now().After(deadline) {
			t.Fatalf("login failed after retries: status=%d body=%s", resp.StatusCode, body)
		}
		time.Sleep(2 * time.Second)
	}
}
