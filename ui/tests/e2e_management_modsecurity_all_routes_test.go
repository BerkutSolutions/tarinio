package tests

import (
	"net/http"
	"os"
	"strings"
	"testing"

	"waf/control-plane/apiroutes"
)

const e2eManagementModSecurityProbeStatus = 418

// TestE2EAdminPanelModSecurityBypassesEveryAdministrativeRoute verifies the
// dedicated e2e-management.test panel vhost through nginx, not control-plane.
// The Host override models an external management domain without requiring DNS.
func TestE2EAdminPanelModSecurityBypassesEveryAdministrativeRoute(t *testing.T) {
	runtimeURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_RUNTIME_URL")), "/")
	if runtimeURL == "" {
		t.Skip("WAF_E2E_RUNTIME_URL is not set; skipping full management ModSecurity e2e")
	}
	activeRuntimeURL := runtimeURL
	if httpsURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_RUNTIME_HTTPS_URL")), "/"); httpsURL != "" {
		activeRuntimeURL = httpsURL
	}
	client, baseURL, _ := newE2EClientAndBase(t, runtimeURL)
	host := strings.TrimSpace(os.Getenv("WAF_E2E_MANAGEMENT_HOST"))
	if host == "" {
		host = "e2e-management.test"
	}
	loginE2EUser(t, client, baseURL, host)
	managementSettings := getWithAuth(t, client, baseURL+"/api/settings/management-hosts", host)
	managementBody := mustReadBody(t, managementSettings.Body)
	if managementSettings.StatusCode != http.StatusOK || !strings.Contains(managementBody, `"`+host+`"`) {
		t.Fatalf("e2e panel host %q is not persisted as a management host: status=%d body=%s", host, managementSettings.StatusCode, managementBody)
	}
	siteID := e2eManagementSiteID(t, client, baseURL, host, "http://"+host)
	original := e2eGetProfile(t, client, baseURL, host, siteID)
	protected := cloneMap(original)
	front := mapGetOrCreate(protected, "front_service")
	front["security_mode"] = "block"
	modsec := mapGetOrCreate(protected, "security_modsecurity")
	modsec["use_modsecurity"] = true
	modsec["use_modsecurity_crs_plugins"] = true
	modsec["use_modsecurity_custom_configuration"] = true
	modsec["custom_configuration"] = map[string]any{
		"path":    "modsec/e2e-management-all-routes.conf",
		"content": `SecRule ARGS:e2e_modsec_probe "@streq enabled" "id:100199,phase:2,deny,status:418,log"`,
	}
	antibot := mapGetOrCreate(protected, "security_antibot")
	antibot["enabled"] = true
	antibot["antibot_challenge"] = "javascript"
	antibot["antibot_uri"] = "/challenge"
	t.Cleanup(func() {
		e2ePutProfile(t, client, baseURL, host, siteID, original)
		_ = e2eCompileAndApply(t, client, baseURL, host)
	})
	e2ePutProfile(t, client, baseURL, host, siteID, protected)
	if revisionID := e2eCompileAndApply(t, client, baseURL, host); revisionID == "" {
		t.Fatal("enable ModSecurity probe and apply management revision failed")
	}
	// A new browser must see a challenge before the login HTML. Once verification
	// sets the anti-bot cookie, the original page and its module must load without
	// another challenge or a substituted HTML response.
	anonymousClient, anonymousBaseURL, _ := newE2EClientAndBase(t, activeRuntimeURL)
	loginResp, err := doE2ERequest(anonymousClient, http.MethodGet, anonymousBaseURL+"/login", host, "text/html", nil, false)
	if err != nil {
		t.Fatalf("open unauthenticated management login: %v", err)
	}
	if loginResp.StatusCode != http.StatusFound && loginResp.StatusCode != http.StatusSeeOther {
		t.Fatalf("unauthenticated management login must redirect to challenge: status=%d body=%s", loginResp.StatusCode, mustReadBody(t, loginResp.Body))
	}
	challengeLocation := strings.TrimSpace(loginResp.Header.Get("Location"))
	if !strings.Contains(challengeLocation, "/challenge") {
		t.Fatalf("unauthenticated management login must redirect to challenge, got %q", challengeLocation)
	}
	if !strings.Contains(strings.ToLower(loginResp.Header.Get("Cache-Control")), "no-store") {
		t.Fatalf("management login redirect must prevent stale HTML caching, got Cache-Control=%q", loginResp.Header.Get("Cache-Control"))
	}
	_ = loginResp.Body.Close()
	challengeLocation = localAntibotLocation(challengeLocation)

	challengePageURL := absolutizeLocation(anonymousBaseURL, challengeLocation)
	challengeResp, err := doE2ERequest(anonymousClient, http.MethodGet, challengePageURL, host, "text/html", nil, false)
	if err != nil {
		t.Fatalf("open management challenge page: %v", err)
	}
	challengeBody := strings.ToLower(mustReadBody(t, challengeResp.Body))
	if challengeResp.StatusCode != http.StatusOK || (!strings.Contains(challengeBody, "verification") && !strings.Contains(challengeBody, "challenge")) {
		t.Fatalf("management challenge page contract mismatch: status=%d body=%s", challengeResp.StatusCode, challengeBody)
	}
	verifyURL, err := buildVerifyURL(anonymousBaseURL, challengeLocation, antibotVerifyURI("/challenge"))
	if err != nil {
		t.Fatalf("build management challenge verify URL: %v", err)
	}
	verifyResp, err := doE2ERequest(anonymousClient, http.MethodGet, verifyURL, host, "text/html", nil, false)
	if err != nil {
		t.Fatalf("verify management challenge: %v", err)
	}
	if verifyResp.StatusCode != http.StatusNoContent {
		t.Fatalf("management challenge verification must set the anti-bot cookie: status=%d body=%s", verifyResp.StatusCode, mustReadBody(t, verifyResp.Body))
	}
	_ = verifyResp.Body.Close()

	verifiedLoginResp, err := doE2ERequest(anonymousClient, http.MethodGet, anonymousBaseURL+"/login", host, "text/html", nil, false)
	if err != nil {
		t.Fatalf("open verified management login: %v", err)
	}
	verifiedLoginBody := strings.ToLower(mustReadBody(t, verifiedLoginResp.Body))
	if verifiedLoginResp.StatusCode < http.StatusOK || verifiedLoginResp.StatusCode >= http.StatusMultipleChoices || strings.Contains(verifiedLoginBody, "challenge") {
		t.Fatalf("verified management login must return UI HTML: status=%d body=%s", verifiedLoginResp.StatusCode, verifiedLoginBody)
	}

	assetResp, err := doE2ERequest(anonymousClient, http.MethodGet, anonymousBaseURL+"/static/js/login.js", host, "text/javascript,*/*", nil, false)
	if err != nil {
		t.Fatalf("request verified management login module: %v", err)
	}
	assetBody := mustReadBody(t, assetResp.Body)
	assetContentType := strings.ToLower(assetResp.Header.Get("Content-Type"))
	if assetResp.StatusCode != http.StatusOK || !strings.Contains(assetContentType, "javascript") || strings.Contains(strings.ToLower(assetBody), "<html") {
		t.Fatalf("verified management login module must be JavaScript, got status=%d content-type=%q body=%s", assetResp.StatusCode, assetContentType, assetBody)
	}
	if activeRuntimeURL != runtimeURL {
		client, baseURL, _ = newE2EClientAndBase(t, activeRuntimeURL)
		loginE2EUser(t, client, baseURL, host)
	}
	for _, path := range apiroutes.AdministrativePaths {
		t.Run(strings.Trim(strings.ReplaceAll(path, "/", "_"), "_"), func(t *testing.T) {
			response := requestE2EJSON(t, client, http.MethodOptions, baseURL+path+"?e2e_modsec_probe=enabled", host, nil)
			defer response.Body.Close()
			if response.StatusCode == e2eManagementModSecurityProbeStatus {
				t.Fatalf("ModSecurity blocked administrative route %s on external management host", path)
			}
		})
	}
}
