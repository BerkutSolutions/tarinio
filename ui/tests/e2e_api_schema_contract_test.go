package tests

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestE2EAPISchemaContracts(t *testing.T) {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	if baseURL == "" {
		t.Skip("WAF_E2E_BASE_URL is not set; skipping schema contracts")
	}
	client, requestBaseURL, requestHostOverride := newE2EClientAndBase(t, baseURL)
	loginE2EUser(t, client, requestBaseURL, requestHostOverride)

	t.Run("AppMetaContract", func(t *testing.T) {
		resp := getWithAuth(t, client, requestBaseURL+"/api/app/meta", requestHostOverride)
		if resp.StatusCode != 200 {
			t.Fatalf("app/meta status=%d body=%s", resp.StatusCode, mustReadBody(t, resp.Body))
		}
		var m map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
			t.Fatalf("decode app/meta: %v", err)
		}
		requireKeys(t, m, "version")
	})

	t.Run("DashboardStatsContract", func(t *testing.T) {
		resp := getWithAuth(t, client, requestBaseURL+"/api/dashboard/stats", requestHostOverride)
		if resp.StatusCode != 200 {
			t.Fatalf("dashboard/stats status=%d body=%s", resp.StatusCode, mustReadBody(t, resp.Body))
		}
		var m map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
			t.Fatalf("decode dashboard/stats: %v", err)
		}
		requireKeys(t, m, "services_up", "services_down", "requests_day", "attacks_day", "blocked_attacks_day", "services")
		requireKeys(t, m, "top_attacker_ips", "top_attacker_countries", "most_attacked_urls")
		if _, ok := m["top_attacker_ips"].([]any); !ok {
			t.Fatalf("dashboard/stats top_attacker_ips has unexpected type: %#v", m["top_attacker_ips"])
		}
		if _, ok := m["top_attacker_countries"].([]any); !ok {
			t.Fatalf("dashboard/stats top_attacker_countries has unexpected type: %#v", m["top_attacker_countries"])
		}
		if _, ok := m["most_attacked_urls"].([]any); !ok {
			t.Fatalf("dashboard/stats most_attacked_urls has unexpected type: %#v", m["most_attacked_urls"])
		}
		attackCount := asIntContract(t, m["attacks_day"])
		ips := m["top_attacker_ips"].([]any)
		if len(ips) > 0 {
			if attackCount <= 0 {
				t.Fatalf("dashboard/stats inconsistent: top_attacker_ips present but attacks_day=%d", attackCount)
			}
		}
	})

	t.Run("EventsContract", func(t *testing.T) {
		resp := getWithAuth(t, client, requestBaseURL+"/api/events", requestHostOverride)
		if resp.StatusCode != 200 {
			t.Fatalf("events status=%d body=%s", resp.StatusCode, mustReadBody(t, resp.Body))
		}
		var payload map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			t.Fatalf("decode events: %v", err)
		}
		requireKeys(t, payload, "events")
	})

	t.Run("AuditContract", func(t *testing.T) {
		resp := getWithAuth(t, client, requestBaseURL+"/api/audit?limit=3&offset=0", requestHostOverride)
		if resp.StatusCode != 200 {
			t.Fatalf("audit status=%d body=%s", resp.StatusCode, mustReadBody(t, resp.Body))
		}
		var payload map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			t.Fatalf("decode audit: %v", err)
		}
		requireKeys(t, payload, "items")
	})

	t.Run("UIBootstrapContracts", func(t *testing.T) {
		uiPages := []string{"/dashboard", "/sites", "/settings", "/administration"}
		for _, path := range uiPages {
			resp := getWithAuth(t, client, requestBaseURL+path, requestHostOverride)
			if resp.StatusCode != 200 {
				t.Fatalf("page %s status=%d body=%s", path, resp.StatusCode, mustReadBody(t, resp.Body))
			}
			body := mustReadBody(t, resp.Body)
			if !strings.Contains(body, `id="content-area"`) {
				t.Fatalf("page %s missing content-area", path)
			}
		}
	})
}

func requireKeys(t *testing.T, data map[string]any, keys ...string) {
	t.Helper()
	for _, k := range keys {
		if _, ok := data[k]; !ok {
			t.Fatalf("missing key %q in %#v", k, data)
		}
	}
}

func asIntContract(t *testing.T, v any) int {
	t.Helper()
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int32:
		return int(n)
	case int64:
		return int(n)
	default:
		t.Fatalf("expected numeric dashboard contract value, got %T (%v)", v, v)
		return 0
	}
}
