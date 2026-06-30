package services

import (
	"fmt"
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

type runtimeRequestCollectorWithOptions interface {
	CollectWithOptions(query url.Values) ([]map[string]any, error)
}

type runtimeRequestCounter interface {
	CollectCount(query url.Values) (int, error)
}

// dashboardContainerIssueReader — интерфейс для получения ошибок из логов контейнеров.
type dashboardContainerIssueReader interface {
	Issues() (DashboardContainerIssuesSummary, error)
}

type DashboardService struct {
	events          dashboardEventReader
	requests        RuntimeRequestCollector
	runtimeReady    dashboardEventProber
	containers      dashboardContainerIssueReader
	sampler         dashboardSnapshotSampler
	requestsCache struct {
		mu   sync.Mutex
		rows []map[string]any
	}
	eventsCache struct {
		mu    sync.Mutex
		items []events.Event
	}
	// dismissedServiceErrors хранит ID ошибок сервисов, которые пользователь скрыл.
	// Хранится in-memory (сбрасывается при перезапуске — это нормально для этого кейса).
	dismissedServiceErrors struct {
		mu   sync.RWMutex
		ids  map[string]struct{}
	}
}

type DashboardServiceStatus struct {
	Name           string                      `json:"name"`
	Up             bool                        `json:"up"`
	CheckedAt      string                      `json:"checked_at"`
	UpstreamErrors []DashboardServiceError     `json:"upstream_errors,omitempty"`
	HasErrors      bool                        `json:"has_errors"`
}

// DashboardServiceError — одна ошибка upstream для сервиса (из nginx логов runtime).
type DashboardServiceError struct {
	ID            string `json:"id"`
	Message       string `json:"message"`
	NormalizedMsg string `json:"normalized_msg"`
	Count         int    `json:"count"`
	FirstSeen     string `json:"first_seen,omitempty"`
	LastSeen      string `json:"last_seen,omitempty"`
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
	RequestUniqueIPsDay    int                      `json:"request_unique_ips_day"`
	RequestTopSites        []DashboardKeyCount      `json:"request_top_sites"`
	RequestTopURLs         []DashboardKeyCount      `json:"request_top_urls"`
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
	UpstreamHealth         []DashboardUpstreamHealth `json:"upstream_health,omitempty"`
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

var processState struct {
	mu          sync.Mutex
	lastSamples map[int]processSample
	lastTotal   uint64
	hasSample   bool
}

func NewDashboardService(events dashboardEventReader, requests RuntimeRequestCollector, runtimeReady dashboardEventProber) *DashboardService {
	s := &DashboardService{
		events:       events,
		requests:     requests,
		runtimeReady: runtimeReady,
		sampler: dashboardSnapshotSampler{
			interval: 15 * time.Second,
		},
	}
	s.dismissedServiceErrors.ids = make(map[string]struct{})
	return s
}

// WithContainerIssueReader подключает сервис чтения ошибок контейнеров.
// Вызывается из app.go после создания DashboardService.
func (s *DashboardService) WithContainerIssueReader(c dashboardContainerIssueReader) {
	if s == nil {
		return
	}
	s.containers = c
}

// DismissServiceErrors помечает ошибки сервиса как скрытые по их ID.
func (s *DashboardService) DismissServiceErrors(errorIDs []string) {
	if s == nil || len(errorIDs) == 0 {
		return
	}
	s.dismissedServiceErrors.mu.Lock()
	defer s.dismissedServiceErrors.mu.Unlock()
	for _, id := range errorIDs {
		if id = strings.TrimSpace(id); id != "" {
			s.dismissedServiceErrors.ids[id] = struct{}{}
		}
	}
}

// isDismissed проверяет скрыта ли ошибка пользователем.
func (s *DashboardService) isDismissed(id string) bool {
	s.dismissedServiceErrors.mu.RLock()
	defer s.dismissedServiceErrors.mu.RUnlock()
	_, ok := s.dismissedServiceErrors.ids[id]
	return ok
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

	// Собираем upstream ошибки из логов runtime контейнера и
	// привязываем к сайтам по имени хоста в сообщении.
	upstreamErrsByHost := s.collectUpstreamErrors()
	for i, st := range statuses {
		errs, ok := upstreamErrsByHost[st.Name]
		if !ok {
			continue
		}
		statuses[i].UpstreamErrors = errs
		statuses[i].HasErrors = len(errs) > 0
	}

	// Добавляем сервисы из upstream errors у которых нет явного статуса.
	for host, errs := range upstreamErrsByHost {
		found := false
		for _, st := range statuses {
			if st.Name == host {
				found = true
				break
			}
		}
		if found {
			continue
		}
		statuses = append(statuses, DashboardServiceStatus{
			Name:           host,
			Up:             true, // upstream ошибки не означают что сервис down
			CheckedAt:      checkedAt,
			UpstreamErrors: errs,
			HasErrors:      len(errs) > 0,
		})
	}

	return statuses
}

// collectUpstreamErrors читает Issues из runtime контейнера и фильтрует
// nginx upstream timed out / connection refused ошибки, группируя их по хосту сервиса.
func (s *DashboardService) collectUpstreamErrors() map[string][]DashboardServiceError {
	if s.containers == nil {
		return nil
	}
	summary, err := s.containers.Issues()
	if err != nil {
		return nil
	}

	out := map[string][]DashboardServiceError{}
	for _, issue := range summary.Issues {
		// Интересуют только upstream ошибки nginx из runtime контейнера.
		if issue.Container != "tarinio-runtime" && issue.Container != "waf-e2e-runtime" {
			continue
		}
		lower := strings.ToLower(issue.NormalizedLog)
		if !strings.Contains(lower, "upstream timed out") &&
			!strings.Contains(lower, "upstream prematurely closed") &&
			!strings.Contains(lower, "no live upstreams while connecting") &&
			!strings.Contains(lower, "connect() failed") {
			continue
		}

		// Извлекаем hostname сервера из поля "server: <host>" в nginx лог строке.
		host := extractNginxServerHost(issue.SampleLog)
		if host == "" {
			host = extractNginxServerHost(issue.NormalizedLog)
		}
		if host == "" {
			continue
		}

		id := fmt.Sprintf("%s|%s|%s", issue.Container, issue.Severity, issue.NormalizedLog)
		// ID детерминирован — используем для dismiss проверки.
		errorID := shortHashID(id)

		if s.isDismissed(errorID) {
			continue
		}

		out[host] = append(out[host], DashboardServiceError{
			ID:            errorID,
			Message:       issue.SampleLog,
			NormalizedMsg: issue.NormalizedLog,
			Count:         issue.Count,
			FirstSeen:     issue.FirstSeen,
			LastSeen:      issue.LastSeen,
		})
	}
	return out
}

// extractNginxServerHost извлекает значение поля "server: <host>" из строки nginx лога.
// Пример: "... server: n8n.hantico.com, request: ..." → "n8n.hantico.com"
func extractNginxServerHost(message string) string {
	const marker = "server: "
	idx := strings.Index(strings.ToLower(message), marker)
	if idx < 0 {
		return ""
	}
	rest := message[idx+len(marker):]
	end := strings.IndexAny(rest, ", 	\n")
	if end < 0 {
		end = len(rest)
	}
	host := strings.TrimSpace(rest[:end])
	// Убираем порт если есть.
	if colonIdx := strings.LastIndex(host, ":"); colonIdx > 0 {
		// Только если это не IPv6.
		if !strings.Contains(host[:colonIdx], ":") {
			host = host[:colonIdx]
		}
	}
	return host
}

// shortHashID возвращает короткий детерминированный ID строки (6 hex символов).
func shortHashID(s string) string {
	h := fnv32a(s)
	return fmt.Sprintf("%08x", h)
}

func fnv32a(s string) uint32 {
	var h uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}

func (s *DashboardService) collectRequestsDay() (int, bool) {
	if s.requests == nil {
		return 0, false
	}
	counter, ok := s.requests.(runtimeRequestCounter)
	if !ok {
		return 0, false
	}
	q := make(url.Values)
	since := time.Now().UTC().Add(-24 * time.Hour)
	q.Set("since", since.Format(time.RFC3339))
	n, err := counter.CollectCount(q)
	if err != nil {
		return 0, false
	}
	return n, true
}

func (s *DashboardService) collectRequests() ([]map[string]any, error) {
	if s.requests == nil {
		return []map[string]any{}, nil
	}
	// Request only the last 25 hours from the storage backend so that
	// OpenSearch/ClickHouse can apply a server-side time filter instead of
	// returning up to maxItems rows and relying on in-process filtering.
	// The extra hour gives a small buffer for clock skew and ensures we never
	// under-count at the boundary of the 24-hour observation window.
	var items []map[string]any
	var err error
	if withOpts, ok := s.requests.(runtimeRequestCollectorWithOptions); ok {
		q := make(url.Values)
		since := time.Now().UTC().Add(-25 * time.Hour)
		q.Set("since", since.Format(time.RFC3339))
		items, err = withOpts.CollectWithOptions(q)
	} else {
		items, err = s.requests.Collect()
	}
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
