package telemetry

import (
	"net/http"
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
		}, time.Since(start).Seconds())
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
	statusCode int
}

func (s *statusRecorder) WriteHeader(statusCode int) {
	s.statusCode = statusCode
	s.ResponseWriter.WriteHeader(statusCode)
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
	return path
}
