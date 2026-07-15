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
	t.Cleanup(func() {
		e2ePutProfile(t, client, baseURL, host, siteID, original)
		_ = e2eCompileAndApply(t, client, baseURL, host)
	})
	e2ePutProfile(t, client, baseURL, host, siteID, protected)
	if revisionID := e2eCompileAndApply(t, client, baseURL, host); revisionID == "" {
		t.Fatal("enable ModSecurity probe and apply management revision failed")
	}
	// The login HTML is protected, but its module, stylesheet and icons must not
	// receive a challenge document. Otherwise browsers reject the HTML response
	// as a JavaScript module because its MIME type is text/html.
	anonymousClient, anonymousBaseURL, _ := newE2EClientAndBase(t, activeRuntimeURL)
	assetResp, err := doE2ERequest(anonymousClient, http.MethodGet, anonymousBaseURL+"/static/js/login.js", host, "text/javascript,*/*", nil, false)
	if err != nil {
		t.Fatalf("request unauthenticated management login module: %v", err)
	}
	assetBody := mustReadBody(t, assetResp.Body)
	assetContentType := strings.ToLower(assetResp.Header.Get("Content-Type"))
	if assetResp.StatusCode != http.StatusOK || !strings.Contains(assetContentType, "javascript") || strings.Contains(strings.ToLower(assetBody), "<html") {
		t.Fatalf("management login module must bypass challenge: status=%d content-type=%q body=%s", assetResp.StatusCode, assetContentType, assetBody)
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
