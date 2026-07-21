package tests

import (
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestE2ESecurityInvariant_PasswordChangeRevokesOtherSessionAndRecordsLogin(t *testing.T) {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	if baseURL == "" {
		t.Skip("WAF_E2E_BASE_URL is not configured")
	}
	admin, requestBaseURL, host := newE2EClientAndBase(t, baseURL)
	loginE2EUser(t, admin, requestBaseURL, host)
	username := e2eUniqueID(t, "e2e-session")
	password, rotated := "Initial-password-123!", "Rotated-password-456!"
	created := postJSON(t, admin, requestBaseURL+"/api/administration/users", host, map[string]any{
		"id": username, "username": username, "email": username + "@e2e.test", "password": password, "role_ids": []string{"auditor"},
	})
	if created.StatusCode != http.StatusCreated {
		t.Fatalf("create isolated user: status=%d body=%s", created.StatusCode, mustReadBody(t, created.Body))
	}
	_ = created.Body.Close()
	first, _, _ := newE2EClientAndBase(t, baseURL)
	second, _, _ := newE2EClientAndBase(t, baseURL)
	loginE2EUserAs(t, first, requestBaseURL, host, username, password)
	loginE2EUserAs(t, second, requestBaseURL, host, username, password)
	assertE2EUserLastLogin(t, admin, requestBaseURL, host, username)
	changed := postJSON(t, first, requestBaseURL+"/api/auth/change-password", host, map[string]any{"current_password": password, "password": rotated})
	if changed.StatusCode != http.StatusOK {
		t.Fatalf("change password: status=%d body=%s", changed.StatusCode, mustReadBody(t, changed.Body))
	}
	_ = changed.Body.Close()
	revoked := getWithAuth(t, second, requestBaseURL+"/api/auth/me", host)
	if revoked.StatusCode != http.StatusUnauthorized && revoked.StatusCode != http.StatusForbidden {
		t.Fatalf("second session must be revoked: status=%d body=%s", revoked.StatusCode, mustReadBody(t, revoked.Body))
	}
	_ = revoked.Body.Close()
	stillCurrent := getWithAuth(t, first, requestBaseURL+"/api/auth/me", host)
	if stillCurrent.StatusCode != http.StatusOK {
		t.Fatalf("password-changing session must remain valid: status=%d body=%s", stillCurrent.StatusCode, mustReadBody(t, stillCurrent.Body))
	}
	_ = stillCurrent.Body.Close()
}

func loginE2EUserAs(t *testing.T, client *http.Client, baseURL, host, username, password string) {
	t.Helper()
	response := postJSON(t, client, baseURL+"/api/auth/login", host, map[string]any{"username": username, "password": password})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("login isolated user: status=%d body=%s", response.StatusCode, mustReadBody(t, response.Body))
	}
	_ = response.Body.Close()
}

func assertE2EUserLastLogin(t *testing.T, admin *http.Client, baseURL, host, username string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		response := getWithAuth(t, admin, baseURL+"/api/administration/users/"+username, host)
		body := mustReadBody(t, response.Body)
		if response.StatusCode == http.StatusOK && strings.Contains(body, `"last_login_at":"`) {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("successful login must update last_login_at for %s", username)
}
