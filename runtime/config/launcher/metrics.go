package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const runtimeMetricsHeader = "X-TARINIO-Metrics-Token"

type runtimeMetrics struct {
	mu             sync.RWMutex
	requests       map[string]uint64
	durations      map[string]*runtimeHistogram
	reloads        map[string]uint64
	bundleLoads    map[string]uint64
	live           float64
	ready          float64
	activeRevision string
}

type runtimeHistogram struct {
	count   uint64
	sum     float64
	buckets []uint64
}

var runtimeBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60}

func newRuntimeMetrics() *runtimeMetrics {
	return &runtimeMetrics{
		requests:    map[string]uint64{},
		durations:   map[string]*runtimeHistogram{},
		reloads:     map[string]uint64{},
		bundleLoads: map[string]uint64{},
		live:        1,
	}
}

func (m *runtimeMetrics) instrument(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &runtimeStatusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(recorder, r)
		m.recordRequest(strings.ToUpper(strings.TrimSpace(r.Method)), normalizeRuntimeRoute(r.URL.Path), recorder.statusCode, time.Since(start))
	})
}

func (m *runtimeMetrics) recordRequest(method, route string, status int, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := method + "\xff" + route + "\xff" + strconv.Itoa(status)
	m.requests[key]++
	durationKey := method + "\xff" + route
	item, ok := m.durations[durationKey]
	if !ok {
		item = &runtimeHistogram{buckets: make([]uint64, len(runtimeBuckets))}
		m.durations[durationKey] = item
	}
	item.count++
	item.sum += duration.Seconds()
	for i, boundary := range runtimeBuckets {
		if duration.Seconds() <= boundary {
			item.buckets[i]++
		}
	}
}

func (m *runtimeMetrics) recordReload(status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reloads[normalizeRuntimeStatus(status)]++
}

func (m *runtimeMetrics) recordBundleLoad(status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bundleLoads[normalizeRuntimeStatus(status)]++
}

func (m *runtimeMetrics) setLive(value bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if value {
		m.live = 1
		return
	}
	m.live = 0
}

func (m *runtimeMetrics) setReady(value bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if value {
		m.ready = 1
		return
	}
	m.ready = 0
}

func (m *runtimeMetrics) setActiveRevision(revisionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activeRevision = strings.TrimSpace(revisionID)
}

func (m *runtimeMetrics) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	expected := strings.TrimSpace(os.Getenv("WAF_RUNTIME_METRICS_TOKEN"))
	presented := strings.TrimSpace(r.Header.Get(runtimeMetricsHeader))
	if presented == "" {
		presented = strings.TrimSpace(r.URL.Query().Get("token"))
	}
	if expected != "" && presented != expected {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	_ = m.writePrometheus(w)
}

func (m *runtimeMetrics) writePrometheus(w io.Writer) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, err := fmt.Fprintln(w, "# HELP tarinio_runtime_http_requests_total Total HTTP requests served by the TARINIO runtime control API"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "# TYPE tarinio_runtime_http_requests_total counter"); err != nil {
		return err
	}
	requestKeys := make([]string, 0, len(m.requests))
	for key := range m.requests {
		requestKeys = append(requestKeys, key)
	}
	sort.Strings(requestKeys)
	for _, key := range requestKeys {
		parts := strings.Split(key, "\xff")
		if len(parts) != 3 {
			continue
		}
		if _, err := fmt.Fprintf(w, "tarinio_runtime_http_requests_total{method=%q,route=%q,status=%q} %d\n", parts[0], parts[1], parts[2], m.requests[key]); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintln(w, "# HELP tarinio_runtime_http_request_duration_seconds Runtime control API request duration"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "# TYPE tarinio_runtime_http_request_duration_seconds histogram"); err != nil {
		return err
	}
	durationKeys := make([]string, 0, len(m.durations))
	for key := range m.durations {
		durationKeys = append(durationKeys, key)
	}
	sort.Strings(durationKeys)
	for _, key := range durationKeys {
		parts := strings.Split(key, "\xff")
		if len(parts) != 2 {
			continue
		}
		item := m.durations[key]
		cumulative := uint64(0)
		for i, boundary := range runtimeBuckets {
			cumulative += item.buckets[i]
			if _, err := fmt.Fprintf(w, "tarinio_runtime_http_request_duration_seconds_bucket{method=%q,route=%q,le=%q} %d\n", parts[0], parts[1], formatRuntimeFloat(boundary), cumulative); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(w, "tarinio_runtime_http_request_duration_seconds_bucket{method=%q,route=%q,le=\"+Inf\"} %d\n", parts[0], parts[1], item.count); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "tarinio_runtime_http_request_duration_seconds_sum{method=%q,route=%q} %s\n", parts[0], parts[1], formatRuntimeFloat(item.sum)); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "tarinio_runtime_http_request_duration_seconds_count{method=%q,route=%q} %d\n", parts[0], parts[1], item.count); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintln(w, "# HELP tarinio_runtime_reload_total Runtime reload attempts grouped by outcome"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "# TYPE tarinio_runtime_reload_total counter"); err != nil {
		return err
	}
	reloadStatuses := make([]string, 0, len(m.reloads))
	for status := range m.reloads {
		reloadStatuses = append(reloadStatuses, status)
	}
	sort.Strings(reloadStatuses)
	for _, status := range reloadStatuses {
		if _, err := fmt.Fprintf(w, "tarinio_runtime_reload_total{status=%q} %d\n", status, m.reloads[status]); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintln(w, "# HELP tarinio_runtime_bundle_load_total Runtime bundle load attempts grouped by outcome"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "# TYPE tarinio_runtime_bundle_load_total counter"); err != nil {
		return err
	}
	bundleStatuses := make([]string, 0, len(m.bundleLoads))
	for status := range m.bundleLoads {
		bundleStatuses = append(bundleStatuses, status)
	}
	sort.Strings(bundleStatuses)
	for _, status := range bundleStatuses {
		if _, err := fmt.Fprintf(w, "tarinio_runtime_bundle_load_total{status=%q} %d\n", status, m.bundleLoads[status]); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintln(w, "# HELP tarinio_runtime_live Runtime liveness state"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "# TYPE tarinio_runtime_live gauge"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "tarinio_runtime_live %s\n", formatRuntimeFloat(m.live)); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(w, "# HELP tarinio_runtime_ready Runtime readiness state"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "# TYPE tarinio_runtime_ready gauge"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "tarinio_runtime_ready %s\n", formatRuntimeFloat(m.ready)); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(w, "# HELP tarinio_runtime_active_revision_info Active runtime revision identity"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "# TYPE tarinio_runtime_active_revision_info gauge"); err != nil {
		return err
	}
	if strings.TrimSpace(m.activeRevision) != "" {
		if _, err := fmt.Fprintf(w, "tarinio_runtime_active_revision_info{revision_id=%q} 1\n", m.activeRevision); err != nil {
			return err
		}
	}
	return nil
}

type runtimeStatusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (s *runtimeStatusRecorder) WriteHeader(statusCode int) {
	s.statusCode = statusCode
	s.ResponseWriter.WriteHeader(statusCode)
}

func normalizeRuntimeStatus(status string) string {
	status = strings.TrimSpace(strings.ToLower(status))
	if status == "" {
		return "unknown"
	}
	return status
}

func normalizeRuntimeRoute(path string) string {
	switch strings.TrimSpace(path) {
	case "":
		return "/"
	case "/security-events/probe":
		return "/security-events/probe"
	case "/security-events":
		return "/security-events"
	case "/requests/probe":
		return "/requests/probe"
	case "/requests/indexes":
		return "/requests/indexes"
	case "/requests":
		return "/requests"
	case "/crs/status":
		return "/crs/status"
	case "/crs/check-updates":
		return "/crs/check-updates"
	case "/crs/update":
		return "/crs/update"
	default:
		return path
	}
}

func formatRuntimeFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
