package tests

import (
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
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestE2EUIFullRegression(t *testing.T) {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	if baseURL == "" {
		t.Skip("WAF_E2E_BASE_URL is not set; skipping full UI regression")
	}

	client, requestBaseURL, requestHostOverride := newE2EClientAndBase(t, baseURL)
	loginE2EUser(t, client, requestBaseURL, requestHostOverride)

	t.Run("MainPages", func(t *testing.T) {
		cases := []struct {
			path    string
			markers []string
		}{
			{path: "/dashboard", markers: []string{`id="content-area"`}},
			{path: "/sites", markers: []string{`id="content-area"`}},
			{path: "/requests", markers: []string{`id="content-area"`}},
			{path: "/events", markers: []string{`id="content-area"`}},
			{path: "/bans", markers: []string{`id="content-area"`}},
			{path: "/settings", markers: []string{`id="content-area"`}},
			{path: "/administration", markers: []string{`id="content-area"`}},
			{path: "/healthcheck", markers: []string{`id="healthcheck-steps"`}},
		}
		for _, tc := range cases {
			tc := tc
			t.Run(tc.path, func(t *testing.T) {
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
	})

	t.Run("CoreAPIs", func(t *testing.T) {
		cases := []string{
			"/api/auth/me",
			"/api/app/meta",
			"/api/app/compat",
			"/api/dashboard/stats",
			"/api/dashboard/containers/overview",
			"/api/dashboard/containers/issues",
			"/api/sites",
			"/api/upstreams",
			"/api/tls-configs",
			"/api/revisions",
			"/api/reports/revisions",
			"/api/settings/runtime",
			"/api/administration/users",
			"/api/administration/roles",
			"/api/administration/scripts",
			"/api/anti-ddos/settings",
			"/api/owasp-crs/status",
			"/api/certificates",
		}
		for _, path := range cases {
			path := path
			t.Run(path, func(t *testing.T) {
				resp := getWithAuth(t, client, requestBaseURL+path, requestHostOverride)
				if resp.StatusCode == http.StatusTooManyRequests {
					t.Fatalf("api %s returned 429", path)
				}
				if resp.StatusCode != http.StatusOK {
					t.Fatalf("api %s failed: status=%d body=%s", path, resp.StatusCode, mustReadBody(t, resp.Body))
				}
				var payload any
				if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
					t.Fatalf("api %s returned invalid json: %v", path, err)
				}
			})
		}
	})

	t.Run("AppCompatFix", func(t *testing.T) {
		resp := getWithAuth(t, client, requestBaseURL+"/api/app/compat", requestHostOverride)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("compat api failed: status=%d body=%s", resp.StatusCode, mustReadBody(t, resp.Body))
		}
		var data struct {
			Items []map[string]any `json:"items"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			t.Fatalf("decode compat response: %v", err)
		}
		if len(data.Items) == 0 {
			t.Fatalf("compat api returned empty items")
		}
		moduleID := strings.TrimSpace(fmt.Sprint(data.Items[0]["module_id"]))
		if moduleID == "" {
			t.Fatalf("compat api returned empty module_id")
		}
		fixResp := postJSON(t, client, requestBaseURL+"/api/app/compat/fix", requestHostOverride, map[string]any{"module_id": moduleID})
		if fixResp.StatusCode != http.StatusOK {
			t.Fatalf("compat fix failed: status=%d body=%s", fixResp.StatusCode, mustReadBody(t, fixResp.Body))
		}
		var fixData map[string]any
		if err := json.NewDecoder(fixResp.Body).Decode(&fixData); err != nil {
			t.Fatalf("decode compat fix response: %v", err)
		}
		if ok, _ := fixData["ok"].(bool); !ok {
			t.Fatalf("expected compat fix ok=true, got %#v", fixData)
		}
	})

	t.Run("LoadAllPageModules", func(t *testing.T) {
		mods, err := filepath.Glob(filepath.Join("..", "app", "static", "js", "pages", "*.js"))
		if err != nil {
			t.Fatalf("glob page modules: %v", err)
		}
		if len(mods) == 0 {
			t.Fatalf("no page modules found")
		}
		for _, path := range mods {
			mod := filepath.Base(path)
			t.Run(mod, func(t *testing.T) {
				url := requestBaseURL + "/static/js/pages/" + mod
				resp := getWithAuthRetry429(t, client, url, requestHostOverride, 5)
				if resp.StatusCode == http.StatusTooManyRequests {
					t.Fatalf("module %s returned 429", mod)
				}
				if resp.StatusCode != http.StatusOK {
					t.Fatalf("module %s failed: status=%d body=%s", mod, resp.StatusCode, mustReadBody(t, resp.Body))
				}
				bodyBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Fatalf("read module %s: %v", mod, err)
				}
				body := strings.TrimSpace(string(bodyBytes))
				if body == "" {
					t.Fatalf("module %s is empty", mod)
				}
				if strings.Contains(strings.ToLower(body), "<html") {
					t.Fatalf("module %s returned html instead of js", mod)
				}
			})
		}
	})
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
		requestHostOverride = originalBaseParsed.Hostname()
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
	if password == "" {
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
