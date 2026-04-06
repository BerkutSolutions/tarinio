package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

	if err := waitForHTTP(client, baseURL+"/login", 90*time.Second); err != nil {
		t.Fatalf("ui is not ready: %v", err)
	}

	loginPayload := map[string]any{
		"username": username,
		"password": password,
	}
	loginResp := postJSON(t, client, baseURL+"/api/auth/login", loginPayload)
	if loginResp.StatusCode != http.StatusOK {
		t.Fatalf("login failed: status=%d body=%s", loginResp.StatusCode, mustReadBody(t, loginResp.Body))
	}
	var loginData map[string]any
	if err := json.NewDecoder(loginResp.Body).Decode(&loginData); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if requires2FA, _ := loginData["requires_2fa"].(bool); requires2FA {
		t.Fatalf("smoke flow requires direct login without 2fa; got requires_2fa=true")
	}

	baseParsed, err := url.Parse(baseURL)
	if err != nil {
		t.Fatalf("parse base url: %v", err)
	}
	cookies := jar.Cookies(baseParsed)
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

	healthcheckPage := getWithAuth(t, client, baseURL+"/healthcheck")
	if healthcheckPage.StatusCode != http.StatusOK {
		t.Fatalf("healthcheck page failed: status=%d", healthcheckPage.StatusCode)
	}
	healthcheckBody := mustReadBody(t, healthcheckPage.Body)
	if !strings.Contains(healthcheckBody, "Проверка системы") {
		t.Fatalf("healthcheck page contract mismatch: missing title marker")
	}

	compatResp := getWithAuth(t, client, baseURL+"/api/app/compat")
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

	fixResp := postJSON(t, client, baseURL+"/api/app/compat/fix", map[string]any{"module_id": firstModuleID})
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

	dashboardPage := getWithAuth(t, client, baseURL+"/dashboard")
	if dashboardPage.StatusCode != http.StatusOK {
		t.Fatalf("dashboard page failed: status=%d body=%s", dashboardPage.StatusCode, mustReadBody(t, dashboardPage.Body))
	}
	dashboardBody := mustReadBody(t, dashboardPage.Body)
	if !strings.Contains(dashboardBody, `id="content-area"`) {
		t.Fatalf("dashboard page contract mismatch: missing content area marker")
	}
}

func waitForHTTP(client *http.Client, target string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := client.Get(target)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 500 {
				return nil
			}
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("timeout waiting for %s", target)
}

func postJSON(t *testing.T, client *http.Client, endpoint string, payload any) *http.Response {
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
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed %s: %v", endpoint, err)
	}
	return resp
}

func getWithAuth(t *testing.T, client *http.Client, endpoint string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		t.Fatalf("create request %s: %v", endpoint, err)
	}
	req.Header.Set("Accept", "application/json,text/html")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed %s: %v", endpoint, err)
	}
	return resp
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
