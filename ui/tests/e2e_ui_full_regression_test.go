package tests

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

func TestE2EUIAssetsAndRouting(t *testing.T) {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	if baseURL == "" {
		t.Skip("WAF_E2E_BASE_URL is not set; skipping UI regression")
	}

	client, requestBaseURL, requestHostOverride := newE2EClientAndBase(t, baseURL)
	loginE2EUser(t, client, requestBaseURL, requestHostOverride)

	pageCases := []struct {
		name    string
		path    string
		markers []string
	}{
		{name: "dashboard", path: "/dashboard", markers: []string{`id="content-area"`}},
		{name: "healthcheck", path: "/healthcheck", markers: []string{`id="healthcheck-steps"`}},
		{name: "services", path: "/services", markers: []string{`id="content-area"`}},
		{name: "requests", path: "/requests", markers: []string{`id="content-area"`}},
		{name: "settings", path: "/settings", markers: []string{`id="content-area"`}},
		{name: "administration", path: "/administration", markers: []string{`id="content-area"`}},
		{name: "bans", path: "/bans", markers: []string{`id="content-area"`}},
	}
	for _, tc := range pageCases {
		t.Run("page_"+tc.name, func(t *testing.T) {
			resp := getWithAuthRetry429(t, client, requestBaseURL+tc.path, requestHostOverride, 3)
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("page %s failed: status=%d body=%s", tc.path, resp.StatusCode, mustReadBody(t, resp.Body))
			}
			body := mustReadBody(t, resp.Body)
			for _, marker := range tc.markers {
				if !strings.Contains(body, marker) {
					t.Fatalf("page %s missing marker %s", tc.path, marker)
				}
			}
		})
	}
}

func TestE2EDashboardSessionPingBodylessPOST(t *testing.T) {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	if baseURL == "" {
		t.Skip("WAF_E2E_BASE_URL is not set; skipping dashboard ping regression")
	}

	client, requestBaseURL, requestHostOverride := newE2EClientAndBase(t, baseURL)
	loginE2EUser(t, client, requestBaseURL, requestHostOverride)

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
		t.Fatalf("session ping request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("session ping failed: status=%d body=%s", resp.StatusCode, mustReadBody(t, resp.Body))
	}

	var payload struct {
		OK bool `json:"ok"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode ping payload: %v", err)
	}
	if !payload.OK {
		t.Fatalf("expected ok=true from ping, got %#v", payload)
	}
}

func TestE2EServicesModuleNoChallengeAfterLogin(t *testing.T) {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	if baseURL == "" {
		t.Skip("WAF_E2E_BASE_URL is not set; skipping services module regression")
	}

	client, requestBaseURL, requestHostOverride := newE2EClientAndBase(t, baseURL)
	loginE2EUser(t, client, requestBaseURL, requestHostOverride)

	resp := getWithAuthRetry429(t, client, requestBaseURL+"/static/js/pages/sites.js?v=20260628-16", requestHostOverride, 3)
	defer resp.Body.Close()
	body := mustReadBody(t, resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("services module failed: status=%d body=%s", resp.StatusCode, body)
	}
	if strings.Contains(body, "<html") || strings.Contains(strings.ToLower(body), "challenge/stage1/verify") {
		t.Fatalf("services module returned challenge/html instead of js: status=%d body=%s", resp.StatusCode, body)
	}
	if !strings.Contains(body, `export { renderSites } from "./sites.stable-page.js";`) {
		t.Fatalf("services module contract mismatch: body=%s", body)
	}
}

func newE2EClientAndBase(t *testing.T, baseURL string) (*http.Client, string, string) {
	t.Helper()
	requestBaseURL := baseURL
	requestHostOverride := ""
	originalBaseParsed, err := url.Parse(baseURL)
	if err != nil {
		t.Fatalf("parse base url: %v", err)
	}
	if strings.EqualFold(originalBaseParsed.Hostname(), "localhost") {
		requestHostOverride = originalBaseParsed.Host
		requestBaseParsed := &url.URL{}
		*requestBaseParsed = *originalBaseParsed
		requestBaseParsed.Host = net.JoinHostPort("127.0.0.1", effectivePort(originalBaseParsed))
		requestBaseURL = strings.TrimRight(requestBaseParsed.String(), "/")
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookie jar init failed: %v", err)
	}
	client := &http.Client{
		Timeout: 20 * time.Second,
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
			host, port, splitErr := net.SplitHostPort(addr)
			if splitErr != nil {
				return dialer.DialContext(ctx, network, addr)
			}
			if strings.EqualFold(host, "localhost") {
				host = "127.0.0.1"
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(host, port))
		}
		client.Transport = transport
	}
	return client, requestBaseURL, requestHostOverride
}

func loginE2EUser(t *testing.T, client *http.Client, requestBaseURL, requestHostOverride string) {
	t.Helper()
	if err := waitForHTTP(client, requestBaseURL+"/login", requestHostOverride, 90*time.Second); err != nil {
		t.Fatalf("ui is not ready: %v", err)
	}
	username := strings.TrimSpace(os.Getenv("WAF_E2E_USERNAME"))
	if username == "" {
		username = "admin"
	}
	password := os.Getenv("WAF_E2E_PASSWORD")
	if password == "" || password == "***" {
		password = "admin"
	}
	loginResp := postJSON(t, client, requestBaseURL+"/api/auth/login", requestHostOverride, map[string]any{
		"username": username,
		"password": password,
	})
	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("login failed: status=%d body=%s", loginResp.StatusCode, mustReadBody(t, loginResp.Body))
	}
	_ = mustReadBody(t, loginResp.Body)
}

func getWithAuthRetry429(t *testing.T, client *http.Client, endpoint, hostOverride string, retries int) *http.Response {
	t.Helper()
	attempts := retries
	if attempts < 1 {
		attempts = 1
	}
	backoff := 200 * time.Millisecond
	for i := 0; i < attempts; i++ {
		req, err := http.NewRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			t.Fatalf("create request %s: %v", endpoint, err)
		}
		req.Header.Set("Accept", "application/json,text/html,*/*")
		if hostOverride != "" {
			req.Host = hostOverride
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed %s: %v", endpoint, err)
		}
		if resp.StatusCode != http.StatusTooManyRequests {
			return resp
		}
		_ = resp.Body.Close()
		if i < attempts-1 {
			time.Sleep(backoff)
			backoff *= 2
		}
	}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		t.Fatalf("create request %s: %v", endpoint, err)
	}
	req.Header.Set("Accept", "application/json,text/html,*/*")
	if hostOverride != "" {
		req.Host = hostOverride
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed %s: %v", endpoint, err)
	}
	return resp
}
