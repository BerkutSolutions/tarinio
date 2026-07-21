package tests

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestE2EDashboardAttackPopulationConsistency(t *testing.T) {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	if baseURL == "" {
		t.Skip("WAF_E2E_BASE_URL is not set; skipping dashboard population consistency test")
	}
	composeFile := strings.TrimSpace(os.Getenv("WAF_E2E_COMPOSE_FILE"))
	if composeFile == "" {
		t.Fatal("WAF_E2E_COMPOSE_FILE is required for dashboard attack telemetry")
	}

	client, requestBaseURL, requestHostOverride := newE2EClientAndBase(t, baseURL)
	loginE2EUser(t, client, requestBaseURL, requestHostOverride)
	siteID := e2eUniqueID(t, "e2e-dashboard")
	upstreamID, host := siteID+"-upstream", siteID+".test"
	deleteProtectionResource(t, client, requestBaseURL+"/api/sites/"+siteID+"?auto_apply=false", requestHostOverride)
	deleteProtectionResource(t, client, requestBaseURL+"/api/upstreams/"+upstreamID+"?auto_apply=false", requestHostOverride)
	createE2EModSecuritySite(t, client, requestBaseURL, requestHostOverride, siteID, upstreamID, host)
	t.Cleanup(func() {
		deleteProtectionResource(t, client, requestBaseURL+"/api/sites/"+siteID+"?auto_apply=false", requestHostOverride)
		deleteProtectionResource(t, client, requestBaseURL+"/api/upstreams/"+upstreamID+"?auto_apply=false", requestHostOverride)
	})
	profile := e2eGetEasyProfile(t, client, requestBaseURL, requestHostOverride, siteID)
	antibot := mapGetOrCreate(profile, "security_antibot")
	antibot["antibot_challenge"] = "javascript"
	antibot["antibot_uri"] = "/challenge"
	e2ePutProfile(t, client, requestBaseURL, requestHostOverride, siteID, profile)
	e2eCompileAndApply(t, client, requestBaseURL, requestHostOverride)
	waitForDashboardChallenge(t, strings.TrimRight(os.Getenv("WAF_E2E_RUNTIME_URL"), "/"), host)
	attackOutput := runProtectionAttacker(t, composeFile, "for i in $(seq 1 8); do wget -S -O /dev/null --header='Host: "+host+"' --post-data=x http://runtime/ 2>&1 || true; done")
	if !strings.Contains(attackOutput, "403") {
		t.Fatalf("attacker did not receive an anti-bot block response: %s", attackOutput)
	}
	time.Sleep(3 * time.Second)
	requestRows := e2eDashboardRequestRows(t, client, requestBaseURL, requestHostOverride)
	t.Logf("generated anti-bot telemetry rows: %d", len(requestRows))
	t.Logf("runtime dashboard summary after attack: %#v", e2eDirectRuntimeDashboardSummary(t))

	stats := waitForDashboardAttacker(t, client, requestBaseURL, requestHostOverride, "172.30.0.30", requestRows)

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
		t.Fatalf("dashboard did not aggregate the generated anti-bot telemetry into top_attacker_ips: stats=%#v requests=%#v", stats, requestRows)
	}

	if attacksDay <= 0 {
		t.Fatalf("dashboard inconsistency: top_attacker_ips present but attacks_day=%d", attacksDay)
	}
	for _, raw := range topIPs {
		item, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("top_attacker_ips item has unexpected type %T", raw)
		}
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

func e2eDirectRuntimeDashboardSummary(t *testing.T) map[string]any {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:18081/requests/dashboard-summary?since="+time.Now().UTC().Add(-24*time.Hour).Format(time.RFC3339), nil)
	if err != nil {
		t.Fatalf("create runtime dashboard summary request: %v", err)
	}
	request.Header.Set("X-WAF-Runtime-Token", "e2e-test-runtime-token")
	response, err := newE2EHTTPClient("http://127.0.0.1:18081", false).Do(request)
	if err != nil {
		t.Fatalf("fetch runtime dashboard summary: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("runtime dashboard summary status=%d body=%s", response.StatusCode, string(body))
	}
	var summary map[string]any
	if err := json.NewDecoder(response.Body).Decode(&summary); err != nil {
		t.Fatalf("decode runtime dashboard summary: %v", err)
	}
	return summary
}

func e2eDashboardRequestRows(t *testing.T, client *http.Client, baseURL, hostOverride string) []map[string]any {
	t.Helper()
	resp := getWithAuth(t, client, baseURL+"/api/requests?limit=100", hostOverride)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("requests status=%d body=%s", resp.StatusCode, string(body))
	}
	var rows []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		t.Fatalf("decode requests: %v", err)
	}
	return rows
}

func waitForDashboardAttacker(t *testing.T, client *http.Client, baseURL, hostOverride, attackerIP string, requestRows []map[string]any) map[string]any {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	var last map[string]any
	for time.Now().Before(deadline) {
		resp := getWithAuth(t, client, baseURL+"/api/dashboard/stats", hostOverride)
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			t.Fatalf("dashboard/stats status=%d body=%s", resp.StatusCode, string(body))
		}
		var stats map[string]any
		err := json.NewDecoder(resp.Body).Decode(&stats)
		_ = resp.Body.Close()
		if err != nil {
			t.Fatalf("decode dashboard/stats: %v", err)
		}
		last = stats
		for _, raw := range e2eDashboardAsSlice(t, stats["top_attacker_ips"], "top_attacker_ips") {
			item, ok := raw.(map[string]any)
			if ok && strings.TrimSpace(item["key"].(string)) == attackerIP && e2eDashboardAsInt(t, item["count"]) > 0 {
				return stats
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("dashboard did not aggregate the generated anti-bot telemetry for %s: stats=%#v requests=%#v", attackerIP, last, requestRows)
	return nil
}

func waitForDashboardChallenge(t *testing.T, runtimeURL, host string) {
	t.Helper()
	client := newE2EHTTPClient(runtimeURL, false)
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		req, err := http.NewRequest(http.MethodGet, runtimeURL+"/", nil)
		if err == nil {
			req.Host = host
			resp, requestErr := client.Do(req)
			if requestErr == nil {
				location := resp.Header.Get("Location")
				_ = resp.Body.Close()
				if resp.StatusCode == http.StatusFound && strings.Contains(location, "/challenge") {
					return
				}
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	t.Fatalf("runtime did not activate AntiBot challenge for %s", host)
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
		if !ok {
			t.Fatalf("dashboard series item has unexpected type %T", raw)
		}
		total += e2eDashboardAsInt(t, item["count"])
	}
	return total
}
