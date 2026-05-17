package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestE2ESecurityModesReality(t *testing.T) {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	if baseURL == "" {
		t.Skip("WAF_E2E_BASE_URL is not set; skipping security modes e2e")
	}

	client, requestBaseURL, requestHostOverride := newE2EClientAndBase(t, baseURL)
	loginE2EUser(t, client, requestBaseURL, requestHostOverride)

	siteID := e2ePickSiteID(t, client, requestBaseURL, requestHostOverride)
	original := e2eGetProfile(t, client, requestBaseURL, requestHostOverride, siteID)
	defer e2ePutProfile(t, client, requestBaseURL, requestHostOverride, siteID, original)

	cases := []struct {
		mode                string
		expectModSecurity   bool
		expectBlocking      bool
		expectWAFEnabled    bool
		expectAccessEnabled bool
		expectRateEnabled   bool
	}{
		{mode: "transparent", expectModSecurity: false, expectBlocking: false, expectWAFEnabled: false, expectAccessEnabled: false, expectRateEnabled: false},
		{mode: "monitor", expectModSecurity: true, expectBlocking: false, expectWAFEnabled: true, expectAccessEnabled: false, expectRateEnabled: false},
		{mode: "block", expectModSecurity: true, expectBlocking: true, expectWAFEnabled: true, expectAccessEnabled: true, expectRateEnabled: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.mode, func(t *testing.T) {
			profile := e2eBuildModeStressProfile(original, tc.mode)
			updated := e2ePutProfile(t, client, requestBaseURL, requestHostOverride, siteID, profile)
			e2eAssertProfileModeReality(t, updated, tc.expectModSecurity, tc.expectBlocking)

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
		})
	}
}

func e2ePickSiteID(t *testing.T, client *http.Client, requestBaseURL, requestHostOverride string) string {
	t.Helper()
	resp := getWithAuth(t, client, requestBaseURL+"/api/sites", requestHostOverride)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list sites failed: status=%d body=%s", resp.StatusCode, mustReadBody(t, resp.Body))
	}
	var items []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("decode sites list: %v", err)
	}
	for _, item := range items {
		id := strings.TrimSpace(fmt.Sprint(item["id"]))
		if id != "" {
			return id
		}
	}
	t.Fatal("no sites found for e2e security modes")
	return ""
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

	antibot := mapGetOrCreate(profile, "security_antibot")
	antibot["antibot_challenge"] = "javascript"
	antibot["scanner_auto_ban_enabled"] = true
	antibot["challenge_escalation_enabled"] = true
	antibot["challenge_escalation_mode"] = "captcha"
	antibot["challenge_rules"] = []any{map[string]any{"path": "/admin", "challenge": "captcha"}}

	auth := mapGetOrCreate(profile, "security_auth_basic")
	auth["use_auth_basic"] = true
	auth["auth_basic_user"] = "admin"
	auth["auth_basic_password"] = "admin"
	auth["users"] = []any{map[string]any{"username": "admin", "password": "admin", "enabled": true}}

	apiPositive := mapGetOrCreate(profile, "security_api_positive")
	apiPositive["use_api_positive_security"] = true
	apiPositive["enforcement_mode"] = "block"
	apiPositive["default_action"] = "deny"
	apiPositive["endpoint_policies"] = []any{map[string]any{"path": "/api/private", "methods": []any{"GET"}, "mode": "block"}}

	modsec := mapGetOrCreate(profile, "security_modsecurity")
	modsec["use_modsecurity"] = true
	modsec["use_modsecurity_crs_plugins"] = true

	country := mapGetOrCreate(profile, "security_country_policy")
	country["blacklist_country"] = []any{"RU"}
	country["whitelist_country"] = []any{}

	return profile
}

func e2eAssertProfileModeReality(t *testing.T, profile map[string]any, expectModSecurity, expectBlocking bool) {
	t.Helper()
	behavior := mapGetOrCreate(profile, "security_behavior_and_limits")
	antibot := mapGetOrCreate(profile, "security_antibot")
	auth := mapGetOrCreate(profile, "security_auth_basic")
	apiPositive := mapGetOrCreate(profile, "security_api_positive")
	modsec := mapGetOrCreate(profile, "security_modsecurity")

	if boolValue(modsec["use_modsecurity"]) != expectModSecurity {
		t.Fatalf("use_modsecurity=%v, want %v", boolValue(modsec["use_modsecurity"]), expectModSecurity)
	}
	if boolValue(behavior["use_limit_req"]) != expectBlocking {
		t.Fatalf("use_limit_req=%v, want %v", boolValue(behavior["use_limit_req"]), expectBlocking)
	}
	if boolValue(behavior["use_limit_conn"]) != expectBlocking {
		t.Fatalf("use_limit_conn=%v, want %v", boolValue(behavior["use_limit_conn"]), expectBlocking)
	}
	if boolValue(behavior["use_bad_behavior"]) != expectBlocking {
		t.Fatalf("use_bad_behavior=%v, want %v", boolValue(behavior["use_bad_behavior"]), expectBlocking)
	}
	if boolValue(behavior["use_blacklist"]) != expectBlocking {
		t.Fatalf("use_blacklist=%v, want %v", boolValue(behavior["use_blacklist"]), expectBlocking)
	}
	if boolValue(auth["use_auth_basic"]) != expectBlocking {
		t.Fatalf("use_auth_basic=%v, want %v", boolValue(auth["use_auth_basic"]), expectBlocking)
	}
	if boolValue(apiPositive["use_api_positive_security"]) != expectBlocking {
		t.Fatalf("use_api_positive_security=%v, want %v", boolValue(apiPositive["use_api_positive_security"]), expectBlocking)
	}
	challenge := strings.TrimSpace(fmt.Sprint(antibot["antibot_challenge"]))
	wantChallenge := "no"
	if expectBlocking {
		wantChallenge = "javascript"
	}
	if challenge != wantChallenge {
		t.Fatalf("antibot_challenge=%q, want %q", challenge, wantChallenge)
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
