package tests

import (
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestE2EManagementHostIsExcludedFromGlobalL7RateLimit(t *testing.T) {
	runtimeURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_RUNTIME_URL")), "/")
	if runtimeURL == "" {
		t.Skip("WAF_E2E_RUNTIME_URL is not set; skipping management anti-ddos rate-limit e2e")
	}
	client, requestBaseURL, _ := newE2EClientAndBase(t, runtimeURL)
	host := strings.TrimSpace(os.Getenv("WAF_E2E_MANAGEMENT_HOST"))
	if host == "" {
		host = "e2e-management.test"
	}
	loginE2EUser(t, client, requestBaseURL, host)
	if e2eManagementSiteID(t, client, requestBaseURL, host, "http://"+host) == "" {
		t.Fatal("management site ID is required for anti-ddos rate-limit e2e")
	}

	previous := getProtectionJSON(t, client, requestBaseURL+"/api/anti-ddos/settings", host)
	settings := cloneMap(previous)
	settings["enforce_l7_rate_limit"] = true
	settings["l7_requests_per_second"] = 1
	settings["l7_burst"] = 1
	settings["l7_status_code"] = http.StatusTooManyRequests
	putProtectionJSON(t, client, requestBaseURL+"/api/anti-ddos/settings", host, settings)
	if revisionID := e2eCompileAndApply(t, client, requestBaseURL, host); revisionID == "" {
		t.Fatal("compile/apply with global L7 rate limiting returned an empty revision ID")
	}
	t.Cleanup(func() {
		putProtectionJSON(t, client, requestBaseURL+"/api/anti-ddos/settings", host, previous)
		_ = e2eCompileAndApply(t, client, requestBaseURL, host)
	})

	for attempt := 0; attempt < 8; attempt++ {
		resp := requestE2EJSON(t, client, http.MethodGet, requestBaseURL+"/api/auth/me", host, nil)
		body := mustReadBody(t, resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusTooManyRequests {
			t.Fatalf("management /api/auth/me was globally rate-limited on attempt %d: %s", attempt+1, body)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("management /api/auth/me status=%d on attempt %d: %s", resp.StatusCode, attempt+1, body)
		}
	}
}
