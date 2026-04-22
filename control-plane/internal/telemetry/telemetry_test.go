package telemetry

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInstrumentHTTPLogsStructuredRequestSummary(t *testing.T) {
	var output bytes.Buffer
	prevWriter := log.Writer()
	prevFlags := log.Flags()
	log.SetOutput(&output)
	log.SetFlags(0)
	defer log.SetOutput(prevWriter)
	defer log.SetFlags(prevFlags)

	metrics := newMetrics()
	handler := metrics.InstrumentHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("ok"))
	}), "node-a")

	req := httptest.NewRequest(http.MethodPost, "/api/administration/scripts/collect-waf-events/run?site=sentry&status=429", nil)
	req.Host = "waf.example.test"
	req.RemoteAddr = "203.0.113.5:43125"
	req.Header.Set("X-Forwarded-For", "198.51.100.10, 203.0.113.5")
	req.Header.Set("X-Request-ID", "req-123")
	req.Header.Set("User-Agent", "curl/8.7.1")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	logLine := output.String()
	for _, fragment := range []string{
		`"component":"control-plane.http"`,
		`"node":"node-a"`,
		`"method":"POST"`,
		`"route":"/api/administration/scripts/:id/run"`,
		`"path":"/api/administration/scripts/collect-waf-events/run"`,
		`"status":202`,
		`"remote_ip":"203.0.113.5"`,
		`"forwarded_for":"198.51.100.10, 203.0.113.5"`,
		`"request_id":"req-123"`,
		`"query_keys":["site","status"]`,
	} {
		if !strings.Contains(logLine, fragment) {
			t.Fatalf("expected log output to contain %q, got %q", fragment, logLine)
		}
	}
}

func TestInstrumentHTTPSkipsHealthEndpoints(t *testing.T) {
	var output bytes.Buffer
	prevWriter := log.Writer()
	prevFlags := log.Flags()
	log.SetOutput(&output)
	log.SetFlags(0)
	defer log.SetOutput(prevWriter)
	defer log.SetFlags(prevFlags)

	metrics := newMetrics()
	handler := metrics.InstrumentHTTP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), "node-a")

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if strings.Contains(output.String(), `"component":"control-plane.http"`) {
		t.Fatalf("expected no request log for health endpoint, got %q", output.String())
	}
}
