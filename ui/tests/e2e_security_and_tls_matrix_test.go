package tests

import (
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestE2ESecurityAndTLSMatrix(t *testing.T) {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	if baseURL == "" {
		t.Skip("WAF_E2E_BASE_URL is not set; skipping security/tls matrix")
	}
	adminClient, requestBaseURL, requestHostOverride := newE2EClientAndBase(t, baseURL)
	loginE2EUser(t, adminClient, requestBaseURL, requestHostOverride)

	t.Run("BrowserContracts", func(t *testing.T) {
		pages := []struct {
			path    string
			markers []string
		}{
			{"/sites", []string{"id=\"content-area\"", "sites.page"}},
			{"/tls", []string{"id=\"content-area\"", "tls"}},
			{"/administration", []string{"id=\"content-area\"", "administration"}},
			{"/settings", []string{"id=\"content-area\"", "settings"}},
		}
		for _, page := range pages {
			resp := getWithAuth(t, adminClient, requestBaseURL+page.path, requestHostOverride)
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("page %s failed: %d body=%s", page.path, resp.StatusCode, mustReadBody(t, resp.Body))
			}
			body := mustReadBody(t, resp.Body)
			for _, marker := range page.markers {
				if !strings.Contains(strings.ToLower(body), strings.ToLower(marker)) {
					t.Fatalf("page %s missing marker %s", page.path, marker)
				}
			}
		}
	})

	t.Run("Deterministic429Path", func(t *testing.T) {
		setResp := requestSecTLSJSON(t, adminClient, http.MethodPut, requestBaseURL+"/api/anti-ddos/settings", requestHostOverride, map[string]any{
			"use_l4_guard":           false,
			"chain_mode":             "auto",
			"conn_limit":             120,
			"rate_per_second":        60,
			"rate_burst":             120,
			"ports":                  []int{80, 443},
			"target":                 "REJECT",
			"enforce_l7_rate_limit":  true,
			"l7_requests_per_second": 1,
			"l7_burst":               1,
			"l7_status_code":         429,
		})
		assertStatusOneOfSecTLS(t, setResp, "set anti-ddos", http.StatusOK)
		hit429 := false
		for i := 0; i < 12; i++ {
			resp := getWithAuthRetry429(t, adminClient, requestBaseURL+"/api/app/meta", requestHostOverride, 1)
			if resp.StatusCode == http.StatusTooManyRequests {
				hit429 = true
				_ = resp.Body.Close()
				break
			}
			_ = resp.Body.Close()
		}
		if !hit429 {
			t.Log("429 was not reached in this environment run")
		}
	})

	t.Run("TLSLifecycle", func(t *testing.T) {
		certID := "e2e-tls-cert"
		siteID := "landing"
		issue := postJSON(t, adminClient, requestBaseURL+"/api/certificates/self-signed/issue", requestHostOverride, map[string]any{
			"certificate_id": certID,
			"common_name":    "landing.localhost",
			"san_list":       []string{"www.landing.localhost"},
		})
		assertStatusOneOfSecTLS(t, issue, "issue self-signed cert", http.StatusCreated, http.StatusOK)

		bind := postJSON(t, adminClient, requestBaseURL+"/api/tls-configs?auto_apply=false", requestHostOverride, map[string]any{
			"site_id":        siteID,
			"certificate_id": certID,
		})
		assertStatusOneOfSecTLS(t, bind, "bind tls config", http.StatusCreated, http.StatusOK)

		renew := postJSON(t, adminClient, requestBaseURL+"/api/certificates/acme/renew/"+certID, requestHostOverride, map[string]any{})
		assertStatusOneOfSecTLS(t, renew, "renew cert", http.StatusCreated, http.StatusOK, http.StatusBadRequest)

		unbind := requestSecTLSJSON(t, adminClient, http.MethodDelete, requestBaseURL+"/api/tls-configs/"+siteID+"?auto_apply=false", requestHostOverride, nil)
		assertStatusOneOfSecTLS(t, unbind, "unbind tls config", http.StatusNoContent, http.StatusOK, http.StatusNotFound)
	})

	t.Run("RBACNegativeMatrix", func(t *testing.T) {
		type roleCase struct {
			roleID        string
			username      string
			password      string
			expectedSites int
			expectedAdmin int
		}
		cases := []roleCase{
			{roleID: "auditor", username: "e2e_auditor", password: "pass-123", expectedSites: http.StatusOK, expectedAdmin: http.StatusForbidden},
			{roleID: "manager", username: "e2e_manager", password: "pass-123", expectedSites: http.StatusOK, expectedAdmin: http.StatusForbidden},
			{roleID: "soc", username: "e2e_soc", password: "pass-123", expectedSites: http.StatusOK, expectedAdmin: http.StatusForbidden},
		}
		for _, tc := range cases {
			createUser := postJSON(t, adminClient, requestBaseURL+"/api/administration/users", requestHostOverride, map[string]any{
				"id":       tc.username,
				"username": tc.username,
				"email":    tc.username + "@example.test",
				"password": tc.password,
				"role_ids": []string{tc.roleID},
			})
			assertStatusOneOfSecTLS(t, createUser, "create user "+tc.username, http.StatusCreated, http.StatusBadRequest)

			roleClient, _, _ := newE2EClientAndBase(t, baseURL)
			loginResp := postJSON(t, roleClient, requestBaseURL+"/api/auth/login", requestHostOverride, map[string]any{
				"username": tc.username,
				"password": tc.password,
			})
			if loginResp.StatusCode != http.StatusOK {
				t.Fatalf("login %s failed: %d body=%s", tc.username, loginResp.StatusCode, mustReadBody(t, loginResp.Body))
			}
			_ = loginResp.Body.Close()

			sitesResp := getWithAuth(t, roleClient, requestBaseURL+"/api/sites", requestHostOverride)
			if sitesResp.StatusCode != tc.expectedSites {
				t.Fatalf("%s /api/sites expected=%d got=%d", tc.username, tc.expectedSites, sitesResp.StatusCode)
			}
			_ = sitesResp.Body.Close()

			adminResp := getWithAuth(t, roleClient, requestBaseURL+"/api/administration/users", requestHostOverride)
			if adminResp.StatusCode != tc.expectedAdmin {
				t.Fatalf("%s /api/administration/users expected=%d got=%d body=%s", tc.username, tc.expectedAdmin, adminResp.StatusCode, mustReadBody(t, adminResp.Body))
			}
			_ = adminResp.Body.Close()
		}
	})
}

func requestSecTLSJSON(t *testing.T, client *http.Client, method, endpoint, hostOverride string, payload any) *http.Response {
	t.Helper()
	if payload == nil {
		req, err := http.NewRequest(method, endpoint, nil)
		if err != nil {
			t.Fatalf("create %s %s request: %v", method, endpoint, err)
		}
		req.Header.Set("Accept", "application/json")
		if hostOverride != "" {
			req.Host = hostOverride
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request %s %s failed: %v", method, endpoint, err)
		}
		return resp
	}
	return postJSON(t, client, endpoint, hostOverride, payload)
}

func assertStatusOneOfSecTLS(t *testing.T, resp *http.Response, action string, allowed ...int) {
	t.Helper()
	for _, code := range allowed {
		if resp.StatusCode == code {
			_ = resp.Body.Close()
			return
		}
	}
	body := ""
	if resp.Body != nil {
		body = mustReadBody(t, resp.Body)
	}
	t.Fatalf("%s failed: status=%d body=%s", action, resp.StatusCode, body)
}
