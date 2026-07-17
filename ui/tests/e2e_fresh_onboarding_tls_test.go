package tests

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestE2EFreshOnboardingSelfSignedTLS(t *testing.T) {
	if strings.TrimSpace(os.Getenv("WAF_E2E_FRESH_ONBOARDING")) != "1" {
		t.Skip("run scripts/run-e2e-tests.ps1 -FreshOnboarding to exercise clean onboarding")
	}
	runtimeURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_RUNTIME_URL")), "/")
	httpsURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_RUNTIME_HTTPS_URL")), "/")
	if runtimeURL == "" || httpsURL == "" {
		t.Fatal("WAF_E2E_RUNTIME_URL and WAF_E2E_RUNTIME_HTTPS_URL are required")
	}

	client, requestBaseURL, _ := newE2EClientAndBase(t, runtimeURL)
	const host = "e2e-onboarding.test"
	waitForFreshOnboarding(t, client, requestBaseURL, host)

	setup := freshOnboardingJSON(t, requestE2EJSON(t, client, http.MethodGet, requestBaseURL+"/api/setup/status", host, nil))
	if needsBootstrap, _ := setup["needs_bootstrap"].(bool); !needsBootstrap {
		t.Fatalf("fresh stack must require onboarding, got %v", setup)
	}

	bootstrap := postJSON(t, client, requestBaseURL+"/api/auth/bootstrap", host, map[string]any{
		"username": "e2e-onboarding-admin",
		"email":    "e2e-onboarding@example.test",
		"password": "e2e-onboarding-password-1234",
	})
	assertFreshOnboardingStatus(t, bootstrap, "bootstrap admin", http.StatusOK)

	me := requestE2EJSON(t, client, http.MethodGet, requestBaseURL+"/api/auth/me", host, nil)
	assertFreshOnboardingStatus(t, me, "use bootstrap session", http.StatusOK)

	const siteID = "e2e-onboarding-site"
	const upstreamID = "e2e-onboarding-upstream"
	const certID = "e2e-onboarding-tls"
	createSite := postJSON(t, client, requestBaseURL+"/api/sites?auto_apply=false", host, map[string]any{
		"id": siteID, "primary_host": host, "enabled": true, "listen_http": true, "listen_https": true,
	})
	assertFreshOnboardingStatus(t, createSite, "create onboarding site", http.StatusCreated, http.StatusOK)
	createUpstream := postJSON(t, client, requestBaseURL+"/api/upstreams?auto_apply=false", host, map[string]any{
		"id": upstreamID, "site_id": siteID, "name": upstreamID, "scheme": "http", "host": "ui", "port": 80, "base_path": "/",
	})
	assertFreshOnboardingStatus(t, createUpstream, "create onboarding upstream", http.StatusCreated, http.StatusOK)
	updateSite := requestE2EJSON(t, client, http.MethodPut, requestBaseURL+"/api/sites/"+siteID+"?auto_apply=false", host, map[string]any{
		"id": siteID, "primary_host": host, "enabled": true, "listen_http": true, "listen_https": true, "default_upstream_id": upstreamID,
	})
	assertFreshOnboardingStatus(t, updateSite, "bind onboarding upstream", http.StatusOK)

	management := freshOnboardingJSON(t, requestE2EJSON(t, client, http.MethodGet, requestBaseURL+"/api/settings/management-hosts", host, nil))
	version, _ := management["version"].(float64)
	setManagement := requestE2EJSON(t, client, http.MethodPut, requestBaseURL+"/api/settings/management-hosts", host, map[string]any{
		"management_hosts": []string{host}, "version": int(version),
	})
	assertFreshOnboardingStatus(t, setManagement, "persist onboarding management host", http.StatusOK)

	issue := postJSON(t, client, requestBaseURL+"/api/certificates/self-signed/issue", host, map[string]any{
		"certificate_id": certID, "common_name": host, "san_list": []string{},
	})
	assertFreshOnboardingStatus(t, issue, "issue onboarding self-signed certificate", http.StatusCreated, http.StatusOK)
	bindTLS := postJSON(t, client, requestBaseURL+"/api/tls-configs?auto_apply=false", host, map[string]any{
		"site_id": siteID, "certificate_id": certID,
	})
	assertFreshOnboardingStatus(t, bindTLS, "bind onboarding certificate", http.StatusCreated, http.StatusOK)

	if revisionID := e2eCompileAndApply(t, client, requestBaseURL, host); revisionID == "" {
		t.Fatal("compile/apply onboarding revision failed")
	}

	httpsClient, httpsBaseURL, _ := newE2EClientAndBase(t, httpsURL)
	waitForFreshHTTPSLogin(t, httpsClient, httpsBaseURL, host)
	login := postJSON(t, httpsClient, httpsBaseURL+"/api/auth/login", host, map[string]any{
		"username": "e2e-onboarding-admin", "password": "e2e-onboarding-password-1234",
	})
	assertFreshOnboardingStatus(t, login, "login through self-signed HTTPS", http.StatusOK)
	secureMe := requestE2EJSON(t, httpsClient, http.MethodGet, httpsBaseURL+"/api/auth/me", host, nil)
	assertFreshOnboardingStatus(t, secureMe, "authenticated HTTPS control plane", http.StatusOK)
}

func freshOnboardingJSON(t *testing.T, response *http.Response) map[string]any {
	t.Helper()
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: status=%d body=%s", response.Request.URL, response.StatusCode, mustReadBody(t, response.Body))
	}
	var payload map[string]any
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode %s: %v", response.Request.URL, err)
	}
	return payload
}

func assertFreshOnboardingStatus(t *testing.T, response *http.Response, action string, allowed ...int) {
	t.Helper()
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	for _, status := range allowed {
		if response.StatusCode == status {
			return
		}
	}
	t.Fatalf("%s: status=%d body=%s", action, response.StatusCode, body)
}

func waitForFreshOnboarding(t *testing.T, client *http.Client, baseURL, host string) {
	t.Helper()
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		request, err := http.NewRequest(http.MethodGet, baseURL+"/onboarding/user-creation", nil)
		if err != nil {
			t.Fatalf("build fresh onboarding request: %v", err)
		}
		request.Host = host
		response, err := client.Do(request)
		if err == nil && response != nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(time.Second)
	}
	t.Fatal("fresh runtime bootstrap did not expose onboarding")
}

func waitForFreshHTTPSLogin(t *testing.T, client *http.Client, baseURL, host string) {
	t.Helper()
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		request, err := http.NewRequest(http.MethodGet, baseURL+"/login", nil)
		if err != nil {
			t.Fatalf("build HTTPS login request: %v", err)
		}
		request.Host = host
		response, err := client.Do(request)
		if err == nil && response != nil {
			body := mustReadBody(t, response.Body)
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK && strings.Contains(body, "login") {
				return
			}
		}
		time.Sleep(time.Second)
	}
	t.Fatal("self-signed HTTPS login did not become available")
}
