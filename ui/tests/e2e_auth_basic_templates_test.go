package tests

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// TestE2EBasicAuthTemplates proves that Basic Auth appearances can be previewed,
// persisted, compiled, and used by the runtime verification endpoint.
func TestE2EBasicAuthTemplates(t *testing.T) {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	if baseURL == "" {
		t.Skip("WAF_E2E_BASE_URL not set; skipping Basic Auth templates e2e")
	}
	client, requestBaseURL, requestHostOverride := newE2EClientAndBase(t, baseURL)
	loginE2EUser(t, client, requestBaseURL, requestHostOverride)

	for _, variant := range []string{"v1", "v2", "v3", "v4", "v5", "v6", "v7", "v8", "v9"} {
		resp := getWithAuth(t, client, requestBaseURL+"/api/error-pages/preview/auth-"+variant, requestHostOverride)
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), `body class="`+variant+`"`) {
			t.Fatalf("preview %s: status=%d does not render selected variant", variant, resp.StatusCode)
		}
	}

	siteID := e2eGetFirstSiteID(t, client, requestBaseURL, requestHostOverride)
	if siteID == "" {
		t.Skip("no sites configured; skipping Basic Auth template switch")
	}
	original := e2eGetEasyProfile(t, client, requestBaseURL, requestHostOverride, siteID)
	if original == nil {
		t.Skip("no easy profile for site; skipping Basic Auth template switch")
	}
	defer func() {
		response := postJSON(t, client, requestBaseURL+"/api/easy-site-profiles/"+siteID, requestHostOverride, original)
		_ = response.Body.Close()
		e2eCompileAndApply(t, client, requestBaseURL, requestHostOverride)
	}()

	profile := e2eGetEasyProfile(t, client, requestBaseURL, requestHostOverride, siteID)
	auth, _ := profile["security_auth_basic"].(map[string]any)
	if auth == nil {
		auth = map[string]any{}
		profile["security_auth_basic"] = auth
	}
	const username, password = "e2e-auth-template", "e2e-auth-template-secret"
	auth["use_auth_basic"] = true
	auth["auth_mode"] = "basic"
	auth["auth_order"] = "auth_first"
	auth["auth_basic_location"] = "sitewide"
	auth["auth_basic_text"] = "E2E protected area"
	auth["auth_basic_template"] = "v6"
	auth["auth_basic_user"] = username
	auth["auth_basic_password"] = password
	auth["users"] = []map[string]any{{"username": username, "password": password, "enabled": true}}

	saved := postJSON(t, client, requestBaseURL+"/api/easy-site-profiles/"+siteID, requestHostOverride, profile)
	_ = saved.Body.Close()
	if saved.StatusCode != http.StatusOK && saved.StatusCode != http.StatusCreated {
		t.Fatalf("save selected Basic Auth template: status=%d", saved.StatusCode)
	}
	reloaded := e2eGetEasyProfile(t, client, requestBaseURL, requestHostOverride, siteID)
	if got, _ := reloaded["security_auth_basic"].(map[string]any)["auth_basic_template"].(string); got != "v6" {
		t.Fatalf("saved auth_basic_template: want v6, got %q", got)
	}
	e2eCompileAndApply(t, client, requestBaseURL, requestHostOverride)

	runtimeURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_AUTH_BASE_URL")), "/")
	if runtimeURL == "" {
		t.Skip("WAF_E2E_AUTH_BASE_URL not set; saved and compiled template but skipping runtime login flow")
	}
	runtimeClient := newE2EHTTPClient(runtimeURL, true)
	page, err := runtimeClient.Get(runtimeURL + "/auth/login?return_uri=/")
	if err != nil {
		t.Fatalf("get Basic Auth page: %v", err)
	}
	pageBody, _ := io.ReadAll(page.Body)
	_ = page.Body.Close()
	if page.StatusCode != http.StatusOK || !strings.Contains(string(pageBody), `body class="v6"`) || !strings.Contains(string(pageBody), "logo800x300_no-text.png") {
		t.Fatalf("selected runtime page did not render v6 with logo: status=%d", page.StatusCode)
	}
	req, err := http.NewRequest(http.MethodPost, runtimeURL+"/auth/verify/basic", nil)
	if err != nil {
		t.Fatalf("create Basic Auth request: %v", err)
	}
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":wrong-password")))
	failed, err := runtimeClient.Do(req)
	if err != nil {
		t.Fatalf("submit invalid Basic Auth credentials: %v", err)
	}
	_, _ = io.ReadAll(failed.Body)
	_ = failed.Body.Close()
	if failed.StatusCode != http.StatusUnauthorized {
		t.Fatalf("invalid Basic Auth verification: want 401, got %d", failed.StatusCode)
	}
	req, err = http.NewRequest(http.MethodPost, runtimeURL+"/auth/verify/basic", nil)
	if err != nil {
		t.Fatalf("create valid Basic Auth request: %v", err)
	}
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":"+password)))
	response, err := runtimeClient.Do(req)
	if err != nil {
		t.Fatalf("submit Basic Auth credentials: %v", err)
	}
	_, _ = io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("Basic Auth verification: want 204, got %d", response.StatusCode)
	}
	e2eWaitForRequestTelemetry(t, client, requestBaseURL, requestHostOverride, "/auth/verify/basic", http.StatusUnauthorized, "auth", "security")
	e2eWaitForRequestTelemetry(t, client, requestBaseURL, requestHostOverride, "/auth/verify/basic", http.StatusNoContent, "", "request")
}

func e2eWaitForRequestTelemetry(t *testing.T, client *http.Client, baseURL, hostOverride, uri string, status int, reason, rowType string) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		resp := getWithAuth(t, client, baseURL+"/api/requests?limit=1000&retention_days=1", hostOverride)
		var rows []map[string]any
		decodeErr := json.NewDecoder(resp.Body).Decode(&rows)
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusOK && decodeErr == nil {
			for _, row := range rows {
				entry, _ := row["entry"].(map[string]any)
				if entry["uri"] == uri && e2eDashboardAsInt(t, entry["status"]) == status && row["security_reason"] == reason && row["row_type"] == rowType {
					return
				}
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("request telemetry missing uri=%s status=%d reason=%q type=%q", uri, status, reason, rowType)
}
