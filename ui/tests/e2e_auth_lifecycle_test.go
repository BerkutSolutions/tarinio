package tests

import (
	"encoding/base64"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
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
	siteID := e2eUniqueID(t, "e2e-auth")
	runtimeHost = siteID + ".test"
	upstreamID := siteID + "-upstream"
	for _, endpoint := range []string{
		"/api/sites/" + siteID + "?auto_apply=false",
		"/api/upstreams/" + upstreamID + "?auto_apply=false",
	} {
		resp := requestE2EJSON(t, adminClient, http.MethodDelete, requestBaseURL+endpoint, hostOverride, nil)
		_ = resp.Body.Close()
	}
	t.Cleanup(func() {
		for _, endpoint := range []string{
			"/api/sites/" + siteID + "?auto_apply=false",
			"/api/upstreams/" + upstreamID + "?auto_apply=false",
		} {
			resp := requestE2EJSON(t, adminClient, http.MethodDelete, requestBaseURL+endpoint, hostOverride, nil)
			_ = resp.Body.Close()
		}
		e2eCompileAndApply(t, adminClient, requestBaseURL, hostOverride)
	})
	resp := postJSON(t, adminClient, requestBaseURL+"/api/sites?auto_apply=false", hostOverride, map[string]any{
		"id": siteID, "primary_host": runtimeHost, "enabled": true, "listen_http": true, "listen_https": false, "use_easy_config": true, "default_upstream_id": upstreamID,
	})
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("create lifecycle site: status=%d body=%s", resp.StatusCode, mustReadBody(t, resp.Body))
	}
	_ = resp.Body.Close()
	resp = postJSON(t, adminClient, requestBaseURL+"/api/upstreams?auto_apply=false", hostOverride, map[string]any{
		"id": upstreamID, "site_id": siteID, "name": upstreamID, "scheme": "http", "host": "upstream-echo", "port": 8888, "base_path": "/",
	})
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("create lifecycle upstream: status=%d body=%s", resp.StatusCode, mustReadBody(t, resp.Body))
	}
	_ = resp.Body.Close()

	const disabledUser, disabledPassword = "e2e-disabled", "disabled-password"
	const activeUser, initialPassword, rotatedPassword = "e2e-active", "initial-password", "rotated-password"
	profile := e2eGetEasyProfile(t, adminClient, requestBaseURL, hostOverride, siteID)
	httpBehavior, _ := profile["http_behavior"].(map[string]any)
	if httpBehavior == nil {
		httpBehavior = map[string]any{}
		profile["http_behavior"] = httpBehavior
	}
	httpBehavior["allowed_methods"] = []string{"GET", "POST", "HEAD", "OPTIONS"}
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
	e2eWaitForMultisiteHost(t, runtimeURL, runtimeHost)

	if resp := e2eBasicVerify(t, runtimeURL, runtimeHost, disabledUser, disabledPassword); resp.StatusCode != http.StatusUnauthorized {
		body := readAndClose(t, resp.Body)
		t.Fatalf("disabled Basic Auth user must be rejected: status=%d body=%s", resp.StatusCode, body)
	} else {
		_ = resp.Body.Close()
	}
	e2eWaitForRequestTelemetry(t, adminClient, requestBaseURL, hostOverride, "/auth/verify/basic", http.StatusUnauthorized, "auth", "security")
	e2eAssertRequestTelemetryRedacted(t, adminClient, requestBaseURL, hostOverride,
		disabledPassword,
		base64.StdEncoding.EncodeToString([]byte(disabledUser+":"+disabledPassword)),
	)
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
	// auth_basic reads its credential file per request, while the session cookie
	// guard becomes active after nginx reloads the new revision. Wait for the
	// freshly issued cookie so the revocation assertion observes one revision.
	e2eWaitForRotatedBasicSession(t, runtimeURL, runtimeHost, activeUser, rotatedPassword, cookie)

	stale := e2eRequestWithCookie(t, runtimeURL+"/", runtimeHost, cookie)
	if stale.StatusCode != http.StatusFound {
		body := readAndClose(t, stale.Body)
		t.Fatalf("old Basic Auth session must be revoked after profile apply: status=%d body=%s", stale.StatusCode, body)
	}
	_ = stale.Body.Close()
}

func e2eAssertRequestTelemetryRedacted(t *testing.T, client *http.Client, baseURL, hostOverride string, forbiddenValues ...string) {
	t.Helper()
	resp := getWithAuth(t, client, baseURL+"/api/requests?limit=1000&retention_days=1", hostOverride)
	body := readAndClose(t, resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get request telemetry for redaction check: status=%d body=%s", resp.StatusCode, body)
	}
	for _, forbidden := range forbiddenValues {
		if forbidden != "" && strings.Contains(body, forbidden) {
			t.Fatalf("request telemetry must redact credential material %q", forbidden)
		}
	}
}

func e2eWaitForRotatedBasicSession(t *testing.T, runtimeURL, host, username, password, staleCookie string) {
	t.Helper()
	stalePair := strings.SplitN(staleCookie, ";", 2)[0]
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		resp := e2eBasicVerify(t, runtimeURL, host, username, password)
		setCookie := resp.Header.Get("Set-Cookie")
		_ = readAndClose(t, resp.Body)
		if resp.StatusCode == http.StatusNoContent && setCookie != "" && strings.SplitN(setCookie, ";", 2)[0] != stalePair {
			return
		}
		time.Sleep(250 * time.Millisecond)
	}
	t.Fatal("rotated Basic Auth revision did not issue a fresh session cookie")
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
	deadline := time.Now().Add(30 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		req, err := http.NewRequest(http.MethodPost, runtimeURL+"/auth/verify/basic", nil)
		if err != nil {
			t.Fatalf("create Basic Auth verification request: %v", err)
		}
		if host != "" {
			req.Host = host
		}
		req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":"+password)))
		resp, err := newE2EHTTPClient(runtimeURL, false).Do(req)
		if err == nil {
			return resp
		}
		lastErr = err
		time.Sleep(250 * time.Millisecond)
	}
	t.Fatalf("Basic Auth verification request did not recover after runtime reload: %v", lastErr)
	return nil
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
