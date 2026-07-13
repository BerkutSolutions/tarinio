package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

const defaultE2EBrowserUserAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"

func TestE2ESmoke_LoginHealthcheckDashboard(t *testing.T) {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	if baseURL == "" {
		t.Skip("WAF_E2E_BASE_URL is not set; skipping e2e smoke test")
	}
	challengeURI := normalizeChallengeURI(strings.TrimSpace(os.Getenv("WAF_E2E_ANTIBOT_CHALLENGE_URI")))

	client, requestBaseURL, requestHostOverride := newE2EClientAndBase(t, baseURL)
	requestParsed, err := url.Parse(requestBaseURL)
	if err != nil {
		t.Fatalf("parse request base URL: %v", err)
	}
	originalParsed, err := url.Parse(baseURL)
	if err != nil {
		t.Fatalf("parse original base URL: %v", err)
	}
	username := strings.TrimSpace(os.Getenv("WAF_E2E_USERNAME"))
	if username == "" {
		username = "admin"
	}
	password := os.Getenv("WAF_E2E_PASSWORD")
	if password == "" {
		password = "admin"
	}

	if err := waitForHTTP(client, requestBaseURL+"/login", requestHostOverride, 90*time.Second); err != nil {
		t.Fatalf("ui is not ready: %v", err)
	}
	ensureManagementLoginAccess(t, client, requestBaseURL, requestHostOverride, challengeURI)

	loginPayload := map[string]any{
		"username": username,
		"password": password,
	}
	loginResp := postJSON(t, client, requestBaseURL+"/api/auth/login", requestHostOverride, loginPayload)
	if loginResp.StatusCode == http.StatusFound || loginResp.StatusCode == http.StatusForbidden || loginResp.StatusCode == http.StatusTooManyRequests {
		_ = loginResp.Body.Close()
		ensureManagementLoginAccess(t, client, requestBaseURL, requestHostOverride, challengeURI)
		loginResp = postJSON(t, client, requestBaseURL+"/api/auth/login", requestHostOverride, loginPayload)
	}
	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("login failed: status=%d body=%s", loginResp.StatusCode, mustReadBody(t, loginResp.Body))
	}
	loginBodyBytes, err := io.ReadAll(loginResp.Body)
	if err != nil {
		t.Fatalf("read login response: %v", err)
	}
	loginContentType := strings.ToLower(strings.TrimSpace(loginResp.Header.Get("Content-Type")))
	if !strings.Contains(loginContentType, "application/json") {
		bodyPreview := strings.TrimSpace(string(loginBodyBytes))
		if len(bodyPreview) > 300 {
			bodyPreview = bodyPreview[:300]
		}
		t.Fatalf("login response is not json: content-type=%q final_url=%q body_preview=%q", loginContentType, loginResp.Request.URL.String(), bodyPreview)
	}
	var loginData map[string]any
	if err := json.Unmarshal(loginBodyBytes, &loginData); err != nil {
		t.Fatalf("decode login response: %v body=%q", err, strings.TrimSpace(string(loginBodyBytes)))
	}
	if requires2FA, _ := loginData["requires_2fa"].(bool); requires2FA {
		t.Fatalf("smoke flow requires direct login without 2fa; got requires_2fa=true")
	}

	if requestHostOverride != "" {
		client.Jar.SetCookies(requestParsed, loginResp.Cookies())
	}
	cookies := client.Jar.Cookies(originalParsed)
	if len(cookies) == 0 {
		cookies = client.Jar.Cookies(requestParsed)
	}
	hasSession := false
	for _, c := range cookies {
		if strings.TrimSpace(c.Name) == "waf_session" && strings.TrimSpace(c.Value) != "" {
			hasSession = true
			break
		}
	}
	if !hasSession {
		t.Fatalf("expected waf_session cookie after login")
	}

	healthcheckPage := getWithAuth(t, client, requestBaseURL+"/healthcheck", requestHostOverride)
	if healthcheckPage.StatusCode != http.StatusOK {
		t.Fatalf("healthcheck page failed: status=%d", healthcheckPage.StatusCode)
	}
	healthcheckBody := mustReadBody(t, healthcheckPage.Body)
	if !strings.Contains(healthcheckBody, `id="healthcheck-steps"`) || !strings.Contains(healthcheckBody, `id="healthcheck-error"`) {
		t.Fatalf("healthcheck page contract mismatch: missing current page markers")
	}

	compatResp := getWithAuth(t, client, requestBaseURL+"/api/app/compat", requestHostOverride)
	if compatResp.StatusCode != http.StatusOK {
		t.Fatalf("compat api failed: status=%d body=%s", compatResp.StatusCode, mustReadBody(t, compatResp.Body))
	}
	var compatData struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.NewDecoder(compatResp.Body).Decode(&compatData); err != nil {
		t.Fatalf("decode compat response: %v", err)
	}
	if len(compatData.Items) == 0 {
		t.Fatalf("compat api returned empty modules list")
	}
	firstModuleID := strings.TrimSpace(fmt.Sprint(compatData.Items[0]["module_id"]))
	if firstModuleID == "" {
		t.Fatalf("compat api returned empty module_id")
	}

	fixResp := postJSON(t, client, requestBaseURL+"/api/app/compat/fix", requestHostOverride, map[string]any{"module_id": firstModuleID})
	if fixResp.StatusCode != http.StatusOK {
		t.Fatalf("compat fix failed: status=%d body=%s", fixResp.StatusCode, mustReadBody(t, fixResp.Body))
	}
	var fixData map[string]any
	if err := json.NewDecoder(fixResp.Body).Decode(&fixData); err != nil {
		t.Fatalf("decode compat fix response: %v", err)
	}
	if ok, _ := fixData["ok"].(bool); !ok {
		t.Fatalf("compat fix response expected ok=true, got: %#v", fixData)
	}

	dashboardPage := getWithAuth(t, client, requestBaseURL+"/dashboard", requestHostOverride)
	if dashboardPage.StatusCode != http.StatusOK {
		t.Fatalf("dashboard page failed: status=%d body=%s", dashboardPage.StatusCode, mustReadBody(t, dashboardPage.Body))
	}
	dashboardBody := mustReadBody(t, dashboardPage.Body)
	if !strings.Contains(dashboardBody, `id="content-area"`) {
		t.Fatalf("dashboard page contract mismatch: missing content area marker")
	}

	dashboardStatsResp := getWithAuth(t, client, requestBaseURL+"/api/dashboard/stats", requestHostOverride)
	if dashboardStatsResp.StatusCode != http.StatusOK {
		t.Fatalf("dashboard stats failed: status=%d body=%s", dashboardStatsResp.StatusCode, mustReadBody(t, dashboardStatsResp.Body))
	}
	var dashboardStats map[string]any
	if err := json.NewDecoder(dashboardStatsResp.Body).Decode(&dashboardStats); err != nil {
		t.Fatalf("decode dashboard stats response: %v", err)
	}
	for _, key := range []string{"services_up", "services_down", "requests_day", "attacks_day", "blocked_attacks_day", "services"} {
		if _, ok := dashboardStats[key]; !ok {
			t.Fatalf("dashboard stats contract mismatch: missing %s", key)
		}
	}
}

func ensureManagementLoginAccess(t *testing.T, client *http.Client, requestBaseURL, requestHostOverride, challengeURI string) {
	t.Helper()

	loginURL := requestBaseURL + "/login"
	resp, err := doE2ERequest(client, http.MethodGet, loginURL, requestHostOverride, "text/html,application/json", nil, false)
	if err != nil {
		t.Fatalf("open login entry: %v", err)
	}

	challengeLocation, challenged := extractChallengeLocation(resp, challengeURI)
	if !challenged {
		if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusBadRequest {
			return
		}
		t.Fatalf("unexpected login entry response: status=%d location=%q", resp.StatusCode, strings.TrimSpace(resp.Header.Get("Location")))
	}

	challengePageURL := absolutizeLocation(requestBaseURL, challengeLocation)
	challengeResp, err := doE2ERequest(client, http.MethodGet, challengePageURL, requestHostOverride, "text/html", nil, false)
	if err != nil {
		t.Fatalf("open challenge page: %v", err)
	}
	if challengeResp.StatusCode != http.StatusOK {
		t.Fatalf("challenge page failed: status=%d body=%s", challengeResp.StatusCode, mustReadBody(t, challengeResp.Body))
	}
	challengeBody := strings.ToLower(mustReadBody(t, challengeResp.Body))
	if !strings.Contains(challengeBody, "verification") && !strings.Contains(challengeBody, "challenge") {
		t.Fatalf("challenge page contract mismatch")
	}

	verifyURL, err := buildVerifyURL(requestBaseURL, challengeLocation, antibotVerifyURI(challengeURI))
	if err != nil {
		t.Fatalf("build challenge verify url: %v", err)
	}
	verifyResp, err := doE2ERequest(client, http.MethodGet, verifyURL, requestHostOverride, "text/html", nil, false)
	if err != nil {
		t.Fatalf("verify challenge: %v", err)
	}
	if verifyResp.StatusCode != http.StatusFound {
		t.Fatalf("challenge verify failed: status=%d body=%s", verifyResp.StatusCode, mustReadBody(t, verifyResp.Body))
	}
	_ = verifyResp.Body.Close()

	postVerifyResp, err := doE2ERequest(client, http.MethodGet, loginURL, requestHostOverride, "text/html,application/json", nil, false)
	if err != nil {
		t.Fatalf("re-open login entry after challenge: %v", err)
	}
	if _, stillChallenged := extractChallengeLocation(postVerifyResp, challengeURI); stillChallenged {
		t.Fatalf("login entry is still challenged after verify")
	}
	if postVerifyResp.StatusCode < http.StatusOK || postVerifyResp.StatusCode >= http.StatusBadRequest {
		t.Fatalf("login entry is still unavailable after verify: status=%d location=%q", postVerifyResp.StatusCode, strings.TrimSpace(postVerifyResp.Header.Get("Location")))
	}
}

func waitForHTTP(client *http.Client, target string, hostOverride string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	lastErr := ""
	for time.Now().Before(deadline) {
		resp, err := doE2ERequest(client, http.MethodGet, target, hostOverride, "text/html,application/json", nil, true)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 500 {
				return nil
			}
			lastErr = fmt.Sprintf("status=%d", resp.StatusCode)
		} else {
			lastErr = err.Error()
		}
		time.Sleep(2 * time.Second)
	}
	if lastErr == "" {
		lastErr = "no response"
	}
	return fmt.Errorf("timeout waiting for %s (%s)", target, lastErr)
}

func postJSON(t *testing.T, client *http.Client, endpoint string, hostOverride string, payload any) *http.Response {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload for %s: %v", endpoint, err)
	}
	req, err := newE2ERequest(http.MethodPost, endpoint, hostOverride, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create request %s: %v", endpoint, err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := doPreparedE2ERequest(client, req, true)
	if err != nil {
		t.Fatalf("request failed %s: %v", endpoint, err)
	}
	return resp
}

func getWithAuth(t *testing.T, client *http.Client, endpoint string, hostOverride string) *http.Response {
	t.Helper()
	req, err := newE2ERequest(http.MethodGet, endpoint, hostOverride, "application/json,text/html", nil)
	if err != nil {
		t.Fatalf("create request %s: %v", endpoint, err)
	}
	resp, err := doPreparedE2ERequest(client, req, true)
	if err != nil {
		t.Fatalf("request failed %s: %v", endpoint, err)
	}
	return resp
}

func newE2ERequest(method, endpoint, hostOverride, accept string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, endpoint, body)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(accept) != "" {
		req.Header.Set("Accept", accept)
	}
	req.Header.Set("User-Agent", defaultE2EBrowserUserAgent)
	if hostOverride != "" {
		req.Host = hostOverride
	}
	return req, nil
}

func doE2ERequest(client *http.Client, method, endpoint, hostOverride, accept string, body io.Reader, follow bool) (*http.Response, error) {
	req, err := newE2ERequest(method, endpoint, hostOverride, accept, body)
	if err != nil {
		return nil, err
	}
	return doPreparedE2ERequest(client, req, follow)
}

func doPreparedE2ERequest(client *http.Client, req *http.Request, follow bool) (*http.Response, error) {
	requestClient := client
	if !follow {
		tmp := *client
		tmp.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
		requestClient = &tmp
	}
	return requestClient.Do(req)
}

func effectivePort(u *url.URL) string {
	if port := strings.TrimSpace(u.Port()); port != "" {
		return port
	}
	if strings.EqualFold(u.Scheme, "https") {
		return "443"
	}
	return "80"
}

func mustReadBody(t *testing.T, body io.ReadCloser) string {
	t.Helper()
	defer func() { _ = body.Close() }()
	raw, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(raw)
}
