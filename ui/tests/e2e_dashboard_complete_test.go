//go:build e2e

package tests

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
)

func TestE2EDashboardComplete(t *testing.T) {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	if baseURL == "" {
		t.Fatal("WAF_E2E_BASE_URL is not set; skipping complete dashboard e2e")
	}
	client, requestBaseURL, hostOverride := newE2EClientAndBase(t, baseURL)
	loginE2EUser(t, client, requestBaseURL, hostOverride)

	t.Run("SeriesTotalsAndBreakdowns", func(t *testing.T) {
		stats := getDashboardJSON(t, client, requestBaseURL+"/api/dashboard/stats", hostOverride)
		assertDashboardSeries(t, stats, "requests_series", "requests_day")
		assertDashboardSeries(t, stats, "attacks_series", "attacks_day")
		assertDashboardSeries(t, stats, "blocked_series", "blocked_attacks_day")
		for _, field := range []string{"request_top_sites", "request_top_urls", "top_attacker_ips", "top_attacker_countries", "most_attacked_urls", "popular_errors"} {
			items, ok := stats[field].([]any)
			if !ok || len(items) == 0 {
				t.Fatalf("dashboard %s must contain deterministic seeded data: %#v", field, stats[field])
			}
		}
	})

	t.Run("ContainersOverviewAndLogs", func(t *testing.T) {
		overview := getDashboardJSON(t, client, requestBaseURL+"/api/dashboard/containers/overview", hostOverride)
		containers, ok := overview["containers"].([]any)
		if !ok || len(containers) == 0 {
			t.Fatalf("container overview is empty: %#v", overview)
		}
		var name string
		for _, raw := range containers {
			item, _ := raw.(map[string]any)
			if strings.EqualFold(strings.TrimSpace(asStringContract(item["state"])), "running") {
				name = strings.TrimSpace(asStringContract(item["name"]))
				break
			}
		}
		if name == "" {
			t.Fatal("container overview has no running container")
		}
		logs := getDashboardJSON(t, client, requestBaseURL+"/api/dashboard/containers/logs?tail=5&container="+url.QueryEscape(name), hostOverride)
		if got := strings.TrimSpace(asStringContract(logs["container"])); got != name {
			t.Fatalf("logs container=%q want=%q: %#v", got, name, logs)
		}
		if _, ok := logs["lines"].([]any); !ok {
			t.Fatalf("logs lines has unexpected type: %#v", logs["lines"])
		}

		bad := getWithAuth(t, client, requestBaseURL+"/api/dashboard/containers/logs?tail=bad&container="+url.QueryEscape(name), hostOverride)
		if bad.StatusCode != http.StatusBadRequest {
			t.Fatalf("invalid logs tail status=%d body=%s", bad.StatusCode, mustReadBody(t, bad.Body))
		}
		_ = bad.Body.Close()
	})
}

func getDashboardJSON(t *testing.T, client *http.Client, endpoint, hostOverride string) map[string]any {
	t.Helper()
	resp := getWithAuth(t, client, endpoint, hostOverride)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s status=%d body=%s", endpoint, resp.StatusCode, mustReadBody(t, resp.Body))
	}
	defer resp.Body.Close()
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode %s: %v", endpoint, err)
	}
	return payload
}

func assertDashboardSeries(t *testing.T, stats map[string]any, seriesField, totalField string) {
	t.Helper()
	series, ok := stats[seriesField].([]any)
	if !ok || len(series) != 24 {
		t.Fatalf("%s must contain 24 buckets: %#v", seriesField, stats[seriesField])
	}
	total := 0
	for _, raw := range series {
		item, _ := raw.(map[string]any)
		if strings.TrimSpace(asStringContract(item["timestamp"])) == "" {
			t.Fatalf("%s bucket is missing timestamp: %#v", seriesField, item)
		}
		total += asIntContract(t, item["count"])
	}
	if want := asIntContract(t, stats[totalField]); total != want {
		t.Fatalf("%s total=%d want %s=%d", seriesField, total, totalField, want)
	}
}

func asStringContract(value any) string {
	text, _ := value.(string)
	return text
}
