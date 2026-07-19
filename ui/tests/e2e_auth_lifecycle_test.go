package tests

import (
	"encoding/base64"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestE2EBasicAuthLifecycle(t *testing.T) {
	panelURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	runtimeURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_RUNTIME_URL")), "/")
	runtimeHost := strings.TrimSpace(firstNonEmpty(os.Getenv("WAF_E2E_AUTH_HOST"), os.Getenv("WAF_E2E_MANAGEMENT_HOST")))
	if panelURL == "" || runtimeURL == "" {
		t.Skip("Basic Auth panel or runtime URL is not configured")
	}
	t.Logf("Basic Auth lifecycle runtime URL: %s", runtimeURL)
	adminClient, requestBaseURL, hostOverride := newE2EClientAndBase(t, panelURL)
	loginE2EUser(t, adminClient, requestBaseURL, hostOverride)
	siteID := e2eGetFirstSiteID(t, adminClient, requestBaseURL, hostOverride)
	if siteID == "" {
		t.Skip("no site is configured for the Basic Auth lifecycle test")
	}
	original := e2eGetEasyProfile(t, adminClient, requestBaseURL, hostOverride, siteID)
	if original == nil {
		t.Skip("no easy profile is configured for the Basic Auth lifecycle test")
	}
	t.Cleanup(func() {
		resp := postJSON(t, adminClient, requestBaseURL+"/api/easy-site-profiles/"+siteID, hostOverride, original)
		_ = resp.Body.Close()
		e2eCompileAndApply(t, adminClient, requestBaseURL, hostOverride)
	})

	const disabledUser, disabledPassword = "e2e-disabled", "disabled-password"
	const activeUser, initialPassword, rotatedPassword = "e2e-active", "initial-password", "rotated-password"
	profile := e2eGetEasyProfile(t, adminClient, requestBaseURL, hostOverride, siteID)
	auth, _ := profile["security_auth_basic"].(map[string]any)
	if auth == nil {
		auth = map[string]any{}
		profile["security_auth_basic"] = auth
	}
	auth["use_auth_basic"] = true
	auth["auth_mode"] = "basic"
	auth["auth_order"] = "auth_first"
	auth["auth_basic_location"] = "sitewide"
	auth["auth_basic_user"] = activeUser
	auth["auth_basic_password"] = initialPassword
	auth["session_inactivity_minutes"] = 5
	auth["users"] = []map[string]any{
		{"username": disabledUser, "password": disabledPassword, "enabled": false},
		{"username": activeUser, "password": initialPassword, "enabled": true},
	}
	e2eSaveAndApplyAuthProfile(t, adminClient, requestBaseURL, hostOverride, siteID, profile)

	if resp := e2eBasicVerify(t, runtimeURL, runtimeHost, disabledUser, disabledPassword); resp.StatusCode != http.StatusUnauthorized {
		body := readAndClose(t, resp.Body)
		t.Fatalf("disabled Basic Auth user must be rejected: status=%d body=%s", resp.StatusCode, body)
	} else {
		_ = resp.Body.Close()
	}
	verified := e2eBasicVerify(t, runtimeURL, runtimeHost, activeUser, initialPassword)
	if verified.StatusCode != http.StatusNoContent {
		body := readAndClose(t, verified.Body)
		t.Fatalf("active Basic Auth user must be accepted: status=%d body=%s", verified.StatusCode, body)
	}
	cookie := verified.Header.Get("Set-Cookie")
	_ = verified.Body.Close()
	if !strings.Contains(cookie, "waf_auth_") || !strings.Contains(cookie, "Max-Age=300") {
		t.Fatalf("Basic Auth verification must issue a five-minute session cookie, got %q", cookie)
	}

	auth["auth_basic_password"] = rotatedPassword
	auth["users"] = []map[string]any{
		{"username": disabledUser, "password": disabledPassword, "enabled": false},
		{"username": activeUser, "password": rotatedPassword, "enabled": true},
	}
	e2eSaveAndApplyAuthProfile(t, adminClient, requestBaseURL, hostOverride, siteID, profile)
	if resp := e2eBasicVerify(t, runtimeURL, runtimeHost, activeUser, initialPassword); resp.StatusCode != http.StatusUnauthorized {
		body := readAndClose(t, resp.Body)
		t.Fatalf("rotated-out Basic Auth password must be rejected: status=%d body=%s", resp.StatusCode, body)
	} else {
		_ = resp.Body.Close()
	}
	if resp := e2eBasicVerify(t, runtimeURL, runtimeHost, activeUser, rotatedPassword); resp.StatusCode != http.StatusNoContent {
		body := readAndClose(t, resp.Body)
		t.Fatalf("rotated Basic Auth password must be accepted: status=%d body=%s", resp.StatusCode, body)
	} else {
		_ = resp.Body.Close()
	}

	stale := e2eRequestWithCookie(t, runtimeURL+"/", runtimeHost, cookie)
	if stale.StatusCode != http.StatusFound {
		body := readAndClose(t, stale.Body)
		t.Fatalf("old Basic Auth session must be revoked after profile apply: status=%d body=%s", stale.StatusCode, body)
	}
	_ = stale.Body.Close()
}

func e2eSaveAndApplyAuthProfile(t *testing.T, client *http.Client, baseURL, hostOverride, siteID string, profile map[string]any) {
	t.Helper()
	resp := postJSON(t, client, baseURL+"/api/easy-site-profiles/"+siteID, hostOverride, profile)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		t.Fatalf("save Basic Auth lifecycle profile: status=%d body=%s", resp.StatusCode, mustReadBody(t, resp.Body))
	}
	_ = resp.Body.Close()
	if revisionID := e2eCompileAndApply(t, client, baseURL, hostOverride); revisionID == "" {
		t.Fatal("compile/apply Basic Auth lifecycle profile returned an empty revision")
	}
}

func e2eBasicVerify(t *testing.T, runtimeURL, host, username, password string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, runtimeURL+"/auth/verify/basic", nil)
	if err != nil {
		t.Fatalf("create Basic Auth verification request: %v", err)
	}
	if host != "" {
		req.Host = host
	}
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":"+password)))
	resp, err := newE2EHTTPClient(runtimeURL, false).Do(req)
	if err != nil {
		t.Fatalf("Basic Auth verification request: %v", err)
	}
	return resp
}

func e2eRequestWithCookie(t *testing.T, endpoint, host, setCookie string) *http.Response {
	t.Helper()
	pair := strings.SplitN(setCookie, ";", 2)[0]
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		t.Fatalf("create Basic Auth session request: %v", err)
	}
	if host != "" {
		req.Host = host
	}
	req.Header.Set("Cookie", pair)
	client := newE2EHTTPClient(endpoint, false)
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("send Basic Auth session request: %v", err)
	}
	return resp
}
