package tests

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestE2EModSecurity_EnableDisableReenableWithScopedExclusion(t *testing.T) {
	requestURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	runtimeURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_RUNTIME_URL")), "/")
	if requestURL == "" || runtimeURL == "" {
		t.Skip("WAF_E2E_BASE_URL and WAF_E2E_RUNTIME_URL are required; skipping ModSecurity behavioral e2e")
	}

	client, requestBaseURL, requestHostOverride := newE2EClientAndBase(t, requestURL)
	loginE2EUser(t, client, requestBaseURL, requestHostOverride)

	const ruleID = 100001
	siteID := e2eUniqueID(t, "e2e-modsec")
	upstreamID := siteID + "-upstream"
	host := siteID + ".test"

	t.Cleanup(func() {
		resp := requestE2EJSON(t, client, http.MethodDelete, requestBaseURL+"/api/sites/"+siteID+"?auto_apply=false", requestHostOverride, nil)
		_ = resp.Body.Close()
		resp = requestE2EJSON(t, client, http.MethodDelete, requestBaseURL+"/api/upstreams/"+upstreamID+"?auto_apply=false", requestHostOverride, nil)
		_ = resp.Body.Close()
	})

	createE2EModSecuritySite(t, client, requestBaseURL, requestHostOverride, siteID, upstreamID, host)
	baseProfile := e2eGetProfile(t, client, requestBaseURL, requestHostOverride, siteID)

	profileFor := func(enabled bool, exclusions []any) map[string]any {
		profile := cloneMap(baseProfile)
		front := mapGetOrCreate(profile, "front_service")
		front["security_mode"] = "block"
		antibot := mapGetOrCreate(profile, "security_antibot")
		antibot["antibot_challenge"] = "no"
		modsec := mapGetOrCreate(profile, "security_modsecurity")
		modsec["use_modsecurity"] = enabled
		modsec["use_modsecurity_crs_plugins"] = false
		modsec["use_modsecurity_custom_configuration"] = true
		modsec["custom_configuration"] = map[string]any{
			"path":    "modsec/e2e-behavioral.conf",
			"content": `SecRule REQUEST_URI "@rx ^/modsec-(?:excluded|control)$" "id:100001,phase:2,deny,status:403,log"`,
		}
		modsec["exclusion_rules"] = exclusions
		return profile
	}

	applyProfile := func(t *testing.T, profile map[string]any) map[string]any {
		t.Helper()
		updated := e2ePutProfile(t, client, requestBaseURL, requestHostOverride, siteID, profile)
		e2eAssertModSecurityProfile(t, updated, profile)
		if revisionID := e2eCompileAndApply(t, client, requestBaseURL, requestHostOverride); revisionID == "" {
			t.Fatal("compile+apply returned an empty revision ID")
		} else if boolValue(mapGetOrCreate(profile, "security_modsecurity")["use_modsecurity"]) {
			assertE2EArtifactActive(t, revisionID, "modsecurity/easy/"+siteID+".conf", "SecRule", "100001")
		}
		time.Sleep(2 * time.Second)
		return updated
	}

	runtimeRequest := func(t *testing.T, path string) int {
		t.Helper()
		req, err := http.NewRequest(http.MethodGet, runtimeURL+path, nil)
		if err != nil {
			t.Fatalf("build runtime request: %v", err)
		}
		req.Host = host
		resp, err := (&http.Client{
			Timeout:   10 * time.Second,
			Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
		}).Do(req)
		if err != nil {
			t.Fatalf("runtime request %s: %v", path, err)
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusMisdirectedRequest {
			t.Fatalf("runtime did not load the test site for %s: status=%d body=%s", path, resp.StatusCode, body)
		}
		return resp.StatusCode
	}

	t.Run("enabled_blocks_attack_without_exclusion", func(t *testing.T) {
		applyProfile(t, profileFor(true, nil))
		assertE2EModSecurityStatus(t, runtimeRequest(t, "/modsec-excluded"), http.StatusForbidden, "enabled ModSecurity")
	})

	t.Run("disabled_allows_attack", func(t *testing.T) {
		applyProfile(t, profileFor(false, nil))
		assertE2EModSecurityNotBlocked(t, runtimeRequest(t, "/modsec-excluded"), "disabled ModSecurity")
	})

	t.Run("reenabled_exclusion_is_scoped", func(t *testing.T) {
		exclusions := []any{map[string]any{
			"path":     "/modsec-excluded",
			"methods":  []any{"GET"},
			"mode":     "exact",
			"rule_ids": []any{ruleID},
			"targets":  []any{"REQUEST_URI"},
			"comment":  "allow only the e2e false-positive path",
		}}
		applyProfile(t, profileFor(true, exclusions))
		assertE2EModSecurityNotBlocked(t, runtimeRequest(t, "/modsec-excluded"), "scoped exclusion")
		assertE2EModSecurityStatus(t, runtimeRequest(t, "/modsec-control"), http.StatusForbidden, "control path outside exclusion")
	})
}

func createE2EModSecuritySite(t *testing.T, client *http.Client, requestBaseURL, requestHostOverride, siteID, upstreamID, host string) {
	t.Helper()
	site := postJSON(t, client, requestBaseURL+"/api/sites?auto_apply=false", requestHostOverride, map[string]any{
		"id": siteID, "primary_host": host, "enabled": true, "default_upstream_id": upstreamID, "listen_http": true, "listen_https": false, "use_easy_config": true,
	})
	siteBody, _ := io.ReadAll(site.Body)
	_ = site.Body.Close()
	if site.StatusCode != http.StatusCreated && site.StatusCode != http.StatusOK {
		t.Fatalf("create site: status=%d body=%s", site.StatusCode, siteBody)
	}
	upstream := postJSON(t, client, requestBaseURL+"/api/upstreams?auto_apply=false", requestHostOverride, map[string]any{
		"id": upstreamID, "site_id": siteID, "name": upstreamID, "scheme": "http", "host": "upstream-echo", "port": 8888, "base_path": "/", "pass_host_header": false,
	})
	upstreamBody, _ := io.ReadAll(upstream.Body)
	_ = upstream.Body.Close()
	if upstream.StatusCode != http.StatusCreated && upstream.StatusCode != http.StatusOK {
		t.Fatalf("create upstream: status=%d body=%s", upstream.StatusCode, upstreamBody)
	}
}

func e2eAssertModSecurityProfile(t *testing.T, updated, expected map[string]any) {
	t.Helper()
	updatedModsec := mapGetOrCreate(updated, "security_modsecurity")
	expectedModsec := mapGetOrCreate(expected, "security_modsecurity")
	if boolValue(updatedModsec["use_modsecurity"]) != boolValue(expectedModsec["use_modsecurity"]) {
		t.Fatalf("saved use_modsecurity=%v, want %v", updatedModsec["use_modsecurity"], expectedModsec["use_modsecurity"])
	}
	if fmt.Sprint(updatedModsec["custom_configuration"]) != fmt.Sprint(expectedModsec["custom_configuration"]) {
		t.Fatalf("saved custom ModSecurity configuration does not round-trip: got=%v want=%v", updatedModsec["custom_configuration"], expectedModsec["custom_configuration"])
	}
	updatedExclusions := normalizeOptionalRuleList(updatedModsec["exclusion_rules"])
	expectedExclusions := normalizeOptionalRuleList(expectedModsec["exclusion_rules"])
	if fmt.Sprint(updatedExclusions) != fmt.Sprint(expectedExclusions) {
		t.Fatalf("saved structured ModSecurity exclusions do not round-trip: got=%v want=%v", updatedModsec["exclusion_rules"], expectedModsec["exclusion_rules"])
	}
}

func normalizeOptionalRuleList(value any) any {
	switch rules := value.(type) {
	case nil:
		return []any{}
	case []any:
		return rules
	default:
		return value
	}
}

func assertE2EModSecurityStatus(t *testing.T, got, want int, stage string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s: status=%d, want %d", stage, got, want)
	}
}

func assertE2EModSecurityNotBlocked(t *testing.T, status int, stage string) {
	t.Helper()
	if status == http.StatusForbidden {
		t.Fatalf("%s: ModSecurity still blocked the request with status %d", stage, status)
	}
}
