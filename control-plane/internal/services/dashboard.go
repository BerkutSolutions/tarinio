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
	sampler      dashboardSnapshotSampler
	requestsCache struct {
		mu   sync.Mutex
		rows []map[string]any
	}
	eventsCache struct {
		mu    sync.Mutex
		items []events.Event
	}
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
		sampler: dashboardSnapshotSampler{
			interval: 15 * time.Second,
		},
	}
}

func (s *DashboardService) Stats() (DashboardStats, error) {
	if cached, ok := s.snapshot(); ok {
		return cached, nil
	}
	stats, err := s.buildSnapshot()
	if err != nil {
		return DashboardStats{}, err
	}
	s.storeSnapshot(stats)
	return stats, nil
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
		s.requestsCache.mu.Lock()
		defer s.requestsCache.mu.Unlock()
		if len(s.requestsCache.rows) > 0 {
			return append([]map[string]any(nil), s.requestsCache.rows...), nil
		}
		// Dashboard must stay available even when runtime request telemetry
		// is temporarily unavailable during startup or reload.
		return []map[string]any{}, nil
	}
	s.requestsCache.mu.Lock()
	s.requestsCache.rows = append([]map[string]any(nil), items...)
	s.requestsCache.mu.Unlock()
	return items, nil
}

func (s *DashboardService) collectEvents() ([]events.Event, error) {
	if s.events == nil {
		return []events.Event{}, nil
	}
	items, err := s.events.List()
	if err != nil {
		s.eventsCache.mu.Lock()
		defer s.eventsCache.mu.Unlock()
		if len(s.eventsCache.items) > 0 {
			return append([]events.Event(nil), s.eventsCache.items...), nil
		}
		return []events.Event{}, nil
	}
	s.eventsCache.mu.Lock()
	s.eventsCache.items = append([]events.Event(nil), items...)
	s.eventsCache.mu.Unlock()
	return items, nil
}

// moved to dashboard_attack_summaries.go and dashboard_attack_filters.go

// moved to dashboard_system_stats.go and dashboard_process_stats.go

// moved to dashboard_parse_helpers.go

// moved to dashboard_process_stats.go
