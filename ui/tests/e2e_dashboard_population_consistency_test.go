package tests

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestE2EDashboardAttackPopulationConsistency(t *testing.T) {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	if baseURL == "" {
		t.Skip("WAF_E2E_BASE_URL is not set; skipping dashboard population consistency test")
	}

	client, requestBaseURL, requestHostOverride := newE2EClientAndBase(t, baseURL)
	loginE2EUser(t, client, requestBaseURL, requestHostOverride)

	resp := getWithAuth(t, client, requestBaseURL+"/api/dashboard/stats", requestHostOverride)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("dashboard/stats status=%d body=%s", resp.StatusCode, string(body))
	}
	defer resp.Body.Close()

	var stats map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		t.Fatalf("decode dashboard/stats: %v", err)
	}

	attacksDay := e2eDashboardAsInt(t, stats["attacks_day"])
	requestsDay := e2eDashboardAsInt(t, stats["requests_day"])
	blockedDay := e2eDashboardAsInt(t, stats["blocked_attacks_day"])
	topIPs := e2eDashboardAsSlice(t, stats["top_attacker_ips"], "top_attacker_ips")
	requestSeries := e2eDashboardAsSlice(t, stats["requests_series"], "requests_series")
	blockedSeries := e2eDashboardAsSlice(t, stats["blocked_series"], "blocked_series")
	if len(requestSeries) != 24 || len(blockedSeries) != 24 {
		t.Fatalf("dashboard must expose 24 hourly buckets, got requests=%d blocked=%d", len(requestSeries), len(blockedSeries))
	}
	if got := e2eDashboardSeriesTotal(t, requestSeries); got != requestsDay {
		t.Fatalf("dashboard inconsistency: request series total=%d requests_day=%d", got, requestsDay)
	}
	if got := e2eDashboardSeriesTotal(t, blockedSeries); got > blockedDay {
		t.Fatalf("dashboard inconsistency: blocked series total=%d blocked_attacks_day=%d", got, blockedDay)
	}

	if len(topIPs) == 0 {
		t.Skip("dashboard currently has no top_attacker_ips; skipping live consistency assertion until attack telemetry exists")
	}

	if attacksDay <= 0 {
		t.Fatalf("dashboard inconsistency: top_attacker_ips present but attacks_day=%d", attacksDay)
	}
	for _, raw := range topIPs {
		item, ok := raw.(map[string]any)
		if !ok { t.Fatalf("top_attacker_ips item has unexpected type %T", raw) }
		if strings.TrimSpace(item["key"].(string)) == "" || e2eDashboardAsInt(t, item["count"]) <= 0 {
			t.Fatalf("top attacker IP item is incomplete: %#v", item)
		}
	}
	if os.Getenv("WAF_E2E_DASHBOARD_SEEDED") == "1" {
		topSites := e2eDashboardAsSlice(t, stats["request_top_sites"], "request_top_sites")
		if len(topSites) < 2 || len(topIPs) == 0 {
			t.Fatalf("deterministic dashboard seed was not fully aggregated: sites=%d ips=%d", len(topSites), len(topIPs))
		}
		first, _ := topIPs[0].(map[string]any)
		if strings.TrimSpace(first["country"].(string)) == "" {
			t.Fatalf("seeded top attacker IP lacks country metadata: %#v", first)
		}
	}
}

func e2eDashboardAsInt(t *testing.T, v any) int {
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
		t.Fatalf("expected numeric dashboard value, got %T (%v)", v, v)
		return 0
	}
}

func e2eDashboardAsSlice(t *testing.T, v any, field string) []any {
	t.Helper()
	items, ok := v.([]any)
	if !ok {
		t.Fatalf("expected dashboard field %s to be []any, got %T (%v)", field, v, v)
	}
	return items
}

func e2eDashboardSeriesTotal(t *testing.T, values []any) int {
	t.Helper()
	total := 0
	for _, raw := range values {
		item, ok := raw.(map[string]any)
		if !ok { t.Fatalf("dashboard series item has unexpected type %T", raw) }
		total += e2eDashboardAsInt(t, item["count"])
	}
	return total
}
