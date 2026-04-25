package tests

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

func TestE2ESmoke_LoginHealthcheckDashboard(t *testing.T) {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	if baseURL == "" {
		t.Skip("WAF_E2E_BASE_URL is not set; skipping e2e smoke test")
	}
	requestBaseURL := baseURL
	requestHostOverride := ""
	originalBaseParsed, err := url.Parse(baseURL)
	if err != nil {
		t.Fatalf("parse base url: %v", err)
	}
	requestBaseParsed := originalBaseParsed
	if strings.EqualFold(originalBaseParsed.Hostname(), "localhost") {
		requestHostOverride = originalBaseParsed.Hostname()
		requestBaseParsed = &url.URL{}
		*requestBaseParsed = *originalBaseParsed
		requestBaseParsed.Host = net.JoinHostPort("127.0.0.1", effectivePort(originalBaseParsed))
		requestBaseURL = strings.TrimRight(requestBaseParsed.String(), "/")
	}
	username := strings.TrimSpace(os.Getenv("WAF_E2E_USERNAME"))
	if username == "" {
		username = "admin"
	}
	password := os.Getenv("WAF_E2E_PASSWORD")
	if password == "" {
		password = "admin"
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookie jar init failed: %v", err)
	}
	client := &http.Client{
		Timeout: 15 * time.Second,
		Jar:     jar,
	}
	if strings.HasPrefix(strings.ToLower(requestBaseURL), "https://") {
		transport := &http.Transport{
			Proxy:                 nil,
			ForceAttemptHTTP2:     false,
			TLSHandshakeTimeout:   15 * time.Second,
			ResponseHeaderTimeout: 15 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
				ServerName:         "localhost",
			},
		}
		dialer := &net.Dialer{Timeout: 15 * time.Second}
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return dialer.DialContext(ctx, network, addr)
			}
			if strings.EqualFold(host, "localhost") {
				host = "127.0.0.1"
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(host, port))
		}
		client.Transport = transport
	}

	if err := waitForHTTP(client, requestBaseURL+"/login", requestHostOverride, 90*time.Second); err != nil {
		t.Fatalf("ui is not ready: %v", err)
	}

	loginPayload := map[string]any{
		"username": username,
		"password": password,
	}
	loginResp := postJSON(t, client, requestBaseURL+"/api/auth/login", requestHostOverride, loginPayload)
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
		jar.SetCookies(requestBaseParsed, loginResp.Cookies())
	}
	cookies := jar.Cookies(originalBaseParsed)
	if len(cookies) == 0 {
		cookies = jar.Cookies(requestBaseParsed)
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

func waitForHTTP(client *http.Client, target string, hostOverride string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	lastErr := ""
	for time.Now().Before(deadline) {
		req, err := http.NewRequest(http.MethodGet, target, nil)
		if err != nil {
			return fmt.Errorf("create readiness request: %w", err)
		}
		if hostOverride != "" {
			req.Host = hostOverride
		}
		resp, err := client.Do(req)
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
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create request %s: %v", endpoint, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if hostOverride != "" {
		req.Host = hostOverride
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed %s: %v", endpoint, err)
	}
	return resp
}

func getWithAuth(t *testing.T, client *http.Client, endpoint string, hostOverride string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		t.Fatalf("create request %s: %v", endpoint, err)
	}
	req.Header.Set("Accept", "application/json,text/html")
	if hostOverride != "" {
		req.Host = hostOverride
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed %s: %v", endpoint, err)
	}
	return resp
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
