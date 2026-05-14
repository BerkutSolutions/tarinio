package tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestE2EAPIEnterpriseMatrix(t *testing.T) {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	if baseURL == "" {
		t.Skip("WAF_E2E_BASE_URL is not set; skipping enterprise matrix")
	}

	client, requestBaseURL, requestHostOverride := newE2EClientAndBase(t, baseURL)
	loginE2EUser(t, client, requestBaseURL, requestHostOverride)

	t.Run("AuthMutations", func(t *testing.T) {
		changePasswordResp := postJSON(t, client, requestBaseURL+"/api/auth/change-password", requestHostOverride, map[string]any{
			"current_password": "admin",
			"password":         "admin",
		})
		assertAnyStatus(t, changePasswordResp, "/api/auth/change-password", http.StatusOK, http.StatusBadRequest, http.StatusForbidden)

		setupResp := postJSON(t, client, requestBaseURL+"/api/auth/2fa/setup", requestHostOverride, map[string]any{})
		assertAnyStatus(t, setupResp, "/api/auth/2fa/setup", http.StatusOK, http.StatusBadRequest, http.StatusForbidden)

		registerBeginResp := postJSON(t, client, requestBaseURL+"/api/auth/passkeys/register/begin", requestHostOverride, map[string]any{
			"name": "e2e-device",
		})
		assertAnyStatus(t, registerBeginResp, "/api/auth/passkeys/register/begin", http.StatusOK, http.StatusBadRequest, http.StatusForbidden)
	})

	t.Run("EnterpriseAdminEndpoints", func(t *testing.T) {
		paths := []string{
			"/api/administration/enterprise",
			"/api/administration/enterprise/scim-tokens",
			"/api/administration/support-bundle",
		}
		for _, path := range paths {
			resp := getWithAuthRetry429(t, client, requestBaseURL+path, requestHostOverride, 2)
			assertAnyStatus(t, resp, path, http.StatusOK, http.StatusForbidden, http.StatusNotImplemented)
		}
	})

	t.Run("SCIMSurface", func(t *testing.T) {
		paths := []string{
			"/scim/v2/ServiceProviderConfig",
			"/scim/v2/Users",
			"/scim/v2/Groups",
		}
		for _, path := range paths {
			resp := requestRaw(t, client, http.MethodGet, requestBaseURL+path, requestHostOverride, nil)
			assertAnyStatus(t, resp, path, http.StatusOK, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotImplemented)
		}
	})

	t.Run("AdministrationScriptsRunDownload", func(t *testing.T) {
		scriptsResp := getWithAuth(t, client, requestBaseURL+"/api/administration/scripts", requestHostOverride)
		if scriptsResp.StatusCode != http.StatusOK {
			t.Fatalf("scripts catalog failed: status=%d body=%s", scriptsResp.StatusCode, mustReadBody(t, scriptsResp.Body))
		}
		var scripts []map[string]any
		if err := json.NewDecoder(scriptsResp.Body).Decode(&scripts); err != nil {
			t.Fatalf("decode scripts catalog: %v", err)
		}
		if len(scripts) == 0 {
			t.Skip("no administration scripts in catalog")
		}

		scriptID := strings.TrimSpace(fmt.Sprint(scripts[0]["id"]))
		if scriptID == "" {
			t.Skip("first script has empty id")
		}
		runResp := postJSON(t, client, requestBaseURL+"/api/administration/scripts/"+scriptID+"/run", requestHostOverride, map[string]any{
			"input": map[string]any{},
		})
		assertAnyStatus(t, runResp, "run script", http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusBadRequest)

		if runResp.StatusCode == http.StatusOK || runResp.StatusCode == http.StatusCreated || runResp.StatusCode == http.StatusAccepted {
			var payload map[string]any
			if err := json.NewDecoder(runResp.Body).Decode(&payload); err != nil {
				t.Fatalf("decode run response: %v", err)
			}
			runID := strings.TrimSpace(fmt.Sprint(payload["run_id"]))
			if runID == "" {
				runID = strings.TrimSpace(fmt.Sprint(payload["id"]))
			}
			if runID != "" {
				downloadResp := requestRaw(t, client, http.MethodGet, requestBaseURL+"/api/administration/scripts/runs/"+runID+"/download", requestHostOverride, nil)
				assertAnyStatus(t, downloadResp, "download run artifact", http.StatusOK, http.StatusNotFound, http.StatusBadRequest)
				return
			}
			t.Skip("run response has no run id; skip download check")
		}
	})
}

func requestRaw(t *testing.T, client *http.Client, method, endpoint, hostOverride string, body any) *http.Response {
	t.Helper()
	if method == http.MethodGet || body == nil {
		req, err := http.NewRequest(method, endpoint, nil)
		if err != nil {
			t.Fatalf("create request %s %s: %v", method, endpoint, err)
		}
		req.Header.Set("Accept", "application/json,text/plain,*/*")
		if hostOverride != "" {
			req.Host = hostOverride
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("request failed %s %s: %v", method, endpoint, err)
		}
		return resp
	}
	return postJSON(t, client, endpoint, hostOverride, body)
}

func assertAnyStatus(t *testing.T, resp *http.Response, action string, allowed ...int) {
	t.Helper()
	for _, s := range allowed {
		if resp.StatusCode == s {
			_ = resp.Body.Close()
			return
		}
	}
	t.Fatalf("%s unexpected status=%d body=%s", action, resp.StatusCode, mustReadBody(t, resp.Body))
}
