package tests

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestE2EBurstStability(t *testing.T) {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("WAF_E2E_BASE_URL")), "/")
	if baseURL == "" {
		t.Skip("WAF_E2E_BASE_URL is not set; skipping burst stability")
	}
	client, requestBaseURL, requestHostOverride := newE2EClientAndBase(t, baseURL)
	loginE2EUser(t, client, requestBaseURL, requestHostOverride)

	targets := []string{
		"/api/app/meta",
		"/api/auth/me",
		"/api/dashboard/stats",
		"/api/sites",
		"/api/upstreams",
		"/api/tls-configs",
		"/api/events",
		"/api/requests?limit=5",
		"/api/audit?limit=5",
		"/dashboard",
		"/sites",
		"/settings",
	}

	total := 0
	hardFailures := 0
	softRate := 0
	hardFailureDetails := make([]string, 0)
	for i := 0; i < 5; i++ {
		for _, path := range targets {
			total++
			resp := getWithAuthRetry429(t, client, requestBaseURL+path, requestHostOverride, 2)
			if resp.StatusCode == 429 {
				softRate++
				_ = resp.Body.Close()
				continue
			}
			if resp.StatusCode >= 500 || resp.StatusCode == 0 {
				hardFailures++
				hardFailureDetails = append(hardFailureDetails, path+"="+resp.Status)
				_ = resp.Body.Close()
				continue
			}
			_ = resp.Body.Close()
		}
		time.Sleep(120 * time.Millisecond)
	}

	if hardFailures > 0 {
		t.Fatalf("burst stability hard failures=%d total=%d soft429=%d details=%v", hardFailures, total, softRate, hardFailureDetails)
	}
	t.Logf("burst stability: total=%d hardFailures=%d soft429=%d", total, hardFailures, softRate)
}
