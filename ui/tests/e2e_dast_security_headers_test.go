package tests

import (
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestE2EDASTSecurityHeadersAndSessionCookie(t *testing.T) {
	panelURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	if panelURL == "" {
		t.Skip("WAF_E2E_BASE_URL is required")
	}
	client, baseURL, hostOverride := newE2EClientAndBase(t, panelURL)
	login := postJSON(t, client, baseURL+"/api/auth/login", hostOverride, map[string]any{
		"username": os.Getenv("WAF_E2E_USERNAME"), "password": os.Getenv("WAF_E2E_PASSWORD"),
	})
	if login.StatusCode != http.StatusOK {
		t.Fatalf("login for cookie DAST check: status=%d body=%s", login.StatusCode, mustReadBody(t, login.Body))
	}
	cookies := login.Cookies()
	_ = login.Body.Close()
	if len(cookies) == 0 {
		t.Fatal("login did not issue a session cookie")
	}
	for _, cookie := range cookies {
		if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode {
			t.Fatalf("session cookie lacks HttpOnly/SameSite=Strict: %+v", cookie)
		}
	}

	response := getWithAuth(t, client, baseURL+"/login", hostOverride)
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("authenticated login page: status=%d", response.StatusCode)
	}
	if value := response.Header.Get("X-Content-Type-Options"); !strings.EqualFold(value, "nosniff") {
		t.Fatalf("X-Content-Type-Options=%q, want nosniff", value)
	}
	if value := response.Header.Get("X-Frame-Options"); value != "" && !strings.EqualFold(value, "SAMEORIGIN") && !strings.EqualFold(value, "DENY") {
		t.Fatalf("unexpected X-Frame-Options=%q", value)
	}
	if strings.Contains(response.Header.Get("Access-Control-Allow-Origin"), "*") && response.Header.Get("Access-Control-Allow-Credentials") == "true" {
		t.Fatal("credentialed management response permits wildcard CORS")
	}
}
