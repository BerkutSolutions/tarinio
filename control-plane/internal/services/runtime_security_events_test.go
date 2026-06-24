package services

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestHTTPRuntimeSecurityEventCollectorCollectFallsBackFromRuntimeHost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/security-events" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"events":[]}`))
	}))
	defer server.Close()

	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	collector := &HTTPRuntimeSecurityEventCollector{
		URL:    "http://runtime:" + parsed.Port() + "/security-events",
		Client: server.Client(),
	}
	if _, err := collector.Collect(); err != nil {
		t.Fatalf("expected fallback collect success, got: %v", err)
	}
}
