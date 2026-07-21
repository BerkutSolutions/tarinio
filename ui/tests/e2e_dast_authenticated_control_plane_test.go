package tests

import (
	"net/http"
	"os"
	"strings"
	"testing"
)

// TestE2EDASTAuthenticatedControlPlane proves the safe DAST contract for the
// management API without letting an active scanner mutate the configuration.
func TestE2EDASTAuthenticatedControlPlane(t *testing.T) {
	panelURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	if panelURL == "" {
		t.Skip("WAF_E2E_BASE_URL is required")
	}

	anonymous, baseURL, hostOverride := newE2EClientAndBase(t, panelURL)
	for _, endpoint := range []string{"/api/administration/users", "/api/sites", "/api/requests", "/api/settings/management-hosts"} {
		t.Run("anonymous_"+strings.Trim(strings.ReplaceAll(endpoint, "/", "_"), "_"), func(t *testing.T) {
			response := requestE2EJSON(t, anonymous, http.MethodGet, baseURL+endpoint, hostOverride, nil)
			defer response.Body.Close()
			if response.StatusCode != http.StatusUnauthorized && response.StatusCode != http.StatusForbidden {
				t.Fatalf("anonymous %s: status=%d, want 401/403", endpoint, response.StatusCode)
			}
		})
	}

	loginE2EUser(t, anonymous, baseURL, hostOverride)
	for _, endpoint := range []string{"/api/auth/me", "/api/sites", "/api/requests?limit=1"} {
		response := getWithAuth(t, anonymous, baseURL+endpoint, hostOverride)
		if response.StatusCode != http.StatusOK {
			t.Fatalf("authenticated safe endpoint %s: status=%d body=%s", endpoint, response.StatusCode, mustReadBody(t, response.Body))
		}
		_ = response.Body.Close()
	}
	t.Log("authenticated DAST allowlist passed without mutating configuration")

	logout := postJSON(t, anonymous, baseURL+"/api/auth/logout", hostOverride, map[string]any{})
	if logout.StatusCode != http.StatusOK && logout.StatusCode != http.StatusNoContent {
		t.Fatalf("logout: status=%d body=%s", logout.StatusCode, mustReadBody(t, logout.Body))
	}
	_ = logout.Body.Close()
	revoked := getWithAuth(t, anonymous, baseURL+"/api/requests?limit=1", hostOverride)
	defer revoked.Body.Close()
	if revoked.StatusCode != http.StatusUnauthorized && revoked.StatusCode != http.StatusForbidden {
		t.Fatalf("revoked session accessed requests: status=%d", revoked.StatusCode)
	}
	t.Logf("revoked session rejected with HTTP %d", revoked.StatusCode)
}
