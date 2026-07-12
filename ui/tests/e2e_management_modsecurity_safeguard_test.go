package tests

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

func TestE2EManagementModSecuritySafeguardKeepsAppPingAvailable(t *testing.T) {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	if baseURL == "" {
		t.Skip("WAF_E2E_BASE_URL is not set; skipping management ModSecurity safeguard e2e")
	}

	client, requestBaseURL, requestHostOverride := newE2EClientAndBase(t, baseURL)
	loginE2EUser(t, client, requestBaseURL, requestHostOverride)
	siteID := e2eManagementSiteID(t, client, requestBaseURL, requestHostOverride, baseURL)
	original := e2eGetProfile(t, client, requestBaseURL, requestHostOverride, siteID)
	t.Cleanup(func() {
		e2ePutProfile(t, client, requestBaseURL, requestHostOverride, siteID, original)
		if revisionID := e2eCompileAndApply(t, client, requestBaseURL, requestHostOverride); revisionID == "" {
			t.Errorf("restore management profile: compile+apply returned an empty revision ID")
		}
	})

	profile := cloneMap(original)
	front := mapGetOrCreate(profile, "front_service")
	front["security_mode"] = "block"
	modsec := mapGetOrCreate(profile, "security_modsecurity")
	modsec["use_modsecurity"] = true
	modsec["use_modsecurity_crs_plugins"] = false
	modsec["use_modsecurity_custom_configuration"] = true
	modsec["custom_configuration"] = map[string]any{
		"path":    "modsec/e2e-management-ping.conf",
		"content": `SecRule REQUEST_URI "@streq /api/app/ping" "id:100010,phase:2,deny,status:403,log"`,
	}

	e2ePutProfile(t, client, requestBaseURL, requestHostOverride, siteID, profile)
	if revisionID := e2eCompileAndApply(t, client, requestBaseURL, requestHostOverride); revisionID == "" {
		t.Fatal("compile+apply returned an empty revision ID")
	}
	time.Sleep(2 * time.Second)

	req, err := http.NewRequest(http.MethodPost, requestBaseURL+"/api/app/ping", nil)
	if err != nil {
		t.Fatalf("create ping request: %v", err)
	}
	req.Header.Set("X-Berkut-Background", "1")
	if requestHostOverride != "" {
		req.Host = requestHostOverride
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("management ping request: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("management /api/app/ping must bypass ModSecurity: status=%d body=%s", resp.StatusCode, body)
	}
}

func e2eManagementSiteID(t *testing.T, client *http.Client, requestBaseURL, requestHostOverride, originalBaseURL string) string {
	t.Helper()
	if configured := strings.TrimSpace(os.Getenv("WAF_E2E_MANAGEMENT_SITE_ID")); configured != "" {
		return configured
	}
	if configured := strings.TrimSpace(os.Getenv("CONTROL_PLANE_DEV_FAST_START_MANAGEMENT_SITE_ID")); configured != "" {
		return configured
	}
	resp := getWithAuth(t, client, requestBaseURL+"/api/sites", requestHostOverride)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list sites: status=%d body=%s", resp.StatusCode, mustReadBody(t, resp.Body))
	}
	var sites []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&sites); err != nil {
		t.Fatalf("decode sites: %v", err)
	}
	for _, site := range sites {
		if id := strings.TrimSpace(stringValue(site["id"])); id == "control-plane-access" || id == "control-plane" || id == "ui" {
			return id
		}
	}
	parsed, err := url.Parse(originalBaseURL)
	if err != nil {
		t.Fatalf("parse WAF_E2E_BASE_URL: %v", err)
	}
	for _, site := range sites {
		if strings.EqualFold(strings.TrimSpace(stringValue(site["primary_host"])), parsed.Hostname()) {
			return strings.TrimSpace(stringValue(site["id"]))
		}
	}
	t.Skip("management site not found; set WAF_E2E_MANAGEMENT_SITE_ID to run safeguard e2e")
	return ""
}

func stringValue(value any) string {
	valueString, _ := value.(string)
	return valueString
}
