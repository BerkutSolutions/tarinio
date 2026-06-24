package services

import (
	"net/url"
	"strings"
	"sync"
	"time"

	"waf/control-plane/internal/events"
)

type dashboardEventReader interface {
	List() ([]events.Event, error)
}

type dashboardEventProber interface {
	Probe() error
}

type DashboardService struct {
	events       dashboardEventReader
	requests     RuntimeRequestCollector
	runtimeReady dashboardEventProber
}

type DashboardServiceStatus struct {
	Name      string `json:"name"`
	Up        bool   `json:"up"`
	CheckedAt string `json:"checked_at"`
}

type DashboardTimeCount struct {
	Label     string `json:"label"`
	Timestamp string `json:"timestamp"`
	Count     int    `json:"count"`
}

type DashboardKeyCount struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}

type DashboardProcessStats struct {
	PID            int     `json:"pid"`
	Name           string  `json:"name"`
	Command        string  `json:"command,omitempty"`
	State          string  `json:"state,omitempty"`
	Threads        int     `json:"threads"`
	CPUPercent     float64 `json:"cpu_percent"`
	MemoryRSSBytes uint64  `json:"memory_rss_bytes"`
	MemoryPercent  float64 `json:"memory_percent"`
}

type DashboardSystemStats struct {
	CPUCores           int                     `json:"cpu_cores"`
	CPULoadPercent     float64                 `json:"cpu_load_percent"`
	MemoryTotalBytes   uint64                  `json:"memory_total_bytes"`
	MemoryUsedBytes    uint64                  `json:"memory_used_bytes"`
	MemoryFreeBytes    uint64                  `json:"memory_free_bytes"`
	MemoryUsedPercent  float64                 `json:"memory_used_percent"`
	Goroutines         int                     `json:"goroutines"`
	ControlPlaneHeapMB uint64                  `json:"control_plane_heap_mb"`
	TopCPUProcesses    []DashboardProcessStats `json:"top_cpu_processes"`
	TopMemoryProcesses []DashboardProcessStats `json:"top_memory_processes"`
}

type DashboardStats struct {
	GeneratedAt            string                   `json:"generated_at"`
	Services               []DashboardServiceStatus `json:"services"`
	ServicesUp             int                      `json:"services_up"`
	ServicesDown           int                      `json:"services_down"`
	RequestsDay            int                      `json:"requests_day"`
	RequestsSeries         []DashboardTimeCount     `json:"requests_series"`
	BlockedSeries          []DashboardTimeCount     `json:"blocked_series"`
	AttacksDay             int                      `json:"attacks_day"`
	BlockedAttacksDay      int                      `json:"blocked_attacks_day"`
	UniqueAttackerIPsDay   int                      `json:"unique_attacker_ips_day"`
	PopularErrors          []DashboardKeyCount      `json:"popular_errors"`
	TopAttackerIPs         []DashboardKeyCount      `json:"top_attacker_ips"`
	TopAttackerCountries   []DashboardKeyCount      `json:"top_attacker_countries"`
	MostAttackedURLs       []DashboardKeyCount      `json:"most_attacked_urls"`
	System                 DashboardSystemStats     `json:"system"`
	ObservationWindowHours int                      `json:"observation_window_hours"`
}

type cpuTimesSample struct {
	idle     uint64
	total    uint64
	captured time.Time
}

type requestAttackSummary struct {
	AttacksDay        int
	BlockedAttacksDay int
	UniqueIPsDay      int
	TopIPs            []DashboardKeyCount
	TopCountries      []DashboardKeyCount
	TopURLs           []DashboardKeyCount
}

type processSample struct {
	pid        int
	name       string
	command    string
	state      string
	threads    int
	rssBytes   uint64
	cpuJiffies uint64
}

var tarinioAdminExactPaths = []string{
	"/",
	"/login",
	"/login/2fa",
	"/challenge",
	"/challenge/verify",
}

var tarinioAdminPrefixPaths = []string{
	"/static/",
	"/api/app/",
	"/api/auth/",
	"/api/dashboard/",
	"/api/reports/",
	"/api/sites",
	"/api/upstreams",
	"/api/certificates",
	"/api/tls-configs",
	"/api/easy-site-profiles",
	"/api/access-policies",
	"/api/requests",
	"/api/revisions",
	"/api/events",
	"/api/bans",
	"/api/jobs",
	"/api/settings",
	"/api/administration",
}

var tarinioAdminSegmentPrefixes = []string{
	"/dashboard",
	"/sites",
	"/services",
	"/anti-ddos",
	"/tls",
	"/requests",
	"/revisions",
	"/events",
	"/bans",
	"/jobs",
	"/administration",
	"/activity",
	"/settings",
	"/about",
	"/profile",
	"/healthcheck",
	"/onboarding",
}

var cpuUsageState struct {
	mu          sync.Mutex
	lastSample  cpuTimesSample
	hasSample   bool
	lastPercent float64
}

func NewDashboardService(events dashboardEventReader, requests RuntimeRequestCollector, runtimeReady dashboardEventProber) *DashboardService {
	return &DashboardService{
		events:       events,
		requests:     requests,
		runtimeReady: runtimeReady,
	}
}

func (s *DashboardService) Stats() (DashboardStats, error) {
	now := time.Now().UTC()
	cutoff := now.Add(-24 * time.Hour)

	out := DashboardStats{
		GeneratedAt:            now.Format(time.RFC3339Nano),
		ObservationWindowHours: 24,
	}

	out.Services = s.collectServiceStatus(now)
	for _, item := range out.Services {
		if item.Up {
			out.ServicesUp++
		} else {
			out.ServicesDown++
		}
	}

	requestRows, err := s.collectRequests()
	if err != nil {
		return DashboardStats{}, err
	}
	requestsDay, requestsSeries, blockedFromRequestsSeries, blockedFromRequestsDay, popularErrors := summarizeRequests(requestRows, cutoff, now)
	out.RequestsDay = requestsDay
	out.RequestsSeries = requestsSeries
	out.BlockedSeries = blockedFromRequestsSeries
	out.PopularErrors = popularErrors

	eventItems, err := s.collectEvents()
	if err != nil {
		return DashboardStats{}, err
	}
	attacksDay, blockedAttacksDay, uniqueIPsDay, topIPs, topCountries, topURLs, blockedFromEventsSeries, eventErrors := summarizeAttackEvents(eventItems, cutoff, now)
	out.AttacksDay = attacksDay
	out.BlockedAttacksDay = blockedAttacksDay
	out.UniqueAttackerIPsDay = uniqueIPsDay
	out.TopAttackerIPs = topIPs
	out.TopAttackerCountries = topCountries
	out.MostAttackedURLs = topURLs
	requestAttackFallback := summarizeRequestAttacks(requestRows, cutoff)
	if out.AttacksDay <= 0 && requestAttackFallback.AttacksDay > 0 {
		out.AttacksDay = requestAttackFallback.AttacksDay
	}
	if out.BlockedAttacksDay <= 0 && requestAttackFallback.BlockedAttacksDay > 0 {
		out.BlockedAttacksDay = requestAttackFallback.BlockedAttacksDay
	}
	if out.UniqueAttackerIPsDay <= 0 && requestAttackFallback.UniqueIPsDay > 0 {
		out.UniqueAttackerIPsDay = requestAttackFallback.UniqueIPsDay
	}
	if len(out.TopAttackerIPs) == 0 {
		out.TopAttackerIPs = requestAttackFallback.TopIPs
	}
	if len(out.TopAttackerCountries) == 0 {
		out.TopAttackerCountries = requestAttackFallback.TopCountries
	}
	if len(out.MostAttackedURLs) == 0 {
		out.MostAttackedURLs = requestAttackFallback.TopURLs
	}
	out.BlockedSeries = blockedFromEventsSeries
	if len(out.BlockedSeries) == 0 {
		out.BlockedSeries = blockedFromRequestsSeries
	} else if out.BlockedAttacksDay <= 0 {
		out.BlockedSeries = mergeSeriesMax(out.BlockedSeries, blockedFromRequestsSeries)
	}
	if out.BlockedAttacksDay <= 0 && blockedFromRequestsDay > 0 {
		out.BlockedAttacksDay = blockedFromRequestsDay
	}
	out.PopularErrors = mergeKeyCountsSum(out.PopularErrors, eventErrors, 7)
	out.System = collectSystemStats()
	return out, nil
}

func (s *DashboardService) Probe(kind string, query url.Values) error {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "", "stats", "dashboard":
		_ = s.collectServiceStatus(time.Now().UTC())
		return nil
	case "requests":
		if prober, ok := s.requests.(RuntimeRequestProber); ok {
			if err := prober.Probe(query); err != nil {
				// Keep the dashboard healthcheck usable even when request
				// telemetry storage is temporarily degraded.
				return nil
			}
		}
		return nil
	case "events":
		if prober, ok := s.events.(dashboardEventProber); ok {
			return prober.Probe()
		}
		return nil
	default:
		return nil
	}
}

func (s *DashboardService) collectServiceStatus(now time.Time) []DashboardServiceStatus {
	checkedAt := now.Format(time.RFC3339Nano)
	statuses := []DashboardServiceStatus{
		{Name: "control-plane", Up: true, CheckedAt: checkedAt},
	}
	if s.runtimeReady != nil {
		statuses = append(statuses, DashboardServiceStatus{
			Name:      "runtime",
			Up:        s.runtimeReady.Probe() == nil,
			CheckedAt: checkedAt,
		})
	}
	return statuses
}

func (s *DashboardService) collectRequests() ([]map[string]any, error) {
	if s.requests == nil {
		return []map[string]any{}, nil
	}
	items, err := s.requests.Collect()
	if err != nil {
		// Dashboard must stay available even when runtime request telemetry
		// is temporarily unavailable during startup or reload.
		return []map[string]any{}, nil
	}
	return items, nil
}

func (s *DashboardService) collectEvents() ([]events.Event, error) {
	if s.events == nil {
		return []events.Event{}, nil
	}
	return s.events.List()
}

// moved to dashboard_attack_summaries.go and dashboard_attack_filters.go

// moved to dashboard_system_stats.go and dashboard_process_stats.go

// moved to dashboard_parse_helpers.go

// moved to dashboard_process_stats.go
