package services

import (
	"testing"
	"time"
)

func makeRequestRow(siteID, uri string, status int, when time.Time) map[string]any {
	return map[string]any{
		"entry": map[string]any{
			"timestamp": when.Format(time.RFC3339),
			"site":      siteID,
			"uri":       uri,
			"status":    float64(status),
		},
	}
}

func TestSummarizeUpstreamHealth_OK(t *testing.T) {
	now := time.Now().UTC()
	rows := []map[string]any{
		makeRequestRow("site-a", "/app", 200, now.Add(-1*time.Minute)),
		makeRequestRow("site-a", "/app/page", 200, now.Add(-2*time.Minute)),
		makeRequestRow("site-a", "/app/data", 200, now.Add(-3*time.Minute)),
	}
	result := summarizeUpstreamHealth(rows, now)
	if len(result) != 1 {
		t.Fatalf("expected 1 site, got %d", len(result))
	}
	h := result[0]
	if h.Status != upstreamHealthOK {
		t.Errorf("expected status ok, got %s (rate=%.2f)", h.Status, h.ErrorRate)
	}
}

func TestSummarizeUpstreamHealth_Warning(t *testing.T) {
	now := time.Now().UTC()
	// 10% 5xx → warning
	rows := make([]map[string]any, 0, 10)
	for i := 0; i < 9; i++ {
		rows = append(rows, makeRequestRow("site-b", "/app", 200, now.Add(-1*time.Minute)))
	}
	rows = append(rows, makeRequestRow("site-b", "/app", 502, now.Add(-1*time.Minute)))

	result := summarizeUpstreamHealth(rows, now)
	if len(result) != 1 {
		t.Fatalf("expected 1 site, got %d", len(result))
	}
	h := result[0]
	if h.Status != upstreamHealthWarning {
		t.Errorf("expected status warning, got %s (rate=%.2f)", h.Status, h.ErrorRate)
	}
}

func TestSummarizeUpstreamHealth_Critical(t *testing.T) {
	now := time.Now().UTC()
	// 20% 5xx → critical
	rows := make([]map[string]any, 0, 10)
	for i := 0; i < 8; i++ {
		rows = append(rows, makeRequestRow("site-c", "/app", 200, now.Add(-1*time.Minute)))
	}
	rows = append(rows, makeRequestRow("site-c", "/app", 500, now.Add(-1*time.Minute)))
	rows = append(rows, makeRequestRow("site-c", "/app", 503, now.Add(-1*time.Minute)))

	result := summarizeUpstreamHealth(rows, now)
	if len(result) != 1 {
		t.Fatalf("expected 1 site, got %d", len(result))
	}
	h := result[0]
	if h.Status != upstreamHealthCritical {
		t.Errorf("expected status critical, got %s (rate=%.2f)", h.Status, h.ErrorRate)
	}
}

func TestSummarizeUpstreamHealth_NoErrors(t *testing.T) {
	now := time.Now().UTC()
	rows := []map[string]any{
		makeRequestRow("site-d", "/app", 200, now.Add(-1*time.Minute)),
	}
	result := summarizeUpstreamHealth(rows, now)
	if len(result) != 1 {
		t.Fatalf("expected 1 site, got %d", len(result))
	}
	if result[0].ErrorRate != 0.0 {
		t.Errorf("expected 0%% error rate, got %.2f", result[0].ErrorRate)
	}
	if result[0].Status != upstreamHealthOK {
		t.Errorf("expected ok status, got %s", result[0].Status)
	}
}

func TestSummarizeUpstreamHealth_IgnoresOldRows(t *testing.T) {
	now := time.Now().UTC()
	rows := []map[string]any{
		// old 5xx outside window — should be ignored
		makeRequestRow("site-e", "/app", 500, now.Add(-10*time.Minute)),
		makeRequestRow("site-e", "/app", 200, now.Add(-1*time.Minute)),
	}
	result := summarizeUpstreamHealth(rows, now)
	if len(result) != 1 {
		t.Fatalf("expected 1 site, got %d", len(result))
	}
	if result[0].ErrorRate != 0.0 {
		t.Errorf("expected 0%% error rate (old rows ignored), got %.2f", result[0].ErrorRate)
	}
}

func TestSummarizeUpstreamHealth_WindowMinutes(t *testing.T) {
	now := time.Now().UTC()
	rows := []map[string]any{
		makeRequestRow("site-f", "/app", 200, now.Add(-1*time.Minute)),
	}
	result := summarizeUpstreamHealth(rows, now)
	if len(result) != 1 {
		t.Fatalf("expected 1 site")
	}
	if result[0].WindowMinutes != upstreamWindowMinutes {
		t.Errorf("expected WindowMinutes=%d, got %d", upstreamWindowMinutes, result[0].WindowMinutes)
	}
}
