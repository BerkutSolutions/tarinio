package tests

import (
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	protectionAttackerService = "e2e-attacker"
	protectionAttackerIP      = "172.30.0.30"
	l4AttackerService         = "e2e-l4-attacker"
	l4AttackerIP              = "172.30.0.31"
)

// TestE2EL4L7AdaptiveProtection proves the complete public-service path:
// real L7 429 responses, adaptive drop decision, installed L4 rule, then a
// failed connection from the same isolated client container.
func TestE2EL4L7AdaptiveProtection(t *testing.T) {
	if strings.TrimSpace(os.Getenv("WAF_E2E_L4_L7_PROTECTION")) != "1" {
		t.Skip("set WAF_E2E_L4_L7_PROTECTION=1 to run L4/L7 adaptive protection e2e")
	}
	runtimeURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_RUNTIME_URL")), "/")
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	composeFile := strings.TrimSpace(os.Getenv("WAF_E2E_COMPOSE_FILE"))
	if runtimeURL == "" || baseURL == "" || composeFile == "" {
		t.Fatal("WAF_E2E_RUNTIME_URL, WAF_E2E_BASE_URL and WAF_E2E_COMPOSE_FILE are required")
	}
	if _, err := os.Stat(composeFile); err != nil {
		t.Fatalf("compose file: %v", err)
	}

	client, requestBaseURL, requestHostOverride := newE2EClientAndBase(t, baseURL)
	loginE2EUser(t, client, requestBaseURL, requestHostOverride)
	const siteID, upstreamID, host = "e2e-l4-l7-site", "e2e-l4-l7-upstream", "e2e-l4-l7.test"
	deleteProtectionResource(t, client, requestBaseURL+"/api/sites/"+siteID+"?auto_apply=false", requestHostOverride)
	deleteProtectionResource(t, client, requestBaseURL+"/api/upstreams/"+upstreamID+"?auto_apply=false", requestHostOverride)

	previousAntiDDoS := getProtectionJSON(t, client, requestBaseURL+"/api/anti-ddos/settings", requestHostOverride)
	t.Cleanup(func() {
		putProtectionJSON(t, client, requestBaseURL+"/api/anti-ddos/settings", requestHostOverride, previousAntiDDoS)
	})

	protectionSettings := map[string]any{
		"use_l4_guard": true, "chain_mode": "input", "conn_limit": 10000, "rate_per_second": 10000,
		"rate_burst": 10000, "ports": []int{80, 443}, "target": "DROP", "enforce_l7_rate_limit": true,
		"l7_requests_per_second": 1, "l7_burst": 1, "l7_status_code": 429,
		"model_enabled": true, "model_poll_interval_seconds": 1, "model_decay_lambda": 0.01,
		"model_throttle_threshold": 1.0, "model_drop_threshold": 2.0, "model_hold_seconds": 60,
		"model_throttle_rate_per_second": 1, "model_throttle_burst": 1, "model_throttle_target": "DROP",
		"model_weight_429": 1.0, "model_weight_403": 1.8, "model_weight_444": 2.2,
		"model_emergency_rps": 10000, "model_emergency_unique_ips": 10000, "model_emergency_per_ip_rps": 2,
	}

	postProtectionJSON(t, client, requestBaseURL+"/api/sites?auto_apply=false", requestHostOverride, map[string]any{
		"id": siteID, "primary_host": host, "enabled": true, "listen_http": true,
		"listen_https": false, "use_easy_config": true, "default_upstream_id": upstreamID,
	})
	postProtectionJSON(t, client, requestBaseURL+"/api/upstreams?auto_apply=false", requestHostOverride, map[string]any{
		"id": upstreamID, "site_id": siteID, "name": upstreamID, "scheme": "http",
		"host": "upstream-echo", "port": 8888, "base_path": "/", "pass_host_header": false,
	})
	profile := e2eGetEasyProfile(t, client, requestBaseURL, requestHostOverride, siteID)
	if profile == nil {
		t.Fatal("could not read protection test profile")
	}
	profile["site_id"] = siteID
	profile["security_mode"] = "block"
	front, _ := profile["front_service"].(map[string]any)
	if front == nil {
		t.Fatal("protection test profile has no front_service settings")
	}
	front["adaptive_model_enabled"] = true
	profile["front_service"] = front
	limits, _ := profile["security_behavior_and_limits"].(map[string]any)
	if limits == nil {
		t.Fatal("protection test profile has no security_behavior_and_limits settings")
	}
	limits["use_limit_req"] = true
	limits["limit_req_url"] = "/"
	limits["limit_req_rate"] = "1r/s"
	limits["use_bad_behavior"] = false
	profile["security_behavior_and_limits"] = limits
	postProtectionJSON(t, client, requestBaseURL+"/api/easy-site-profiles/"+siteID, requestHostOverride, profile)
	// Apply global protection last: earlier E2E cleanups can enqueue auto-apply
	// jobs, and the L4/L7 revision must be the active one before traffic starts.
	putProtectionJSON(t, client, requestBaseURL+"/api/anti-ddos/settings", requestHostOverride, protectionSettings)
	e2eCompileAndApply(t, client, requestBaseURL, requestHostOverride)
	waitForProtectionHTTP(t, runtimeURL, host)

	var floodOutput string
	t.Run("L7 returns 429 during a real flood", func(t *testing.T) {
		floodOutput = runProtectionCompose(t, composeFile, l4AttackerService, "for i in $(seq 1 40); do wget -S -O /dev/null --header='Host: "+host+"' http://runtime/ 2>&1 || true; done")
		if !strings.Contains(floodOutput, "429") {
			t.Fatalf("L7 rate limit did not return a real 429; attacker output=%s", floodOutput)
		}
	})

	var adaptive, chain string
	t.Run("adaptive model promotes the abusive IP to drop", func(t *testing.T) {
		deadline := time.Now().Add(35 * time.Second)
		for time.Now().Before(deadline) {
			adaptive = runProtectionRuntime(t, composeFile, "cat /etc/waf/l4guard-adaptive/adaptive.json 2>/dev/null || true")
			if adaptiveHasDropEntry(adaptive, l4AttackerIP) {
				return
			}
			time.Sleep(time.Second)
		}
		t.Fatalf("adaptive model did not emit a drop entry for attacker %s: %s", l4AttackerIP, adaptive)
	})
	t.Run("L4 guard drops new connections from the adaptive IP", func(t *testing.T) {
		deadline := time.Now().Add(12 * time.Second)
		for time.Now().Before(deadline) {
			chain = runProtectionRuntime(t, composeFile, "iptables -w -S WAF-RUNTIME-L4 2>/dev/null || true")
			if strings.Contains(chain, "-s "+l4AttackerIP+"/32 -p tcp -j DROP") {
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
		if !strings.Contains(chain, "-s "+l4AttackerIP+"/32 -p tcp -j DROP") {
			t.Fatalf("L4 guard did not install attacker drop rule: %s", chain)
		}
		blockedOutput := runProtectionCompose(t, composeFile, l4AttackerService, "wget --tries=1 -T 3 -S -O /dev/null --header='Host: "+host+"' http://runtime/ 2>&1 || true")
		if strings.Contains(blockedOutput, "HTTP/1.1 200") || strings.Contains(blockedOutput, "HTTP/1.1 429") {
			t.Fatalf("L4 drop did not block a new attacker connection: %s", blockedOutput)
		}
	})
}

func adaptiveHasDropEntry(raw, ip string) bool {
	var payload struct {
		Entries []struct {
			IP     string `json:"ip"`
			Action string `json:"action"`
		} `json:"entries"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return false
	}
	for _, entry := range payload.Entries {
		if entry.IP == ip && entry.Action == "drop" {
			return true
		}
	}
	return false
}

func getProtectionJSON(t *testing.T, client *http.Client, endpoint, host string) map[string]any {
	t.Helper()
	resp := requestE2EJSON(t, client, http.MethodGet, endpoint, host, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: status=%d body=%s", endpoint, resp.StatusCode, mustReadBody(t, resp.Body))
	}
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode %s: %v", endpoint, err)
	}
	return payload
}

func putProtectionJSON(t *testing.T, client *http.Client, endpoint, host string, payload map[string]any) {
	t.Helper()
	resp := requestE2EJSON(t, client, http.MethodPut, endpoint, host, payload)
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PUT %s: status=%d body=%s", endpoint, resp.StatusCode, body)
	}
}

func postProtectionJSON(t *testing.T, client *http.Client, endpoint, host string, payload map[string]any) {
	t.Helper()
	resp := postJSON(t, client, endpoint, host, payload)
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("POST %s: status=%d body=%s", endpoint, resp.StatusCode, body)
	}
}

func deleteProtectionResource(t *testing.T, client *http.Client, endpoint, host string) {
	t.Helper()
	resp := requestE2EJSON(t, client, http.MethodDelete, endpoint, host, nil)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		t.Fatalf("DELETE %s: status=%d", endpoint, resp.StatusCode)
	}
}

func runProtectionAttacker(t *testing.T, composeFile, command string) string {
	t.Helper()
	return runProtectionCompose(t, composeFile, protectionAttackerService, command)
}

func runProtectionRuntime(t *testing.T, composeFile, command string) string {
	t.Helper()
	return runProtectionCompose(t, composeFile, "runtime", command)
}

func runProtectionCompose(t *testing.T, composeFile, service, command string) string {
	t.Helper()
	cmd := exec.Command("docker", "compose", "-f", filepath.Clean(composeFile), "exec", "-T", service, "sh", "-lc", command)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("docker compose exec %s: %v\n%s", service, err, out)
	}
	return string(out)
}

func waitForProtectionHTTP(t *testing.T, runtimeURL, host string) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		req, err := http.NewRequest(http.MethodGet, runtimeURL+"/", nil)
		if err == nil {
			req.Host = host
			resp, err := (&http.Client{Timeout: 3 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}).Do(req)
			if err == nil {
				_ = resp.Body.Close()
				return
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatal("public protection test site did not become reachable")
}
