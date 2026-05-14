package tests

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestE2ERecoveryFlow_BadConfigThenRepair(t *testing.T) {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	if baseURL == "" {
		t.Skip("WAF_E2E_BASE_URL is not set; skipping recovery flow")
	}
	client, requestBaseURL, requestHostOverride := newE2EClientAndBase(t, baseURL)
	loginE2EUser(t, client, requestBaseURL, requestHostOverride)

	t.Run("RejectInvalidAntiDDoSConfig", func(t *testing.T) {
		invalidResp := requestDeepJSON(t, client, "PUT", requestBaseURL+"/api/anti-ddos/settings", requestHostOverride, map[string]any{
			"use_l4_guard":    true,
			"chain_mode":      "auto",
			"conn_limit":      -1,
			"rate_per_second": 100,
			"rate_burst":      200,
			"ports":           []int{80, 443},
			"target":          "REJECT",
		})
		if invalidResp.StatusCode != 400 {
			t.Fatalf("expected 400 for invalid anti-ddos config, got=%d body=%s", invalidResp.StatusCode, mustReadBody(t, invalidResp.Body))
		}
		_ = invalidResp.Body.Close()
	})

	t.Run("ApplyValidAntiDDoSConfig", func(t *testing.T) {
		validResp := requestDeepJSON(t, client, "PUT", requestBaseURL+"/api/anti-ddos/settings", requestHostOverride, map[string]any{
			"use_l4_guard":           false,
			"chain_mode":             "auto",
			"conn_limit":             120,
			"rate_per_second":        60,
			"rate_burst":             120,
			"ports":                  []int{80, 443},
			"target":                 "REJECT",
			"enforce_l7_rate_limit":  true,
			"l7_requests_per_second": 20,
			"l7_burst":               40,
			"l7_status_code":         429,
		})
		if validResp.StatusCode != 200 {
			t.Fatalf("expected 200 for valid anti-ddos config, got=%d body=%s", validResp.StatusCode, mustReadBody(t, validResp.Body))
		}
		_ = validResp.Body.Close()
	})

	t.Run("CompileAndApplyAfterRepair", func(t *testing.T) {
		compileResp := postJSON(t, client, requestBaseURL+"/api/revisions/compile", requestHostOverride, map[string]any{})
		if compileResp.StatusCode != 201 {
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
			t.Fatalf("compile returned empty revision id")
		}
		applyResp := postJSON(t, client, requestBaseURL+"/api/revisions/"+payload.Revision.ID+"/apply", requestHostOverride, map[string]any{})
		if applyResp.StatusCode != 201 {
			t.Fatalf("apply after repair failed: status=%d body=%s", applyResp.StatusCode, mustReadBody(t, applyResp.Body))
		}
		_ = applyResp.Body.Close()
	})
}
