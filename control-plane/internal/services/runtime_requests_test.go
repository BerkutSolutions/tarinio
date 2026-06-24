package services

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestHTTPRuntimeRequestCollectorProbeFallsBackFromRuntimeHost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/requests/probe" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	collector := &HTTPRuntimeRequestCollector{
		URL:    "http://runtime:" + parsed.Port() + "/requests",
		Client: server.Client(),
	}
	if err := collector.Probe(nil); err != nil {
		t.Fatalf("expected fallback probe success, got: %v", err)
	}
}
