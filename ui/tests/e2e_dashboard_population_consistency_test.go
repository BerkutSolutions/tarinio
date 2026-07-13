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
	topIPs := e2eDashboardAsSlice(t, stats["top_attacker_ips"], "top_attacker_ips")

	if len(topIPs) == 0 {
		t.Skip("dashboard currently has no top_attacker_ips; skipping live consistency assertion until attack telemetry exists")
	}

	if attacksDay <= 0 {
		t.Fatalf("dashboard inconsistency: top_attacker_ips present but attacks_day=%d", attacksDay)
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
