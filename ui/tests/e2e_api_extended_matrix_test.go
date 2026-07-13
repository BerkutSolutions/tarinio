package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestE2EAPIExtendedMatrix(t *testing.T) {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	if baseURL == "" {
		t.Skip("WAF_E2E_BASE_URL is not set; skipping extended e2e matrix")
	}
	client, requestBaseURL, requestHostOverride := newE2EClientAndBase(t, baseURL)
	loginE2EUser(t, client, requestBaseURL, requestHostOverride)

	t.Run("AdditionalReadEndpoints", func(t *testing.T) {
		paths := []string{
			"/api/audit",
			"/api/waf-policies",
			"/api/access-policies",
			"/api/rate-limit-policies",
			"/api/easy-site-profiles",
			"/api/easy-site-profiles/catalog/countries",
			"/api/anti-ddos/rule-suggestions",
			"/api/tls/auto-renew",
			"/api/administration/zero-trust/health",
		}
		for _, path := range paths {
			resp := getWithAuthRetry429(t, client, requestBaseURL+path, requestHostOverride, 3)
			if resp.StatusCode != http.StatusOK {
				t.Fatalf("%s failed: status=%d body=%s", path, resp.StatusCode, mustReadBody(t, resp.Body))
			}
			_ = resp.Body.Close()
		}
	})

	t.Run("AuthExtendedEndpoints", func(t *testing.T) {
		meResp := getWithAuth(t, client, requestBaseURL+"/api/auth/me", requestHostOverride)
		if meResp.StatusCode != http.StatusOK {
			t.Fatalf("/api/auth/me failed: status=%d body=%s", meResp.StatusCode, mustReadBody(t, meResp.Body))
		}
		_ = meResp.Body.Close()

		statusResp := getWithAuth(t, client, requestBaseURL+"/api/auth/2fa/status", requestHostOverride)
		if statusResp.StatusCode != http.StatusOK {
			t.Fatalf("/api/auth/2fa/status failed: status=%d body=%s", statusResp.StatusCode, mustReadBody(t, statusResp.Body))
		}
		_ = statusResp.Body.Close()

		passkeysResp := getWithAuth(t, client, requestBaseURL+"/api/auth/passkeys", requestHostOverride)
		if passkeysResp.StatusCode != http.StatusOK {
			t.Fatalf("/api/auth/passkeys failed: status=%d body=%s", passkeysResp.StatusCode, mustReadBody(t, passkeysResp.Body))
		}
		_ = passkeysResp.Body.Close()
	})

	t.Run("RevisionsCompileApplyFlow", func(t *testing.T) {
		compileResp := postJSON(t, client, requestBaseURL+"/api/revisions/compile", requestHostOverride, map[string]any{})
		if compileResp.StatusCode != http.StatusCreated {
			t.Fatalf("compile failed: status=%d body=%s", compileResp.StatusCode, mustReadBody(t, compileResp.Body))
		}
		var payload struct {
			Revision struct {
				ID string `json:"id"`
			} `json:"revision"`
		}
		if err := json.NewDecoder(compileResp.Body).Decode(&payload); err != nil {
			t.Fatalf("decode compile response: %v", err)
		}
		if strings.TrimSpace(payload.Revision.ID) == "" {
			t.Fatalf("compile response has empty revision id")
		}
		applyResp := postJSON(t, client, fmt.Sprintf("%s/api/revisions/%s/apply", requestBaseURL, payload.Revision.ID), requestHostOverride, map[string]any{})
		if applyResp.StatusCode != http.StatusCreated {
			t.Fatalf("apply failed: status=%d body=%s", applyResp.StatusCode, mustReadBody(t, applyResp.Body))
		}
		_ = applyResp.Body.Close()
	})

	t.Run("PoliciesCRUD", func(t *testing.T) {
		const siteID = "e2e-policy-site"
		createSite := postJSON(t, client, requestBaseURL+"/api/sites?auto_apply=false", requestHostOverride, map[string]any{
			"id": siteID, "primary_host": "e2e-policy.test", "enabled": true, "listen_http": true, "listen_https": false,
		})
		assertE2EStatus(t, createSite, "create policy site", http.StatusCreated, http.StatusOK)
		t.Cleanup(func() {
			resp := requestE2EJSON(t, client, http.MethodDelete, requestBaseURL+"/api/sites/"+siteID+"?auto_apply=false", requestHostOverride, nil)
			_ = resp.Body.Close()
		})
		wafID := "e2e-waf-policy"
		wafCreate := postJSON(t, client, requestBaseURL+"/api/waf-policies?auto_apply=false", requestHostOverride, map[string]any{
			"id":          wafID,
			"site_id":     siteID,
			"enabled":     true,
			"mode":        "detection",
			"crs_enabled": true,
		})
		assertE2EStatus(t, wafCreate, "/api/waf-policies create", http.StatusCreated, http.StatusOK)
		wafDelete := requestE2EJSON(t, client, http.MethodDelete, requestBaseURL+"/api/waf-policies/"+wafID+"?auto_apply=false", requestHostOverride, nil)
		assertE2EStatus(t, wafDelete, "/api/waf-policies delete", http.StatusNoContent, http.StatusOK, http.StatusNotFound)

		accessID := "e2e-access-policy"
		accessCreate := postJSON(t, client, requestBaseURL+"/api/access-policies?auto_apply=false", requestHostOverride, map[string]any{
			"id":        accessID,
			"site_id":   siteID,
			"enabled":   true,
			"allowlist": []string{"127.0.0.1"},
		})
		assertE2EStatus(t, accessCreate, "/api/access-policies create", http.StatusCreated, http.StatusOK)
		accessDelete := requestE2EJSON(t, client, http.MethodDelete, requestBaseURL+"/api/access-policies/"+accessID+"?auto_apply=false", requestHostOverride, nil)
		assertE2EStatus(t, accessDelete, "/api/access-policies delete", http.StatusNoContent, http.StatusOK, http.StatusNotFound)

		rateID := "e2e-rate-policy"
		rateCreate := postJSON(t, client, requestBaseURL+"/api/rate-limit-policies?auto_apply=false", requestHostOverride, map[string]any{
			"id":      rateID,
			"site_id": siteID,
			"enabled": true,
			"limits": map[string]any{
				"requests_per_second": 5,
				"burst":               10,
			},
		})
		assertE2EStatus(t, rateCreate, "/api/rate-limit-policies create", http.StatusCreated, http.StatusOK)
		rateDelete := requestE2EJSON(t, client, http.MethodDelete, requestBaseURL+"/api/rate-limit-policies/"+rateID+"?auto_apply=false", requestHostOverride, nil)
		assertE2EStatus(t, rateDelete, "/api/rate-limit-policies delete", http.StatusNoContent, http.StatusOK, http.StatusNotFound)
	})

	t.Run("CertIssueAndSuggestions", func(t *testing.T) {
		certID := "e2e-selfsigned-cert"
		certIssue := postJSON(t, client, requestBaseURL+"/api/certificates/self-signed/issue", requestHostOverride, map[string]any{
			"certificate_id": certID,
			"common_name":    "e2e.localhost",
			"san_list":       []string{"www.e2e.localhost"},
		})
		assertE2EStatus(t, certIssue, "/api/certificates/self-signed/issue", http.StatusCreated, http.StatusOK)

		suggestCreate := postJSON(t, client, requestBaseURL+"/api/anti-ddos/rule-suggestions", requestHostOverride, map[string]any{
			"path_prefix": "/.env",
			"hits":        50,
			"unique_ips":  20,
		})
		assertE2EStatus(t, suggestCreate, "/api/anti-ddos/rule-suggestions create", http.StatusCreated, http.StatusOK)
	})
}

func requestE2EJSON(t *testing.T, client *http.Client, method, endpoint, hostOverride string, payload any) *http.Response {
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
			t.Fatalf("%s %s failed: %v", method, endpoint, err)
		}
		return resp
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload for %s: %v", endpoint, err)
	}
	req, err := newE2ERequest(method, endpoint, hostOverride, "application/json", strings.NewReader(string(body)))
	if err != nil {
		t.Fatalf("create %s %s request: %v", method, endpoint, err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := doPreparedE2ERequest(client, req, true)
	if err != nil {
		t.Fatalf("%s %s failed: %v", method, endpoint, err)
	}
	return resp
}

func assertE2EStatus(t *testing.T, resp *http.Response, action string, allowed ...int) {
	t.Helper()
	for _, s := range allowed {
		if resp.StatusCode == s {
			_ = resp.Body.Close()
			return
		}
	}
	t.Fatalf("%s failed: status=%d body=%s", action, resp.StatusCode, mustReadBody(t, resp.Body))
}
