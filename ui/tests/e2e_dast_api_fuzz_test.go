package tests

import (
	"net/http"
	"os"
	"strings"
	"testing"
)

// TestE2EDASTAPIInputFuzz verifies that malformed and oversized management
// requests are rejected predictably without reaching a server-error state.
func TestE2EDASTAPIInputFuzz(t *testing.T) {
	panelURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	if panelURL == "" {
		t.Skip("WAF_E2E_BASE_URL is required")
	}
	client, baseURL, hostOverride := newE2EClientAndBase(t, panelURL)
	for _, tc := range []struct {
		name string
		body string
	}{
		{name: "malformed_json", body: `{"username":`},
		{name: "unknown_fields", body: `{"username":"invalid","password":"invalid","unexpected":{"nested":true}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response, err := doE2ERequest(client, http.MethodPost, baseURL+"/api/auth/login", hostOverride, "application/json", strings.NewReader(tc.body), false)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			if response.StatusCode < http.StatusBadRequest || response.StatusCode >= http.StatusInternalServerError {
				t.Fatalf("%s: status=%d, want controlled 4xx", tc.name, response.StatusCode)
			}
		})
	}

	request, err := newE2ERequest(http.MethodGet, baseURL+"/api/auth/me", hostOverride, "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("X-DAST-Fuzz", strings.Repeat("A", 16*1024))
	response, err := doPreparedE2ERequest(client, request, false)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode >= http.StatusInternalServerError {
		t.Fatalf("oversized header: status=%d, want non-5xx", response.StatusCode)
	}
	t.Logf("API input fuzzing produced controlled status %d for oversized header", response.StatusCode)
}
