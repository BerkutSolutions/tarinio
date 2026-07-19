package tests

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
)

// TestE2ERequestStorageParity proves that the runtime storage aggregate is
// authoritative regardless of request-list pagination and timezone grouping.
// The common E2E runner executes this against the configured file, OpenSearch
// or ClickHouse request backend.
func TestE2ERequestStorageParity(t *testing.T) {
	panelURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	runtimeURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_RUNTIME_HEALTH_URL")), "/")
	token := strings.TrimSpace(os.Getenv("WAF_E2E_RUNTIME_API_TOKEN"))
	if panelURL == "" || runtimeURL == "" || token == "" {
		t.Skip("E2E panel URL, runtime URL or runtime token is not configured")
	}

	compact := e2eRuntimeDashboardSummary(t, runtimeURL, token, "limit=1&retention_days=1&tz_offset_minutes=0")
	wide := e2eRuntimeDashboardSummary(t, runtimeURL, token, "limit=1000&retention_days=1&tz_offset_minutes=0")
	moscow := e2eRuntimeDashboardSummary(t, runtimeURL, token, "limit=1000&retention_days=1&tz_offset_minutes=180")
	for _, field := range []string{"requests_day", "blocked_day", "attacks_day", "unique_ips_day", "unique_attacker_ips_day"} {
		if e2eStorageNumber(t, compact[field]) != e2eStorageNumber(t, wide[field]) || e2eStorageNumber(t, wide[field]) != e2eStorageNumber(t, moscow[field]) {
			t.Fatalf("request aggregate field %s depends on pagination or timezone: compact=%v wide=%v moscow=%v", field, compact[field], wide[field], moscow[field])
		}
	}
	for _, field := range []string{"requests_series", "blocked_series"} {
		if e2eStorageSeriesTotal(t, compact[field], field) != e2eStorageSeriesTotal(t, wide[field], field) || e2eStorageSeriesTotal(t, wide[field], field) != e2eStorageSeriesTotal(t, moscow[field], field) {
			t.Fatalf("request aggregate series %s changes its total across pagination/timezone", field)
		}
		if got := len(e2eStorageSlice(t, wide[field], field)); got != 24 {
			t.Fatalf("runtime aggregate %s must contain 24 hourly buckets, got %d", field, got)
		}
	}

}

func e2eRuntimeDashboardSummary(t *testing.T, runtimeURL, token, rawQuery string) map[string]any {
	t.Helper()
	endpoint := runtimeURL + "/requests/dashboard-summary"
	if values, err := url.ParseQuery(rawQuery); err == nil {
		endpoint += "?" + values.Encode()
	}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		t.Fatalf("create runtime summary request: %v", err)
	}
	req.Header.Set("X-WAF-Runtime-Token", token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("get runtime summary: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("runtime summary: status=%d body=%s", resp.StatusCode, string(body))
	}
	var summary map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&summary); err != nil {
		t.Fatalf("decode runtime summary: %v", err)
	}
	return summary
}

func e2eStorageNumber(t *testing.T, value any) int {
	t.Helper()
	return e2eDashboardAsInt(t, value)
}

func e2eStorageSlice(t *testing.T, value any, field string) []any {
	t.Helper()
	return e2eDashboardAsSlice(t, value, field)
}

func e2eStorageSeriesTotal(t *testing.T, value any, field string) int {
	t.Helper()
	return e2eDashboardSeriesTotal(t, e2eStorageSlice(t, value, field))
}
