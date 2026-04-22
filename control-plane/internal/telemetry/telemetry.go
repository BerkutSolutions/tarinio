package telemetry

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"waf/control-plane/internal/appmeta"
	"waf/internal/observability"
)

type Metrics struct {
	registry           *observability.Registry
	buildInfo          *observability.GaugeVec
	httpRequests       *observability.CounterVec
	httpDuration       *observability.HistogramVec
	revisionCompile    *observability.CounterVec
	revisionApply      *observability.CounterVec
	haLockAcquire      *observability.CounterVec
	haLockWait         *observability.HistogramVec
	haLeaderRuns       *observability.CounterVec
	runtimeReloadCalls *observability.CounterVec
}

var defaultMetrics = newMetrics()

func newMetrics() *Metrics {
	registry := observability.NewRegistry()
	return &Metrics{
		registry:           registry,
		buildInfo:          registry.Gauge("tarinio_build_info", "Build and node identity for the running TARINIO control-plane", "version", "node", "ha_enabled"),
		httpRequests:       registry.Counter("tarinio_http_requests_total", "Total HTTP requests served by the TARINIO control-plane", "node", "method", "route", "status"),
		httpDuration:       registry.Histogram("tarinio_http_request_duration_seconds", "HTTP request duration served by the TARINIO control-plane", observability.DefaultDurationBuckets(), "node", "method", "route"),
		revisionCompile:    registry.Counter("tarinio_revision_compile_total", "Revision compile attempts grouped by outcome", "node", "status"),
		revisionApply:      registry.Counter("tarinio_revision_apply_total", "Revision apply attempts grouped by outcome", "node", "status"),
		haLockAcquire:      registry.Counter("tarinio_ha_lock_acquire_total", "HA lock acquisition attempts grouped by result", "node", "key", "status"),
		haLockWait:         registry.Histogram("tarinio_ha_lock_wait_seconds", "Time spent waiting for HA lock acquisition", observability.DefaultDurationBuckets(), "node", "key"),
		haLeaderRuns:       registry.Counter("tarinio_ha_leader_runs_total", "Leader election attempts grouped by result", "node", "key", "status"),
		runtimeReloadCalls: registry.Counter("tarinio_runtime_reload_calls_total", "Runtime reload invocations delegated by the control-plane", "node", "status"),
	}
}

func Default() *Metrics {
	return defaultMetrics
}

func (m *Metrics) Registry() *observability.Registry {
	return m.registry
}

func (m *Metrics) RecordBuild(nodeID string, haEnabled bool) {
	if m == nil {
		return
	}
	ha := "false"
	if haEnabled {
		ha = "true"
	}
	m.buildInfo.Set(map[string]string{
		"version":    appmeta.AppVersion,
		"node":       normalizeNode(nodeID),
		"ha_enabled": ha,
	}, 1)
}

func (m *Metrics) InstrumentHTTP(next http.Handler, nodeID string) http.Handler {
	if m == nil || next == nil {
		return next
	}
	nodeID = normalizeNode(nodeID)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(recorder, r)
		route := normalizeRoute(r.URL.Path)
		statusCode := recorder.statusCode
		duration := time.Since(start)
		m.httpRequests.Inc(map[string]string{
			"node":   nodeID,
			"method": strings.ToUpper(strings.TrimSpace(r.Method)),
			"route":  route,
			"status": strconv.Itoa(statusCode),
		})
		m.httpDuration.Observe(map[string]string{
			"node":   nodeID,
			"method": strings.ToUpper(strings.TrimSpace(r.Method)),
			"route":  route,
		}, duration.Seconds())
		logHTTPRequest(nodeID, r, route, statusCode, recorder.bytesWritten, duration)
	})
}

func (m *Metrics) RecordRevisionCompile(nodeID, status string) {
	m.revisionCompile.Inc(map[string]string{"node": normalizeNode(nodeID), "status": normalizeStatus(status)})
}

func (m *Metrics) RecordRevisionApply(nodeID, status string) {
	m.revisionApply.Inc(map[string]string{"node": normalizeNode(nodeID), "status": normalizeStatus(status)})
}

func (m *Metrics) RecordHALock(nodeID, key, status string, wait time.Duration) {
	if m == nil {
		return
	}
	nodeID = normalizeNode(nodeID)
	key = normalizeKey(key)
	status = normalizeStatus(status)
	m.haLockAcquire.Inc(map[string]string{"node": nodeID, "key": key, "status": status})
	m.haLockWait.Observe(map[string]string{"node": nodeID, "key": key}, wait.Seconds())
}

func (m *Metrics) RecordLeaderRun(nodeID, key, status string) {
	if m == nil {
		return
	}
	m.haLeaderRuns.Inc(map[string]string{
		"node":   normalizeNode(nodeID),
		"key":    normalizeKey(key),
		"status": normalizeStatus(status),
	})
}

func (m *Metrics) RecordRuntimeReload(nodeID, status string) {
	if m == nil {
		return
	}
	m.runtimeReloadCalls.Inc(map[string]string{"node": normalizeNode(nodeID), "status": normalizeStatus(status)})
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (s *statusRecorder) WriteHeader(statusCode int) {
	s.statusCode = statusCode
	s.ResponseWriter.WriteHeader(statusCode)
}

func (s *statusRecorder) Write(p []byte) (int, error) {
	if s.statusCode == 0 {
		s.statusCode = http.StatusOK
	}
	n, err := s.ResponseWriter.Write(p)
	s.bytesWritten += n
	return n, err
}

type httpRequestLogEntry struct {
	Timestamp     string   `json:"timestamp"`
	Component     string   `json:"component"`
	Node          string   `json:"node"`
	Method        string   `json:"method"`
	Route         string   `json:"route"`
	Path          string   `json:"path"`
	Host          string   `json:"host,omitempty"`
	Status        int      `json:"status"`
	DurationMS    int64    `json:"duration_ms"`
	ResponseBytes int      `json:"response_bytes"`
	RemoteIP      string   `json:"remote_ip,omitempty"`
	ForwardedFor  string   `json:"forwarded_for,omitempty"`
	QueryKeys     []string `json:"query_keys,omitempty"`
	RequestID     string   `json:"request_id,omitempty"`
	Referer       string   `json:"referer,omitempty"`
	UserAgent     string   `json:"user_agent,omitempty"`
	ContentLength int64    `json:"content_length,omitempty"`
}

func logHTTPRequest(nodeID string, r *http.Request, route string, statusCode int, responseBytes int, duration time.Duration) {
	if r == nil || !shouldLogHTTPRequest(r.URL.Path) {
		return
	}
	entry := httpRequestLogEntry{
		Timestamp:     time.Now().UTC().Format(time.RFC3339Nano),
		Component:     "control-plane.http",
		Node:          normalizeNode(nodeID),
		Method:        strings.ToUpper(strings.TrimSpace(r.Method)),
		Route:         route,
		Path:          strings.TrimSpace(r.URL.Path),
		Host:          strings.TrimSpace(r.Host),
		Status:        statusCode,
		DurationMS:    duration.Milliseconds(),
		ResponseBytes: responseBytes,
		RemoteIP:      requestRemoteIP(r),
		ForwardedFor:  truncateLogValue(r.Header.Get("X-Forwarded-For"), 256),
		QueryKeys:     requestQueryKeys(r),
		RequestID:     truncateLogValue(firstHeader(r, "X-Request-ID", "X-Correlation-ID"), 128),
		Referer:       truncateLogValue(r.Header.Get("Referer"), 256),
		UserAgent:     truncateLogValue(r.UserAgent(), 256),
	}
	if r.ContentLength > 0 {
		entry.ContentLength = r.ContentLength
	}
	raw, err := json.Marshal(entry)
	if err != nil {
		log.Printf("[warn] telemetry request log marshal failed: %v", err)
		return
	}
	log.Printf("%s", raw)
}

func shouldLogHTTPRequest(path string) bool {
	switch strings.TrimSpace(path) {
	case "", "/healthz", "/metrics":
		return false
	default:
		return true
	}
}

func requestRemoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func requestQueryKeys(r *http.Request) []string {
	if r == nil || r.URL == nil {
		return nil
	}
	values := r.URL.Query()
	if len(values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		keys = append(keys, trimmed)
	}
	sort.Strings(keys)
	return keys
}

func firstHeader(r *http.Request, names ...string) string {
	for _, name := range names {
		value := strings.TrimSpace(r.Header.Get(name))
		if value != "" {
			return value
		}
	}
	return ""
}

func truncateLogValue(value string, limit int) string {
	value = strings.TrimSpace(value)
	if value == "" || limit <= 0 {
		return value
	}
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}

func normalizeNode(nodeID string) string {
	nodeID = strings.TrimSpace(nodeID)
	if nodeID == "" {
		return "single-node"
	}
	return nodeID
}

func normalizeStatus(status string) string {
	status = strings.TrimSpace(strings.ToLower(status))
	if status == "" {
		return "unknown"
	}
	return status
}

func normalizeKey(key string) string {
	key = strings.TrimSpace(strings.ToLower(key))
	if key == "" {
		return "unspecified"
	}
	return strings.NewReplacer(":", "_", "/", "_", "-", "_").Replace(key)
}

func normalizeRoute(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	if strings.HasPrefix(path, "/api/revisions/") {
		switch {
		case strings.HasSuffix(path, "/apply"):
			return "/api/revisions/:id/apply"
		default:
			return "/api/revisions/:id"
		}
	}
	if strings.HasPrefix(path, "/api/sites/") {
		return "/api/sites/:id"
	}
	if strings.HasPrefix(path, "/api/upstreams/") {
		return "/api/upstreams/:id"
	}
	if strings.HasPrefix(path, "/api/tls-configs/") {
		return "/api/tls-configs/:id"
	}
	if strings.HasPrefix(path, "/api/waf-policies/") {
		return "/api/waf-policies/:id"
	}
	if strings.HasPrefix(path, "/api/access-policies/") {
		return "/api/access-policies/:id"
	}
	if strings.HasPrefix(path, "/api/rate-limit-policies/") {
		return "/api/rate-limit-policies/:id"
	}
	if strings.HasPrefix(path, "/api/easy-site-profiles/") {
		return "/api/easy-site-profiles/:id"
	}
	if strings.HasPrefix(path, "/api/certificates/") {
		return "/api/certificates/:id"
	}
	if strings.HasPrefix(path, "/api/certificate-materials/export/") {
		return "/api/certificate-materials/export/:id"
	}
	if strings.HasPrefix(path, "/api/administration/scripts/runs/") {
		return "/api/administration/scripts/runs/:id/download"
	}
	if strings.HasPrefix(path, "/api/administration/scripts/") && strings.HasSuffix(path, "/run") {
		return "/api/administration/scripts/:id/run"
	}
	return path
}
