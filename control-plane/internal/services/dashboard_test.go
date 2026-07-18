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

func TestDashboardService_DismissalsAreActorScoped(t *testing.T) {
	service := NewDashboardService(&fakeDashboardEventReader{}, nil, nil)
	stats := DashboardStats{Services: []DashboardServiceStatus{{
		Name: "runtime",
		UpstreamErrors: []DashboardServiceError{
			{ID: "shared-error", Message: "upstream timed out"},
		},
		HasErrors: true,
	}}}

	service.DismissServiceErrors("operator-a", []string{"shared-error"})
	for _, check := range []struct {
		actor string
		want  int
	}{
		{actor: "operator-a", want: 0},
		{actor: "operator-b", want: 1},
		{actor: "", want: 1},
	} {
		got := service.filterDismissedServiceErrors(stats, check.actor)
		if actual := len(got.Services[0].UpstreamErrors); actual != check.want {
			t.Fatalf("actor %q: expected %d visible errors, got %d", check.actor, check.want, actual)
		}
	}
	if !stats.Services[0].HasErrors || len(stats.Services[0].UpstreamErrors) != 1 {
		t.Fatal("filter must not mutate the shared dashboard snapshot")
	}
}

func TestDashboardService_StatsForActorWithoutProcessDetailsKeepsContractButRedactsInventory(t *testing.T) {
	service := NewDashboardService(&fakeDashboardEventReader{}, nil, nil)
	service.storeSnapshot(DashboardStats{System: DashboardSystemStats{
		TopCPUProcesses:    []DashboardProcessStats{{PID: 1, Name: "init", Command: "init --secret"}},
		TopMemoryProcesses: []DashboardProcessStats{{PID: 2, Name: "worker", Command: "worker --token"}},
	}})

	redacted, err := service.StatsForActorWithProcessDetails("viewer", false)
	if err != nil {
		t.Fatalf("redacted stats: %v", err)
	}
	if len(redacted.System.TopCPUProcesses) != 0 || len(redacted.System.TopMemoryProcesses) != 0 {
		t.Fatal("non-administrative dashboard response exposed host-process inventory")
	}
	privileged, err := service.StatsForActorWithProcessDetails("admin", true)
	if err != nil {
		t.Fatalf("privileged stats: %v", err)
	}
	if len(privileged.System.TopCPUProcesses) != 1 || len(privileged.System.TopMemoryProcesses) != 1 {
		t.Fatal("administrative dashboard response lost process inventory")
	}
}

type fakeDashboardRequestCollector struct {
	items []map[string]any
	err   error
	count int
}

func (f *fakeDashboardRequestCollector) Collect() ([]map[string]any, error) {
	return append([]map[string]any(nil), f.items...), f.err
}

func (f *fakeDashboardRequestCollector) CollectWithOptions(_ url.Values) ([]map[string]any, error) {
	return append([]map[string]any(nil), f.items...), f.err
}

func (f *fakeDashboardRequestCollector) CollectCount(_ url.Values) (int, error) {
	if f.err != nil {
		return 0, f.err
	}
	if f.count > 0 {
		return f.count, nil
	}
	return len(f.items), nil
}

func (f *fakeDashboardRequestCollector) Probe(_ url.Values) error { return f.err }

type fakeDashboardRuntimeProbe struct{ err error }

func (f *fakeDashboardRuntimeProbe) Probe() error { return f.err }

type fakeDashboardSummaryCollector struct {
	fakeDashboardRequestCollector
	summary RuntimeRequestDashboardSummary
}

func (f *fakeDashboardSummaryCollector) CollectDashboardSummary(_ url.Values) (RuntimeRequestDashboardSummary, error) {
	return f.summary, f.err
}

func TestDashboardService_RuntimeSummaryIsAuthoritativeForAttackWidgets(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Hour)
	collector := &fakeDashboardSummaryCollector{summary: RuntimeRequestDashboardSummary{
		RequestsDay: 7, BlockedDay: 2, AttacksDay: 2, UniqueAttackerIPs: 1,
		RequestsSeries:   []DashboardTimeCount{{Timestamp: now.Format(time.RFC3339), Count: 7}},
		BlockedSeries:    []DashboardTimeCount{{Timestamp: now.Format(time.RFC3339), Count: 2}},
		TopAttackerIPs:   []DashboardKeyCount{{Key: "203.0.113.20", Count: 2, Country: "DE"}},
		TopCountries:     []DashboardKeyCount{{Key: "DE", Count: 2}},
		MostAttackedURLs: []DashboardKeyCount{{Key: "/blocked", Count: 2}},
		PopularErrors:    []DashboardKeyCount{{Key: "403", Count: 1}, {Key: "429", Count: 1}},
	}}
	eventsReader := &fakeDashboardEventReader{items: []events.Event{{
		Type: events.TypeSecurityWAF, OccurredAt: now.Format(time.RFC3339),
		Details: map[string]any{"client_ip": "198.51.100.99", "country": "RU", "path": "/other", "status": 403},
	}}}
	service := NewDashboardService(eventsReader, collector, nil)
	stats, err := service.Stats()
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if stats.AttacksDay != 2 || stats.BlockedAttacksDay != 2 || stats.UniqueAttackerIPsDay != 1 {
		t.Fatalf("summary totals were mixed with events: %+v", stats)
	}
	if len(stats.TopAttackerIPs) != 1 || stats.TopAttackerIPs[0].Country != "DE" {
		t.Fatalf("summary IP metadata was not preserved: %#v", stats.TopAttackerIPs)
	}
}

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
	if stats.RequestUniqueIPsDay != 2 {
		t.Fatalf("expected 2 unique request IPs, got %d", stats.RequestUniqueIPsDay)
	}
	if len(stats.RequestTopSites) == 0 || stats.RequestTopSites[0].Key != "site-a" {
		t.Fatalf("expected request top sites to be populated, got %#v", stats.RequestTopSites)
	}
	if len(stats.RequestTopURLs) == 0 || stats.RequestTopURLs[0].Key != "/checkout" {
		t.Fatalf("expected request top urls to be populated, got %#v", stats.RequestTopURLs)
	}
	if stats.AttacksDay != 1 {
		t.Fatalf("expected 1 attack, got %d", stats.AttacksDay)
	}
	if len(stats.AttacksSeries) != 24 {
		t.Fatalf("expected 24 hourly attack buckets, got %#v", stats.AttacksSeries)
	}
	attackSeriesTotal := 0
	for _, bucket := range stats.AttacksSeries {
		attackSeriesTotal += bucket.Count
	}
	if attackSeriesTotal != stats.AttacksDay {
		t.Fatalf("hourly attack series (%d) must match the authoritative daily count (%d): %#v", attackSeriesTotal, stats.AttacksDay, stats.AttacksSeries)
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

func TestDashboardService_UsesFilteredRowsForEveryRequestWidget(t *testing.T) {
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
				},
			},
		},
		count: 48780,
	}

	service := NewDashboardService(&fakeDashboardEventReader{}, requests, &fakeDashboardRuntimeProbe{})
	stats, err := service.Stats()
	if err != nil {
		t.Fatalf("stats failed: %v", err)
	}

	if stats.RequestsDay != 1 {
		t.Fatalf("expected requests_day to match the filtered request rows, got %d", stats.RequestsDay)
	}
	if len(stats.RequestTopSites) != 1 || stats.RequestTopSites[0].Count != 1 {
		t.Fatalf("expected sampled top-site breakdown to stay available, got %#v", stats.RequestTopSites)
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

func TestDashboardService_RuntimeRequestsProbeFailureDoesNotFailProbe(t *testing.T) {
	service := NewDashboardService(
		&fakeDashboardEventReader{},
		&fakeDashboardRequestCollector{err: errors.New("runtime requests unavailable")},
		&fakeDashboardRuntimeProbe{},
	)

	query := url.Values{}
	query.Set("day", "2026-06-22")
	query.Set("probe", "requests")

	if err := service.Probe("requests", query); err != nil {
		t.Fatalf("expected requests probe failure to be tolerated, got %v", err)
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

func TestDashboardService_DoesNotOverrideEventAttacksWithBlockedRequestFallback(t *testing.T) {
	now := time.Now().UTC()
	service := NewDashboardService(
		&fakeDashboardEventReader{
			items: []events.Event{
				{
					ID:         "evt-attack",
					Type:       events.TypeSecurityWAF,
					SiteID:     "site-a",
					OccurredAt: now.Format(time.RFC3339),
					Details: map[string]any{
						"blocked":   true,
						"status":    403,
						"client_ip": "203.0.113.77",
						"path":      "/checkout",
						"country":   "RU",
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
						"site":      "site-a",
						"uri":       "/challenge",
						"status":    403,
						"client_ip": "198.51.100.20",
						"country":   "DE",
					},
				},
				{
					"ingested_at": now.Add(-time.Minute).Format(time.RFC3339),
					"entry": map[string]any{
						"timestamp": now.Add(-time.Minute).Format(time.RFC3339),
						"site":      "site-a",
						"uri":       "/challenge",
						"status":    403,
						"client_ip": "198.51.100.21",
						"country":   "FR",
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
	if stats.AttacksDay != 1 {
		t.Fatalf("expected event-derived attacks to win over request fallback, got %d", stats.AttacksDay)
	}
	if stats.BlockedAttacksDay != 1 {
		t.Fatalf("expected event-derived blocked attacks to win over request fallback, got %d", stats.BlockedAttacksDay)
	}
	if len(stats.TopAttackerIPs) == 0 || stats.TopAttackerIPs[0].Key != "203.0.113.77" {
		t.Fatalf("expected event-derived top attacker IP to remain primary, got %#v", stats.TopAttackerIPs)
	}
	if len(stats.TopAttackerCountries) != 1 || stats.TopAttackerCountries[0].Key != "RU" {
		t.Fatalf("expected event-only countries to remain unchanged when event data is already complete, got %#v", stats.TopAttackerCountries)
	}
	if len(stats.MostAttackedURLs) != 1 || stats.MostAttackedURLs[0].Key != "/checkout" {
		t.Fatalf("expected event-only attacked URLs to remain unchanged when event data is already complete, got %#v", stats.MostAttackedURLs)
	}
}

func TestDashboardService_BackfillsMissingEventAttackBreakdownsFromRequests(t *testing.T) {
	now := time.Now().UTC()
	service := NewDashboardService(
		&fakeDashboardEventReader{
			items: []events.Event{
				{
					ID:         "evt-attack",
					Type:       events.TypeSecurityWAF,
					SiteID:     "site-a",
					OccurredAt: now.Format(time.RFC3339),
					Details: map[string]any{
						"blocked":   true,
						"status":    403,
						"client_ip": "203.0.113.77",
						"path":      "/checkout",
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
						"site":      "site-a",
						"uri":       "/challenge",
						"status":    403,
						"client_ip": "198.51.100.20",
						"country":   "DE",
					},
				},
				{
					"ingested_at": now.Add(-time.Minute).Format(time.RFC3339),
					"entry": map[string]any{
						"timestamp": now.Add(-time.Minute).Format(time.RFC3339),
						"site":      "site-a",
						"uri":       "/challenge",
						"status":    403,
						"client_ip": "198.51.100.21",
						"country":   "FR",
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
	if stats.AttacksDay != 1 {
		t.Fatalf("expected event attack count to remain authoritative when events exist, got %d", stats.AttacksDay)
	}
	if stats.BlockedAttacksDay != 1 {
		t.Fatalf("expected blocked attack count to remain authoritative when events exist, got %d", stats.BlockedAttacksDay)
	}
	if stats.UniqueAttackerIPsDay != 1 {
		t.Fatalf("expected unique attacker IP count to remain authoritative when events exist, got %d", stats.UniqueAttackerIPsDay)
	}
	if len(stats.TopAttackerIPs) == 0 || stats.TopAttackerIPs[0].Key != "203.0.113.77" {
		t.Fatalf("expected event-derived top attacker IP to remain primary, got %#v", stats.TopAttackerIPs)
	}
	if len(stats.TopAttackerCountries) != 0 {
		t.Fatalf("expected countries to stay empty when events have no country and request fallback is unavailable, got %#v", stats.TopAttackerCountries)
	}
	if len(stats.MostAttackedURLs) != 1 || stats.MostAttackedURLs[0].Key != "/checkout" {
		t.Fatalf("expected attacked URLs to remain event-derived when events are authoritative, got %#v", stats.MostAttackedURLs)
	}
}

func TestDashboardService_BackfillsCountriesAndURLsWhenEventCountsExistButEventBreakdownsArePartial(t *testing.T) {
	now := time.Now().UTC()
	service := NewDashboardService(
		&fakeDashboardEventReader{
			items: []events.Event{
				{
					ID:         "evt-attack",
					Type:       events.TypeSecurityWAF,
					SiteID:     "site-a",
					OccurredAt: now.Format(time.RFC3339),
					Details: map[string]any{
						"blocked":   true,
						"status":    403,
						"client_ip": "203.0.113.77",
						"path":      "/checkout",
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
						"site":      "site-a",
						"uri":       "/challenge",
						"status":    403,
						"client_ip": "198.51.100.20",
						"country":   "DE",
					},
				},
				{
					"ingested_at": now.Add(-time.Minute).Format(time.RFC3339),
					"entry": map[string]any{
						"timestamp": now.Add(-time.Minute).Format(time.RFC3339),
						"site":      "site-a",
						"uri":       "/challenge",
						"status":    403,
						"client_ip": "198.51.100.21",
						"country":   "FR",
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
	if stats.AttacksDay != 1 {
		t.Fatalf("expected event attack count to remain authoritative when partial event breakdown is backfilled, got %d", stats.AttacksDay)
	}
	if stats.BlockedAttacksDay != 1 {
		t.Fatalf("expected blocked attack count to remain authoritative when partial event breakdown is backfilled, got %d", stats.BlockedAttacksDay)
	}
	if stats.UniqueAttackerIPsDay != 1 {
		t.Fatalf("expected unique attacker IP count to remain authoritative when partial event breakdown is backfilled, got %d", stats.UniqueAttackerIPsDay)
	}
	if len(stats.TopAttackerCountries) != 0 {
		t.Fatalf("expected countries to stay empty because request fallback challenge paths are filtered from public dashboard stats, got %#v", stats.TopAttackerCountries)
	}
	if len(stats.MostAttackedURLs) != 1 || stats.MostAttackedURLs[0].Key != "/checkout" {
		t.Fatalf("expected event-derived attacked URL to remain authoritative when request fallback challenge paths are filtered, got %#v", stats.MostAttackedURLs)
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

func TestDashboardService_SkipsChallengeRequestsWithQueryString(t *testing.T) {
	now := time.Now().UTC()
	service := NewDashboardService(
		&fakeDashboardEventReader{},
		&fakeDashboardRequestCollector{
			items: []map[string]any{
				{
					"ingested_at": now.Format(time.RFC3339),
					"entry": map[string]any{
						"timestamp": now.Format(time.RFC3339),
						"site":      "ui_example_test",
						"host":      "ui.example.test",
						"uri":       "/challenge?return_uri=/login&return_args=",
						"status":    200,
						"client_ip": "203.0.113.10",
					},
				},
				{
					"ingested_at": now.Format(time.RFC3339),
					"entry": map[string]any{
						"timestamp": now.Format(time.RFC3339),
						"site":      "site-a",
						"host":      "site-a.example.com",
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
		t.Fatalf("expected challenge requests to be skipped, got %d", stats.RequestsDay)
	}
}
