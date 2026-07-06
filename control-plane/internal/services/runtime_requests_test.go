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

func TestHTTPRuntimeRequestCollectorCountFallsBackFromRuntimeHost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/requests/count" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"count":9}`))
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
	count, err := collector.CollectCount(nil)
	if err != nil {
		t.Fatalf("expected fallback count success, got: %v", err)
	}
	if count != 9 {
		t.Fatalf("expected count=9, got %d", count)
	}
}

func TestSetRuntimeAuthHeaderSetsLegacyAndCurrentHeaders(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://example.test", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	setRuntimeAuthHeader(req, " runtime-secret ")

	if got := req.Header.Get(runtimeAuthHeader); got != "runtime-secret" {
		t.Fatalf("expected %s header to be trimmed token, got %q", runtimeAuthHeader, got)
	}
	if got := req.Header.Get(legacyRuntimeAuthHeader); got != "runtime-secret" {
		t.Fatalf("expected %s header to be trimmed token, got %q", legacyRuntimeAuthHeader, got)
	}
}
