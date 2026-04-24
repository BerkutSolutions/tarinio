package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSecurityEventSourceSkipsStaticBurstNoise(t *testing.T) {
	t.Setenv("WAF_SECURITY_EVENT_BURST_RPS_THRESHOLD", "25")
	t.Setenv("WAF_SECURITY_EVENT_BURST_PATH_RPS_THRESHOLD", "10")

	lines := make([]string, 0, 40)
	for i := 0; i < 40; i++ {
		lines = append(lines, mustMarshalAccessLogLine(t, map[string]any{
			"timestamp":  "2026-04-19T20:11:40Z",
			"client_ip":  "46.159.189.39",
			"method":     "GET",
			"uri":        "/_static/dist/sentry/app.js?v=1",
			"status":     200,
			"site":       "sentry_hantico_ru",
			"host":       "sentry.hantico.ru",
			"country":    "RU",
			"user_agent": "Mozilla/5.0",
		}))
	}

	events := readSecurityEventsFromLines(t, lines)
	if len(events) != 0 {
		t.Fatalf("expected static asset burst to be ignored, got %+v", events)
	}
}

func TestSecurityEventSourceSeparatesBurstScopeBySite(t *testing.T) {
	t.Setenv("WAF_SECURITY_EVENT_BURST_RPS_THRESHOLD", "25")
	t.Setenv("WAF_SECURITY_EVENT_BURST_PATH_RPS_THRESHOLD", "10")

	lines := make([]string, 0, 30)
	for i := 0; i < 15; i++ {
		lines = append(lines, mustMarshalAccessLogLine(t, map[string]any{
			"timestamp":  "2026-04-19T20:16:40Z",
			"client_ip":  "46.159.189.39",
			"method":     "GET",
			"uri":        "/dashboard-" + string(rune('a'+i)),
			"status":     200,
			"site":       "waf_hantico_ru",
			"host":       "waf.hantico.ru",
			"country":    "RU",
			"user_agent": "Mozilla/5.0",
		}))
		lines = append(lines, mustMarshalAccessLogLine(t, map[string]any{
			"timestamp":  "2026-04-19T20:16:40Z",
			"client_ip":  "46.159.189.39",
			"method":     "GET",
			"uri":        "/issues-" + string(rune('a'+i)),
			"status":     200,
			"site":       "sentry_hantico_ru",
			"host":       "sentry.hantico.ru",
			"country":    "RU",
			"user_agent": "Mozilla/5.0",
		}))
	}

	events := readSecurityEventsFromLines(t, lines)
	if len(events) != 0 {
		t.Fatalf("expected per-site burst accounting to avoid cross-site false positives, got %+v", events)
	}
}

func TestSecurityEventSourceDetectsDynamicBurst(t *testing.T) {
	t.Setenv("WAF_SECURITY_EVENT_BURST_RPS_THRESHOLD", "25")
	t.Setenv("WAF_SECURITY_EVENT_BURST_PATH_RPS_THRESHOLD", "10")

	lines := make([]string, 0, 26)
	for i := 0; i < 26; i++ {
		lines = append(lines, mustMarshalAccessLogLine(t, map[string]any{
			"timestamp":  "2026-04-19T20:16:40Z",
			"client_ip":  "46.159.189.39",
			"method":     "POST",
			"uri":        "/api/orders",
			"status":     200,
			"site":       "waf_hantico_ru",
			"host":       "waf.hantico.ru",
			"country":    "RU",
			"user_agent": "Mozilla/5.0",
		}))
	}

	events := readSecurityEventsFromLines(t, lines)
	if len(events) != 2 {
		t.Fatalf("expected overall and path burst events, got %+v", events)
	}
}

func TestSecurityEventSourceSkipsAdminAppTrafficOnPublicHost(t *testing.T) {
	lines := []string{
		mustMarshalAccessLogLine(t, map[string]any{
			"timestamp":  "2026-04-23T11:21:00Z",
			"client_ip":  "46.159.189.39",
			"method":     "GET",
			"uri":        "/api/events",
			"status":     403,
			"site":       "waf_hantico_ru",
			"host":       "waf.hantico.ru",
			"country":    "RU",
			"user_agent": "Mozilla/5.0",
		}),
		mustMarshalAccessLogLine(t, map[string]any{
			"timestamp":  "2026-04-23T11:21:00Z",
			"client_ip":  "198.51.100.7",
			"method":     "GET",
			"uri":        "/checkout",
			"status":     403,
			"site":       "shop_example_com",
			"host":       "shop.example.com",
			"country":    "DE",
			"user_agent": "Mozilla/5.0",
		}),
	}

	events := readSecurityEventsFromLines(t, lines)
	if len(events) != 1 {
		t.Fatalf("expected only non-admin security event to remain, got %+v", events)
	}
	if got := events[0].Summary; got != "access blocked" {
		t.Fatalf("expected surviving event to be access blocked, got %q", got)
	}
	if got := events[0].Details["path"]; got != "/checkout" {
		t.Fatalf("expected surviving event path to be /checkout, got %#v", got)
	}
}

func TestSecurityEventSourceThreatIntelEmitsReputationEvent(t *testing.T) {
	root := t.TempDir()
	feedPath := filepath.Join(root, "threat-intel-feed.txt")
	if err := os.WriteFile(feedPath, []byte("203.0.113.10,77,scanner\n"), 0o644); err != nil {
		t.Fatalf("write threat intel feed: %v", err)
	}
	t.Setenv("WAF_THREAT_INTEL_FEEDS", feedPath)
	t.Setenv("WAF_THREAT_INTEL_REFRESH_SECONDS", "3600")

	lines := []string{
		mustMarshalAccessLogLine(t, map[string]any{
			"timestamp":  "2026-04-24T12:00:00Z",
			"client_ip":  "203.0.113.10",
			"method":     "GET",
			"uri":        "/catalog",
			"status":     200,
			"site":       "shop_example_com",
			"host":       "shop.example.com",
			"country":    "DE",
			"city":       "Berlin",
			"user_agent": "Mozilla/5.0",
		}),
		mustMarshalAccessLogLine(t, map[string]any{
			"timestamp":  "2026-04-24T12:00:00Z",
			"client_ip":  "203.0.113.10",
			"method":     "GET",
			"uri":        "/cart",
			"status":     200,
			"site":       "shop_example_com",
			"host":       "shop.example.com",
			"country":    "DE",
			"city":       "Berlin",
			"user_agent": "Mozilla/5.0",
		}),
	}

	events := readSecurityEventsFromLines(t, lines)
	intelEvents := make([]securityEvent, 0, 1)
	for _, event := range events {
		if event.Type == "security_threat_intel" {
			intelEvents = append(intelEvents, event)
		}
	}
	if len(intelEvents) != 1 {
		t.Fatalf("expected exactly one threat intel event per ip/feed per tick, got %+v", intelEvents)
	}
	if got := asString(intelEvents[0].Details["city"]); got != "Berlin" {
		t.Fatalf("expected city in threat intel event, got %q", got)
	}
	if got := parseIntValue(intelEvents[0].Details["intel_score"]); got != 77 {
		t.Fatalf("expected intel score 77, got %d", got)
	}
}

func readSecurityEventsFromLines(t *testing.T, lines []string) []securityEvent {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, "access.log")
	content := []byte{}
	for _, line := range lines {
		content = append(content, []byte(line+"\n")...)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write access log: %v", err)
	}
	source := newSecurityEventSource(path)
	events, err := source.next()
	if err != nil {
		t.Fatalf("read security events: %v", err)
	}
	return events
}

func mustMarshalAccessLogLine(t *testing.T, payload map[string]any) string {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal access log payload: %v", err)
	}
	return string(raw)
}
