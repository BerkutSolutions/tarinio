package tests

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestE2ESecurityModesReality(t *testing.T) {
	requestURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	runtimeURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_RUNTIME_URL")), "/")
	if requestURL == "" || runtimeURL == "" {
		t.Skip("WAF_E2E_BASE_URL and WAF_E2E_RUNTIME_URL are required; skipping security modes e2e")
	}

	client, requestBaseURL, requestHostOverride := newE2EClientAndBase(t, requestURL)
	loginE2EUser(t, client, requestBaseURL, requestHostOverride)

	const (
		siteID     = "e2e-security-modes-site"
		upstreamID = "e2e-security-modes-upstream"
		host       = "e2e-security-modes.test"
	)

	t.Cleanup(func() {
		resp := requestE2EJSON(t, client, http.MethodDelete, requestBaseURL+"/api/sites/"+siteID+"?auto_apply=false", requestHostOverride, nil)
		_ = resp.Body.Close()
		resp = requestE2EJSON(t, client, http.MethodDelete, requestBaseURL+"/api/upstreams/"+upstreamID+"?auto_apply=false", requestHostOverride, nil)
		_ = resp.Body.Close()
	})

	createE2ESecurityModesSite(t, client, requestBaseURL, requestHostOverride, siteID, upstreamID, host)
	original := e2eGetProfile(t, client, requestBaseURL, requestHostOverride, siteID)

	cases := []struct {
		mode                string
		expectWAFEnabled    bool
		expectAccessEnabled bool
		expectRateEnabled   bool
		expectBlocking      bool
	}{
		{mode: "transparent", expectWAFEnabled: false, expectAccessEnabled: true, expectRateEnabled: true, expectBlocking: false},
		{mode: "monitor", expectWAFEnabled: false, expectAccessEnabled: true, expectRateEnabled: true, expectBlocking: false},
		{mode: "block", expectWAFEnabled: false, expectAccessEnabled: true, expectRateEnabled: true, expectBlocking: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.mode, func(t *testing.T) {
			profile := e2eBuildModeStressProfile(original, tc.mode)
			updated := e2ePutProfile(t, client, requestBaseURL, requestHostOverride, siteID, profile)
			e2eAssertProfileModeReality(t, updated)

			revisionID := e2eCompileAndApply(t, client, requestBaseURL, requestHostOverride)
			if revisionID == "" {
				t.Fatal("compile+apply returned an empty revision ID")
			}

			wafEnabled := e2ePolicyEnabledForSite(t, client, requestBaseURL, requestHostOverride, "/api/waf-policies", siteID)
			accessEnabled := e2ePolicyEnabledForSite(t, client, requestBaseURL, requestHostOverride, "/api/access-policies", siteID)
			rateEnabled := e2ePolicyEnabledForSite(t, client, requestBaseURL, requestHostOverride, "/api/rate-limit-policies", siteID)
			if wafEnabled != tc.expectWAFEnabled {
				t.Fatalf("%s mode: waf policy enabled=%v, want %v", tc.mode, wafEnabled, tc.expectWAFEnabled)
			}
			if accessEnabled != tc.expectAccessEnabled {
				t.Fatalf("%s mode: access policy enabled=%v, want %v", tc.mode, accessEnabled, tc.expectAccessEnabled)
			}
			if rateEnabled != tc.expectRateEnabled {
				t.Fatalf("%s mode: rate-limit policy enabled=%v, want %v", tc.mode, rateEnabled, tc.expectRateEnabled)
			}

			e2eAssertSecurityModeRuntimeBehavior(t, runtimeURL, host, tc.mode, tc.expectBlocking)
		})
	}
}

func createE2ESecurityModesSite(t *testing.T, client *http.Client, requestBaseURL, requestHostOverride, siteID, upstreamID, host string) {
	t.Helper()
	site := postJSON(t, client, requestBaseURL+"/api/sites?auto_apply=false", requestHostOverride, map[string]any{
		"id":                  siteID,
		"primary_host":        host,
		"enabled":             true,
		"default_upstream_id": upstreamID,
		"listen_http":         true,
		"listen_https":        false,
		"use_easy_config":     true,
	})
	siteBody, _ := io.ReadAll(site.Body)
	_ = site.Body.Close()
	if site.StatusCode != http.StatusCreated && site.StatusCode != http.StatusOK {
		t.Fatalf("create site: status=%d body=%s", site.StatusCode, string(siteBody))
	}

	upstream := postJSON(t, client, requestBaseURL+"/api/upstreams?auto_apply=false", requestHostOverride, map[string]any{
		"id":               upstreamID,
		"site_id":          siteID,
		"name":             upstreamID,
		"scheme":           "http",
		"host":             "upstream-echo",
		"port":             8888,
		"base_path":        "/",
		"pass_host_header": false,
	})
	upstreamBody, _ := io.ReadAll(upstream.Body)
	_ = upstream.Body.Close()
	if upstream.StatusCode != http.StatusCreated && upstream.StatusCode != http.StatusOK {
		t.Fatalf("create upstream: status=%d body=%s", upstream.StatusCode, string(upstreamBody))
	}
}

func e2eGetProfile(t *testing.T, client *http.Client, requestBaseURL, requestHostOverride, siteID string) map[string]any {
	t.Helper()
	resp := getWithAuth(t, client, requestBaseURL+"/api/easy-site-profiles/"+siteID, requestHostOverride)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get profile failed: status=%d body=%s", resp.StatusCode, mustReadBody(t, resp.Body))
	}
	var profile map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		t.Fatalf("decode profile: %v", err)
	}
	return profile
}

func e2ePutProfile(t *testing.T, client *http.Client, requestBaseURL, requestHostOverride, siteID string, profile map[string]any) map[string]any {
	t.Helper()
	resp := requestE2EJSON(t, client, http.MethodPut, requestBaseURL+"/api/easy-site-profiles/"+siteID, requestHostOverride, profile)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("put profile failed: status=%d body=%s", resp.StatusCode, mustReadBody(t, resp.Body))
	}
	var updated map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&updated); err != nil {
		t.Fatalf("decode updated profile: %v", err)
	}
	return updated
}

func e2eBuildModeStressProfile(base map[string]any, mode string) map[string]any {
	profile := cloneMap(base)
	front := mapGetOrCreate(profile, "front_service")
	front["security_mode"] = mode
	front["profile"] = "balanced"

	behavior := mapGetOrCreate(profile, "security_behavior_and_limits")
	behavior["use_limit_req"] = true
	behavior["use_limit_conn"] = true
	behavior["use_bad_behavior"] = true
	behavior["use_blacklist"] = true
	behavior["limit_req_rate"] = "20r/s"
	behavior["blacklist_ip"] = []any{"198.51.100.10"}
	behavior["custom_limit_rules"] = []any{map[string]any{"path": "/stress", "rate": "5r/s"}}

	modsec := mapGetOrCreate(profile, "security_modsecurity")
	modsec["use_modsecurity"] = true
	modsec["use_modsecurity_crs_plugins"] = false
	modsec["use_modsecurity_custom_configuration"] = true
	modsec["custom_configuration"] = map[string]any{
		"path":    "modsec/e2e-security-modes.conf",
		"content": `SecRule REQUEST_URI "@streq /mode-check" "id:100002,phase:2,deny,status:403,log"`,
	}

	return profile
}

func e2eAssertProfileModeReality(t *testing.T, profile map[string]any) {
	t.Helper()
	front := mapGetOrCreate(profile, "front_service")
	behavior := mapGetOrCreate(profile, "security_behavior_and_limits")
	modsec := mapGetOrCreate(profile, "security_modsecurity")

	if !boolValue(modsec["use_modsecurity"]) {
		t.Fatal("saving a non-block mode must preserve the ModSecurity setting")
	}
	if strings.TrimSpace(fmt.Sprint(front["security_mode"])) == "" {
		t.Fatal("saved profile must keep security_mode")
	}
	if !boolValue(behavior["use_limit_req"]) || !boolValue(behavior["use_limit_conn"]) || !boolValue(behavior["use_bad_behavior"]) || !boolValue(behavior["use_blacklist"]) {
		t.Fatal("saving a non-block mode must preserve traffic protection settings")
	}
	customConfiguration, ok := modsec["custom_configuration"].(map[string]any)
	if !ok {
		t.Fatal("saved profile must keep custom ModSecurity configuration")
	}
	if !strings.Contains(fmt.Sprint(customConfiguration["content"]), `REQUEST_URI "@streq /mode-check"`) {
		t.Fatalf("saved ModSecurity custom configuration lost runtime trigger: %v", customConfiguration["content"])
	}
}

func e2ePolicyEnabledForSite(t *testing.T, client *http.Client, requestBaseURL, requestHostOverride, path, siteID string) bool {
	t.Helper()
	resp := getWithAuth(t, client, requestBaseURL+path, requestHostOverride)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get %s failed: status=%d body=%s", path, resp.StatusCode, mustReadBody(t, resp.Body))
	}
	var items []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	for _, item := range items {
		if strings.TrimSpace(fmt.Sprint(item["site_id"])) != siteID {
			continue
		}
		return boolValue(item["enabled"])
	}
	return false
}

func cloneMap(src map[string]any) map[string]any {
	out := make(map[string]any, len(src))
	for key, value := range src {
		if nested, ok := value.(map[string]any); ok {
			out[key] = cloneMap(nested)
			continue
		}
		out[key] = value
	}
	return out
}

func mapGetOrCreate(parent map[string]any, key string) map[string]any {
	if value, ok := parent[key].(map[string]any); ok {
		return value
	}
	value := map[string]any{}
	parent[key] = value
	return value
}

func boolValue(value any) bool {
	v, _ := value.(bool)
	return v
}

func e2eAssertSecurityModeRuntimeBehavior(t *testing.T, runtimeURL, host, mode string, expectBlocking bool) {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	var lastStatus int
	var lastLocation string
	var lastBody string
	for time.Now().Before(deadline) {
		status, location, body := e2eSecurityModeRuntimeProbe(t, runtimeURL, host, mode)
		lastStatus = status
		lastLocation = location
		lastBody = body
		if e2eSecurityModeBehaviorMatches(mode, expectBlocking, status, location, body) {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}

	if expectBlocking {
		t.Fatalf("%s mode must reactivate blocking after compile/apply: status=%d location=%q body=%.300s", mode, lastStatus, lastLocation, lastBody)
	}
	t.Fatalf("%s mode must preserve settings without enforcing them after compile/apply: status=%d location=%q body=%.300s", mode, lastStatus, lastLocation, lastBody)
}

func e2eSecurityModeRuntimeProbe(t *testing.T, runtimeURL, host, mode string) (int, string, string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, runtimeURL+"/mode-check", nil)
	if err != nil {
		t.Fatalf("runtime request: %v", err)
	}
	req.Host = host
	req.Header.Set("X-E2E-Security-Mode", mode)

	resp, err := (&http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}).Do(req)
	if err != nil {
		t.Fatalf("runtime request failed: %v", err)
	}

	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	return resp.StatusCode, resp.Header.Get("Location"), string(body)
}

func e2eSecurityModeBehaviorMatches(mode string, expectBlocking bool, status int, location, body string) bool {
	isBlocking := status == http.StatusUnauthorized ||
		status == http.StatusForbidden ||
		status == http.StatusTooManyRequests ||
		status == http.StatusUnavailableForLegalReasons
	if status == http.StatusFound || status == http.StatusSeeOther {
		if strings.Contains(location, "/challenge") || strings.Contains(location, "/auth") {
			isBlocking = true
		}
	}
	if expectBlocking {
		return isBlocking
	}

	if isBlocking || status != http.StatusOK {
		return false
	}

	lowerBody := strings.ToLower(body)
	return strings.Contains(lowerBody, "/mode-check") &&
		strings.Contains(lowerBody, "x-e2e-security-mode") &&
		strings.Contains(lowerBody, strings.ToLower(mode))
}
