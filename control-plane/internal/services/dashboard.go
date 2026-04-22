package services

import (
	"bufio"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
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
	if requestAttackFallback.AttacksDay > out.AttacksDay {
		out.AttacksDay = requestAttackFallback.AttacksDay
	}
	if requestAttackFallback.BlockedAttacksDay > out.BlockedAttacksDay {
		out.BlockedAttacksDay = requestAttackFallback.BlockedAttacksDay
	}
	if requestAttackFallback.UniqueIPsDay > out.UniqueAttackerIPsDay {
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
	out.BlockedSeries = mergeSeriesMax(out.BlockedSeries, blockedFromRequestsSeries)
	if blockedFromRequestsDay > out.BlockedAttacksDay {
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
			return prober.Probe(query)
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
		return nil, err
	}
	return items, nil
}

func (s *DashboardService) collectEvents() ([]events.Event, error) {
	if s.events == nil {
		return []events.Event{}, nil
	}
	return s.events.List()
}

func summarizeRequests(items []map[string]any, cutoff, now time.Time) (int, []DashboardTimeCount, []DashboardTimeCount, int, []DashboardKeyCount) {
	var total int
	series := map[time.Time]int{}
	blockedSeries := map[time.Time]int{}
	var blockedTotal int
	errorCounts := map[string]int{}

	for _, row := range items {
		entry, ok := row["entry"].(map[string]any)
		if !ok {
			continue
		}
		uri := strings.TrimSpace(asString(entry["uri"]))
		siteID := strings.TrimSpace(asString(entry["site"]))
		if shouldSkipInternalRequest(uri, siteID) {
			continue
		}
		when := parseAnyTime(entry["timestamp"])
		if when.IsZero() {
			when = parseAnyTime(row["ingested_at"])
		}
		if when.IsZero() || when.Before(cutoff) {
			continue
		}
		total++
		bucket := when.UTC().Truncate(time.Hour)
		series[bucket]++
		statusCode := parseAnyInt(entry["status"])
		if statusCode == 403 || statusCode == 429 || statusCode == 444 {
			blockedTotal++
			blockedSeries[bucket]++
		}
		if statusCode >= 400 && statusCode <= 599 {
			errorCounts[strconv.Itoa(statusCode)]++
		}
	}

	seriesOut := buildHourlySeries(series, now)
	blockedOut := buildHourlySeries(blockedSeries, now)
	return total, seriesOut, blockedOut, blockedTotal, topCounts(errorCounts, 7)
}

func summarizeAttackEvents(items []events.Event, cutoff, now time.Time) (int, int, int, []DashboardKeyCount, []DashboardKeyCount, []DashboardKeyCount, []DashboardTimeCount, []DashboardKeyCount) {
	var attacks int
	var blocked int
	ipCounts := map[string]int{}
	countryCounts := map[string]int{}
	urlCounts := map[string]int{}
	errorCounts := map[string]int{}
	uniqueIPs := map[string]struct{}{}
	blockedByHour := map[time.Time]int{}

	for _, item := range items {
		if item.Type != events.TypeSecurityAccess && item.Type != events.TypeSecurityRateLimit && item.Type != events.TypeSecurityWAF {
			continue
		}
		when := parseAnyTime(item.OccurredAt)
		if when.IsZero() || when.Before(cutoff) {
			continue
		}
		if shouldSkipInternalRequest("", item.SiteID) {
			continue
		}
		// Not every security event should be counted as an "attack" in the dashboard.
		// Runtime may emit warning signals like "burst detected (not blocked)" during normal admin UI activity.
		// If backend explicitly marks the event as not blocked, treat it as an alert and skip it from attack metrics.
		if raw, ok := item.Details["blocked"]; ok {
			if flag, typeOK := raw.(bool); typeOK && !flag {
				continue
			}
		}

		attacks++
		ip := strings.TrimSpace(asString(item.Details["client_ip"]))
		if ip == "" {
			ip = strings.TrimSpace(asString(item.Details["ip"]))
		}
		if ip != "" {
			uniqueIPs[ip] = struct{}{}
			ipCounts[ip]++
		}

		urlPath := strings.TrimSpace(asString(item.Details["path"]))
		if urlPath == "" {
			urlPath = strings.TrimSpace(asString(item.Details["uri"]))
		}
		if urlPath != "" {
			urlCounts[urlPath]++
		}

		country := strings.ToUpper(strings.TrimSpace(asString(item.Details["country"])))
		if country == "" {
			country = strings.ToUpper(strings.TrimSpace(asString(item.Details["client_country"])))
		}
		if country == "" {
			country = "UNKNOWN"
		}
		countryCounts[country]++
		statusCode := parseAnyInt(item.Details["status"])
		if statusCode >= 400 && statusCode <= 599 {
			errorCounts[strconv.Itoa(statusCode)]++
		}

		if isBlockedEvent(item) {
			blocked++
			blockedByHour[when.UTC().Truncate(time.Hour)]++
		}
	}

	return attacks, blocked, len(uniqueIPs), topCounts(ipCounts, 10), topCounts(countryCounts, 10), topCounts(urlCounts, 10), buildHourlySeries(blockedByHour, now), topCounts(errorCounts, 7)
}

func summarizeRequestAttacks(items []map[string]any, cutoff time.Time) requestAttackSummary {
	ipCounts := map[string]int{}
	countryCounts := map[string]int{}
	urlCounts := map[string]int{}
	uniqueIPs := map[string]struct{}{}

	var out requestAttackSummary
	for _, row := range items {
		entry, ok := row["entry"].(map[string]any)
		if !ok {
			continue
		}
		when := parseAnyTime(entry["timestamp"])
		if when.IsZero() {
			when = parseAnyTime(row["ingested_at"])
		}
		if when.IsZero() || when.Before(cutoff) {
			continue
		}
		uri := strings.TrimSpace(asString(entry["uri"]))
		siteID := strings.TrimSpace(asString(entry["site"]))
		if shouldSkipInternalRequest(uri, siteID) {
			continue
		}
		statusCode := parseAnyInt(entry["status"])
		if statusCode != 403 && statusCode != 429 && statusCode != 444 {
			continue
		}

		out.AttacksDay++
		out.BlockedAttacksDay++
		if uri != "" {
			urlCounts[uri]++
		}

		ip := strings.TrimSpace(asString(entry["client_ip"]))
		if ip != "" {
			uniqueIPs[ip] = struct{}{}
			ipCounts[ip]++
		}

		country := strings.ToUpper(strings.TrimSpace(asString(entry["country"])))
		if country == "" {
			country = strings.ToUpper(strings.TrimSpace(asString(entry["client_country"])))
		}
		if country == "" {
			country = "UNKNOWN"
		}
		countryCounts[country]++
	}

	out.UniqueIPsDay = len(uniqueIPs)
	out.TopIPs = topCounts(ipCounts, 10)
	out.TopCountries = topCounts(countryCounts, 10)
	out.TopURLs = topCounts(urlCounts, 10)
	return out
}

func buildHourlySeries(values map[time.Time]int, now time.Time) []DashboardTimeCount {
	end := now.UTC().Truncate(time.Hour)
	start := end.Add(-23 * time.Hour)
	out := make([]DashboardTimeCount, 0, 24)
	for i := 0; i < 24; i++ {
		ts := start.Add(time.Duration(i) * time.Hour)
		out = append(out, DashboardTimeCount{
			Label:     ts.Format("15:04"),
			Timestamp: ts.Format(time.RFC3339),
			Count:     values[ts],
		})
	}
	return out
}

func shouldSkipInternalRequest(uri string, siteID string) bool {
	path := strings.ToLower(strings.TrimSpace(uri))
	site := strings.ToLower(strings.TrimSpace(siteID))
	site = strings.ReplaceAll(site, "_", "-")
	if site == "control-plane-access" || site == "control-plane" || site == "ui" {
		return true
	}
	if path == "" {
		return false
	}
	return strings.HasPrefix(path, "/api/dashboard") ||
		strings.HasPrefix(path, "/dashboard") ||
		strings.HasPrefix(path, "/healthz") ||
		strings.HasPrefix(path, "/readyz") ||
		strings.HasPrefix(path, "/login")
}

func mergeSeriesMax(primary []DashboardTimeCount, secondary []DashboardTimeCount) []DashboardTimeCount {
	if len(primary) == 0 {
		return append([]DashboardTimeCount(nil), secondary...)
	}
	secMap := make(map[string]DashboardTimeCount, len(secondary))
	for _, item := range secondary {
		secMap[item.Timestamp] = item
	}
	out := make([]DashboardTimeCount, 0, len(primary))
	for _, item := range primary {
		other, ok := secMap[item.Timestamp]
		if ok && other.Count > item.Count {
			item.Count = other.Count
		}
		out = append(out, item)
	}
	return out
}

func mergeKeyCountsSum(primary []DashboardKeyCount, secondary []DashboardKeyCount, limit int) []DashboardKeyCount {
	sums := map[string]int{}
	for _, item := range primary {
		key := strings.TrimSpace(item.Key)
		if key == "" {
			continue
		}
		sums[key] += item.Count
	}
	for _, item := range secondary {
		key := strings.TrimSpace(item.Key)
		if key == "" {
			continue
		}
		sums[key] += item.Count
	}
	return topCounts(sums, limit)
}

func isBlockedEvent(item events.Event) bool {
	if blocked, ok := item.Details["blocked"].(bool); ok {
		return blocked
	}
	status := parseAnyInt(item.Details["status"])
	return status == 403 || status == 429 || status == 444
}

func topCounts(values map[string]int, limit int) []DashboardKeyCount {
	out := make([]DashboardKeyCount, 0, len(values))
	for key, count := range values {
		if strings.TrimSpace(key) == "" || count <= 0 {
			continue
		}
		out = append(out, DashboardKeyCount{Key: key, Count: count})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].Key < out[j].Key
		}
		return out[i].Count > out[j].Count
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func collectSystemStats() DashboardSystemStats {
	total, free := readMemoryFromProc()
	used := uint64(0)
	if total >= free {
		used = total - free
	}
	usedPercent := 0.0
	if total > 0 {
		usedPercent = float64(used) * 100 / float64(total)
	}

	memStats := runtime.MemStats{}
	runtime.ReadMemStats(&memStats)
	topCPUProcesses, topMemoryProcesses := collectTopProcessStats(total)

	return DashboardSystemStats{
		CPUCores:           runtime.NumCPU(),
		CPULoadPercent:     readCPULoadPercent(),
		MemoryTotalBytes:   total,
		MemoryUsedBytes:    used,
		MemoryFreeBytes:    free,
		MemoryUsedPercent:  round1(usedPercent),
		Goroutines:         runtime.NumGoroutine(),
		ControlPlaneHeapMB: memStats.HeapAlloc / (1024 * 1024),
		TopCPUProcesses:    topCPUProcesses,
		TopMemoryProcesses: topMemoryProcesses,
	}
}

func readCPULoadPercent() float64 {
	current, ok := readCPUTimesSample()
	if !ok {
		return 0
	}

	cpuUsageState.mu.Lock()
	if cpuUsageState.hasSample {
		percent, valid := calculateCPUUsagePercent(cpuUsageState.lastSample, current)
		cpuUsageState.lastSample = current
		if valid {
			cpuUsageState.lastPercent = percent
			cpuUsageState.mu.Unlock()
			return percent
		}
		lastPercent := cpuUsageState.lastPercent
		cpuUsageState.mu.Unlock()
		return lastPercent
	}
	cpuUsageState.lastSample = current
	cpuUsageState.hasSample = true
	cpuUsageState.mu.Unlock()

	time.Sleep(120 * time.Millisecond)

	next, ok := readCPUTimesSample()
	if !ok {
		return 0
	}
	percent, valid := calculateCPUUsagePercent(current, next)
	if !valid {
		return 0
	}

	cpuUsageState.mu.Lock()
	cpuUsageState.lastSample = next
	cpuUsageState.lastPercent = percent
	cpuUsageState.hasSample = true
	cpuUsageState.mu.Unlock()
	return percent
}

func readCPUTimesSample() (cpuTimesSample, bool) {
	content, err := os.ReadFile("/proc/stat")
	if err != nil {
		return cpuTimesSample{}, false
	}
	lines := strings.Split(string(content), "\n")
	if len(lines) == 0 {
		return cpuTimesSample{}, false
	}
	fields := strings.Fields(strings.TrimSpace(lines[0]))
	if len(fields) < 5 || fields[0] != "cpu" {
		return cpuTimesSample{}, false
	}
	var total uint64
	values := make([]uint64, 0, len(fields)-1)
	for _, field := range fields[1:] {
		value, err := strconv.ParseUint(field, 10, 64)
		if err != nil {
			return cpuTimesSample{}, false
		}
		values = append(values, value)
		total += value
	}
	idle := values[3]
	if len(values) > 4 {
		idle += values[4]
	}
	return cpuTimesSample{
		idle:     idle,
		total:    total,
		captured: time.Now(),
	}, true
}

func calculateCPUUsagePercent(previous, current cpuTimesSample) (float64, bool) {
	if current.total <= previous.total {
		return 0, false
	}
	totalDelta := current.total - previous.total
	if totalDelta == 0 {
		return 0, false
	}
	idleDelta := uint64(0)
	if current.idle >= previous.idle {
		idleDelta = current.idle - previous.idle
	}
	busyDelta := totalDelta - minUint64(idleDelta, totalDelta)
	percent := round1((float64(busyDelta) * 100) / float64(totalDelta))
	if percent < 0 {
		return 0, false
	}
	if percent > 100 {
		percent = 100
	}
	return percent, true
}

func minUint64(left, right uint64) uint64 {
	if left < right {
		return left
	}
	return right
}

func readMemoryFromProc() (uint64, uint64) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	defer file.Close()

	var totalKB uint64
	var availableKB uint64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			totalKB = parseMemInfoKB(line)
		} else if strings.HasPrefix(line, "MemAvailable:") {
			availableKB = parseMemInfoKB(line)
		}
	}
	if totalKB == 0 {
		return 0, 0
	}
	return totalKB * 1024, availableKB * 1024
}

func parseMemInfoKB(line string) uint64 {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0
	}
	value, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return 0
	}
	return value
}

func parseAnyTime(value any) time.Time {
	raw := strings.TrimSpace(asString(value))
	if raw == "" {
		return time.Time{}
	}
	if ts, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return ts.UTC()
	}
	if ts, err := time.Parse(time.RFC3339, raw); err == nil {
		return ts.UTC()
	}
	return time.Time{}
}

func parseAnyInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case float32:
		return int(typed)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err != nil {
			return 0
		}
		return parsed
	default:
		return 0
	}
}

func asString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []byte:
		return string(typed)
	case int:
		return strconv.Itoa(typed)
	case int32:
		return strconv.Itoa(int(typed))
	case int64:
		return strconv.FormatInt(typed, 10)
	case float64:
		return strconv.Itoa(int(typed))
	case float32:
		return strconv.Itoa(int(typed))
	default:
		return ""
	}
}

func round1(value float64) float64 {
	return float64(int(value*10+0.5)) / 10
}

func collectTopProcessStats(totalMemoryBytes uint64) ([]DashboardProcessStats, []DashboardProcessStats) {
	if runtime.GOOS != "linux" {
		return nil, nil
	}

	first, firstTotal, ok := readProcessSamples()
	if !ok {
		return nil, nil
	}

	time.Sleep(120 * time.Millisecond)

	second, secondTotal, ok := readProcessSamples()
	if !ok {
		second = first
		secondTotal = firstTotal
	}

	totalDelta := uint64(0)
	if secondTotal > firstTotal {
		totalDelta = secondTotal - firstTotal
	}

	items := make([]DashboardProcessStats, 0, len(second))
	for pid, current := range second {
		if current.name == "" {
			continue
		}
		previous, hasPrevious := first[pid]
		cpuPercent := 0.0
		if hasPrevious && totalDelta > 0 && current.cpuJiffies >= previous.cpuJiffies {
			cpuPercent = round1(float64(current.cpuJiffies-previous.cpuJiffies) * 100 / float64(totalDelta))
		}

		memoryPercent := 0.0
		if totalMemoryBytes > 0 {
			memoryPercent = round1(float64(current.rssBytes) * 100 / float64(totalMemoryBytes))
		}

		items = append(items, DashboardProcessStats{
			PID:            pid,
			Name:           current.name,
			Command:        current.command,
			State:          current.state,
			Threads:        current.threads,
			CPUPercent:     cpuPercent,
			MemoryRSSBytes: current.rssBytes,
			MemoryPercent:  memoryPercent,
		})
	}

	topCPU := append([]DashboardProcessStats(nil), items...)
	sort.Slice(topCPU, func(i, j int) bool {
		if topCPU[i].CPUPercent == topCPU[j].CPUPercent {
			if topCPU[i].MemoryRSSBytes == topCPU[j].MemoryRSSBytes {
				return topCPU[i].PID < topCPU[j].PID
			}
			return topCPU[i].MemoryRSSBytes > topCPU[j].MemoryRSSBytes
		}
		return topCPU[i].CPUPercent > topCPU[j].CPUPercent
	})
	if len(topCPU) > 10 {
		topCPU = topCPU[:10]
	}

	topMemory := append([]DashboardProcessStats(nil), items...)
	sort.Slice(topMemory, func(i, j int) bool {
		if topMemory[i].MemoryRSSBytes == topMemory[j].MemoryRSSBytes {
			if topMemory[i].CPUPercent == topMemory[j].CPUPercent {
				return topMemory[i].PID < topMemory[j].PID
			}
			return topMemory[i].CPUPercent > topMemory[j].CPUPercent
		}
		return topMemory[i].MemoryRSSBytes > topMemory[j].MemoryRSSBytes
	})
	if len(topMemory) > 10 {
		topMemory = topMemory[:10]
	}

	return topCPU, topMemory
}

func readProcessSamples() (map[int]processSample, uint64, bool) {
	totalSample, ok := readCPUTimesSample()
	if !ok {
		return nil, 0, false
	}

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, 0, false
	}

	samples := make(map[int]processSample, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 0 {
			continue
		}
		sample, ok := readSingleProcessSample(pid)
		if !ok {
			continue
		}
		samples[pid] = sample
	}

	return samples, totalSample.total, true
}

func readSingleProcessSample(pid int) (processSample, bool) {
	procRoot := filepath.Join("/proc", strconv.Itoa(pid))
	statContent, err := os.ReadFile(filepath.Join(procRoot, "stat"))
	if err != nil {
		return processSample{}, false
	}
	sample, ok := parseProcessStat(pid, string(statContent))
	if !ok {
		return processSample{}, false
	}

	if cmdlineContent, err := os.ReadFile(filepath.Join(procRoot, "cmdline")); err == nil {
		command := strings.TrimSpace(strings.ReplaceAll(string(cmdlineContent), "\x00", " "))
		if command != "" {
			sample.command = command
		}
	}
	if sample.command == "" {
		sample.command = sample.name
	}
	return sample, true
}

func parseProcessStat(pid int, raw string) (processSample, bool) {
	raw = strings.TrimSpace(raw)
	closing := strings.LastIndex(raw, ")")
	opening := strings.Index(raw, "(")
	if opening < 0 || closing <= opening {
		return processSample{}, false
	}

	name := strings.TrimSpace(raw[opening+1 : closing])
	rest := strings.Fields(strings.TrimSpace(raw[closing+1:]))
	if len(rest) < 22 {
		return processSample{}, false
	}

	utime, err := strconv.ParseUint(rest[11], 10, 64)
	if err != nil {
		return processSample{}, false
	}
	stime, err := strconv.ParseUint(rest[12], 10, 64)
	if err != nil {
		return processSample{}, false
	}
	threads, err := strconv.Atoi(rest[17])
	if err != nil {
		threads = 0
	}
	rssPages, err := strconv.ParseInt(rest[21], 10, 64)
	if err != nil {
		rssPages = 0
	}

	return processSample{
		pid:        pid,
		name:       name,
		state:      strings.TrimSpace(rest[0]),
		threads:    threads,
		rssBytes:   uint64(maxInt64(rssPages, 0)) * uint64(os.Getpagesize()),
		cpuJiffies: utime + stime,
	}, true
}

func maxInt64(left, right int64) int64 {
	if left > right {
		return left
	}
	return right
}
