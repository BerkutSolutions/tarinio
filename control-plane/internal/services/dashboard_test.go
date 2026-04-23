package services

import (
	"errors"
	"net/url"
	"testing"
	"time"

	"waf/control-plane/internal/events"
)

type fakeDashboardEventReader struct {
	items []events.Event
	err   error
}

func (f *fakeDashboardEventReader) List() ([]events.Event, error) {
	return append([]events.Event(nil), f.items...), f.err
}

func (f *fakeDashboardEventReader) Probe() error { return f.err }

type fakeDashboardRequestCollector struct {
	items []map[string]any
	err   error
}

func (f *fakeDashboardRequestCollector) Collect() ([]map[string]any, error) {
	return append([]map[string]any(nil), f.items...), f.err
}

func (f *fakeDashboardRequestCollector) Probe(_ url.Values) error { return f.err }

type fakeDashboardRuntimeProbe struct{ err error }

func (f *fakeDashboardRuntimeProbe) Probe() error { return f.err }

func TestDashboardService_StatsExposeCurrentWidgetData(t *testing.T) {
	now := time.Now().UTC()
	requests := &fakeDashboardRequestCollector{
		items: []map[string]any{
			{
				"ingested_at": now.Format(time.RFC3339),
				"entry": map[string]any{
					"timestamp": now.Format(time.RFC3339),
					"site":      "site-a",
					"uri":       "/checkout",
					"status":    200,
					"client_ip": "203.0.113.10",
					"method":    "GET",
					"country":   "RU",
				},
			},
			{
				"ingested_at": now.Add(-10 * time.Minute).Format(time.RFC3339),
				"entry": map[string]any{
					"timestamp": now.Add(-10 * time.Minute).Format(time.RFC3339),
					"site":      "site-a",
					"uri":       "/signin",
					"status":    429,
					"client_ip": "203.0.113.11",
					"method":    "POST",
					"country":   "US",
				},
			},
		},
	}
	eventReader := &fakeDashboardEventReader{
		items: []events.Event{
			{
				ID:              "evt-1",
				Type:            events.TypeSecurityWAF,
				Severity:        events.SeverityWarning,
				SiteID:          "site-a",
				SourceComponent: "runtime",
				OccurredAt:      now.Add(-5 * time.Minute).Format(time.RFC3339),
				Summary:         "blocked request",
				Details: map[string]any{
					"blocked":   true,
					"status":    403,
					"client_ip": "203.0.113.10",
					"path":      "/checkout",
					"country":   "RU",
				},
			},
		},
	}

	service := NewDashboardService(eventReader, requests, &fakeDashboardRuntimeProbe{})
	stats, err := service.Stats()
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}

	if stats.ServicesUp != 2 || stats.ServicesDown != 0 {
		t.Fatalf("unexpected service summary: up=%d down=%d", stats.ServicesUp, stats.ServicesDown)
	}
	if stats.RequestsDay != 2 {
		t.Fatalf("expected 2 requests, got %d", stats.RequestsDay)
	}
	if stats.AttacksDay != 1 {
		t.Fatalf("expected 1 attack, got %d", stats.AttacksDay)
	}
	if stats.BlockedAttacksDay != 1 {
		t.Fatalf("expected 1 blocked attack, got %d", stats.BlockedAttacksDay)
	}
	if len(stats.Services) != 2 || stats.Services[1].Name != "runtime" || !stats.Services[1].Up {
		t.Fatalf("expected runtime service to be up, got %#v", stats.Services)
	}
	if len(stats.TopAttackerIPs) == 0 || stats.TopAttackerIPs[0].Key != "203.0.113.10" {
		t.Fatalf("expected top attacker ip to be populated, got %#v", stats.TopAttackerIPs)
	}
	if len(stats.MostAttackedURLs) == 0 || stats.MostAttackedURLs[0].Key != "/checkout" {
		t.Fatalf("expected attacked urls to be populated, got %#v", stats.MostAttackedURLs)
	}
}

func TestDashboardService_RuntimeProbeFailureMarksRuntimeDown(t *testing.T) {
	service := NewDashboardService(
		&fakeDashboardEventReader{},
		&fakeDashboardRequestCollector{},
		&fakeDashboardRuntimeProbe{err: errors.New("runtime down")},
	)

	stats, err := service.Stats()
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}

	if stats.ServicesUp != 1 || stats.ServicesDown != 1 {
		t.Fatalf("unexpected service summary: up=%d down=%d", stats.ServicesUp, stats.ServicesDown)
	}
	if len(stats.Services) != 2 || stats.Services[1].Name != "runtime" || stats.Services[1].Up {
		t.Fatalf("expected runtime service to be down, got %#v", stats.Services)
	}
}

func TestDashboardService_RuntimeRequestsFailureDoesNotFailStats(t *testing.T) {
	now := time.Now().UTC()
	service := NewDashboardService(
		&fakeDashboardEventReader{
			items: []events.Event{
				{
					ID:         "evt-1",
					Type:       events.TypeSecurityWAF,
					SiteID:     "site-a",
					OccurredAt: now.Format(time.RFC3339),
					Details: map[string]any{
						"blocked":   true,
						"status":    403,
						"client_ip": "203.0.113.10",
						"path":      "/checkout",
					},
				},
			},
		},
		&fakeDashboardRequestCollector{err: errors.New("runtime requests unavailable")},
		&fakeDashboardRuntimeProbe{},
	)

	stats, err := service.Stats()
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}
	if stats.RequestsDay != 0 {
		t.Fatalf("expected requests_day=0 on request collector failure, got %d", stats.RequestsDay)
	}
	if stats.AttacksDay != 1 {
		t.Fatalf("expected attack stats from events to stay available, got %d", stats.AttacksDay)
	}
}

func TestDashboardService_FallsBackToBlockedRequestsForAttackWidgets(t *testing.T) {
	now := time.Now().UTC()
	service := NewDashboardService(
		&fakeDashboardEventReader{},
		&fakeDashboardRequestCollector{
			items: []map[string]any{
				{
					"ingested_at": now.Format(time.RFC3339),
					"entry": map[string]any{
						"timestamp": now.Format(time.RFC3339),
						"site":      "site-b",
						"uri":       "/signin",
						"status":    403,
						"client_ip": "198.51.100.7",
						"country":   "DE",
						"method":    "POST",
					},
				},
				{
					"ingested_at": now.Add(-time.Minute).Format(time.RFC3339),
					"entry": map[string]any{
						"timestamp": now.Add(-time.Minute).Format(time.RFC3339),
						"site":      "site-b",
						"uri":       "/checkout",
						"status":    429,
						"client_ip": "198.51.100.8",
						"country":   "FR",
						"method":    "GET",
					},
				},
			},
		},
		&fakeDashboardRuntimeProbe{},
	)

	stats, err := service.Stats()
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}

	if stats.AttacksDay != 2 {
		t.Fatalf("expected blocked requests fallback to expose 2 attacks, got %d", stats.AttacksDay)
	}
	if stats.BlockedAttacksDay != 2 {
		t.Fatalf("expected blocked requests fallback to expose 2 blocked attacks, got %d", stats.BlockedAttacksDay)
	}
	if stats.UniqueAttackerIPsDay != 2 {
		t.Fatalf("expected 2 unique attacker IPs from requests fallback, got %d", stats.UniqueAttackerIPsDay)
	}
	if len(stats.TopAttackerIPs) == 0 || stats.TopAttackerIPs[0].Key != "198.51.100.7" {
		t.Fatalf("expected top attacker IPs to be populated from blocked requests, got %#v", stats.TopAttackerIPs)
	}
	if len(stats.TopAttackerCountries) == 0 {
		t.Fatalf("expected top attacker countries to be populated from blocked requests")
	}
	if len(stats.MostAttackedURLs) == 0 {
		t.Fatalf("expected attacked URLs to be populated from blocked requests")
	}
}

func TestDashboardService_SkipsInternalManagementTraffic(t *testing.T) {
	now := time.Now().UTC()
	service := NewDashboardService(
		&fakeDashboardEventReader{
			items: []events.Event{
				{
					ID:         "evt-ui",
					Type:       events.TypeSecurityWAF,
					SiteID:     "",
					OccurredAt: now.Format(time.RFC3339),
					Details: map[string]any{
						"path":   "/api/events",
						"host":   "localhost",
						"status": 403,
					},
				},
			},
		},
		&fakeDashboardRequestCollector{
			items: []map[string]any{
				{
					"ingested_at": now.Format(time.RFC3339),
					"entry": map[string]any{
						"timestamp": now.Format(time.RFC3339),
						"site":      "",
						"host":      "localhost",
						"uri":       "/api/requests",
						"status":    200,
						"client_ip": "127.0.0.1",
					},
				},
				{
					"ingested_at": now.Format(time.RFC3339),
					"entry": map[string]any{
						"timestamp": now.Format(time.RFC3339),
						"site":      "site-a",
						"host":      "shop.example.com",
						"uri":       "/checkout",
						"status":    200,
						"client_ip": "203.0.113.10",
					},
				},
			},
		},
		&fakeDashboardRuntimeProbe{},
	)

	stats, err := service.Stats()
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}
	if stats.RequestsDay != 1 {
		t.Fatalf("expected internal requests to be skipped, got %d", stats.RequestsDay)
	}
	if stats.AttacksDay != 0 {
		t.Fatalf("expected internal security events to be skipped, got %d", stats.AttacksDay)
	}
}

func TestDashboardService_SkipsAdminAppTrafficOnPublicHost(t *testing.T) {
	now := time.Now().UTC()
	service := NewDashboardService(
		&fakeDashboardEventReader{
			items: []events.Event{
				{
					ID:         "evt-admin-host",
					Type:       events.TypeSecurityAccess,
					SiteID:     "waf_hantico_ru",
					OccurredAt: now.Format(time.RFC3339),
					Details: map[string]any{
						"path":      "/api/events",
						"host":      "waf.hantico.ru",
						"status":    403,
						"client_ip": "46.159.189.39",
					},
				},
				{
					ID:         "evt-site",
					Type:       events.TypeSecurityAccess,
					SiteID:     "shop_example_com",
					OccurredAt: now.Format(time.RFC3339),
					Details: map[string]any{
						"path":      "/checkout",
						"host":      "shop.example.com",
						"status":    403,
						"client_ip": "198.51.100.7",
					},
				},
			},
		},
		&fakeDashboardRequestCollector{
			items: []map[string]any{
				{
					"ingested_at": now.Format(time.RFC3339),
					"entry": map[string]any{
						"timestamp": now.Format(time.RFC3339),
						"site":      "waf_hantico_ru",
						"host":      "waf.hantico.ru",
						"uri":       "/api/dashboard/stats",
						"status":    200,
						"client_ip": "46.159.189.39",
					},
				},
				{
					"ingested_at": now.Format(time.RFC3339),
					"entry": map[string]any{
						"timestamp": now.Format(time.RFC3339),
						"site":      "shop_example_com",
						"host":      "shop.example.com",
						"uri":       "/checkout",
						"status":    200,
						"client_ip": "198.51.100.7",
					},
				},
			},
		},
		&fakeDashboardRuntimeProbe{},
	)

	stats, err := service.Stats()
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}
	if stats.RequestsDay != 1 {
		t.Fatalf("expected admin app requests on public host to be skipped, got %d", stats.RequestsDay)
	}
	if stats.AttacksDay != 1 {
		t.Fatalf("expected only real site attack to remain, got %d", stats.AttacksDay)
	}
	if len(stats.TopAttackerIPs) == 0 || stats.TopAttackerIPs[0].Key != "198.51.100.7" {
		t.Fatalf("expected admin host event to be skipped from attacker IPs, got %#v", stats.TopAttackerIPs)
	}
}
