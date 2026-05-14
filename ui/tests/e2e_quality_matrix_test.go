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

func TestE2EQualityMatrix_MainPagesAndAPIs(t *testing.T) {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	if baseURL == "" {
		t.Skip("WAF_E2E_BASE_URL is not set; skipping e2e quality matrix")
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
	client := &http.Client{Timeout: 20 * time.Second, Jar: jar}
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

	if err := waitForHTTP(client, requestBaseURL+"/login", requestHostOverride, 90*time.Second); err != nil {
		t.Fatalf("ui is not ready: %v", err)
	}
	loginResp := postJSON(t, client, requestBaseURL+"/api/auth/login", requestHostOverride, map[string]any{
		"username": username,
		"password": password,
	})
	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("login failed: status=%d body=%s", loginResp.StatusCode, mustReadBody(t, loginResp.Body))
	}
	_ = mustReadBody(t, loginResp.Body)
	if requestHostOverride != "" {
		jar.SetCookies(requestBaseParsed, loginResp.Cookies())
	}

	pageCases := []struct {
		name    string
		path    string
		markers []string
	}{
		{name: "dashboard", path: "/dashboard", markers: []string{`id="content-area"`}},
		{name: "healthcheck", path: "/healthcheck", markers: []string{`id="healthcheck-steps"`}},
		{name: "sites", path: "/sites", markers: []string{`id="content-area"`}},
		{name: "requests", path: "/requests", markers: []string{`id="content-area"`}},
		{name: "settings", path: "/settings", markers: []string{`id="content-area"`}},
		{name: "administration", path: "/administration", markers: []string{`id="content-area"`}},
		{name: "bans", path: "/bans", markers: []string{`id="content-area"`}},
	}
	for _, tc := range pageCases {
		t.Run("page_"+tc.name, func(t *testing.T) {
			resp := getWithAuth(t, client, requestBaseURL+tc.path, requestHostOverride)
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

	apiCases := []struct {
		name string
		path string
	}{
		{name: "app_meta", path: "/api/app/meta"},
		{name: "app_compat", path: "/api/app/compat"},
		{name: "auth_me", path: "/api/auth/me"},
		{name: "dashboard_stats", path: "/api/dashboard/stats"},
		{name: "dashboard_containers", path: "/api/dashboard/containers/overview"},
		{name: "sites", path: "/api/sites"},
		{name: "upstreams", path: "/api/upstreams"},
		{name: "tls_configs", path: "/api/tls-configs"},
		{name: "revisions", path: "/api/revisions"},
		{name: "settings_runtime", path: "/api/settings/runtime"},
		{name: "administration_roles", path: "/api/administration/roles"},
		{name: "administration_users", path: "/api/administration/users"},
		{name: "administration_scripts", path: "/api/administration/scripts"},
	}
	for _, tc := range apiCases {
		t.Run("api_"+tc.name, func(t *testing.T) {
			resp := getWithAuth(t, client, requestBaseURL+tc.path, requestHostOverride)
			if resp.StatusCode == http.StatusTooManyRequests {
				t.Fatalf("api %s returned 429", tc.path)
			}
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("api %s failed: status=%d body=%s", tc.path, resp.StatusCode, mustReadBody(t, resp.Body))
			}
			var payload any
			if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
				t.Fatalf("api %s returned invalid json: %v", tc.path, err)
			}
		})
	}
}
