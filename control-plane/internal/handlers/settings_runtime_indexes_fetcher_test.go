package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestRuntimeIndexFetcherFetchIncludesRuntimeToken(t *testing.T) {
	received := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r.Header.Get("X-WAF-Runtime-Token")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[],"total":0,"limit":10,"offset":0,"stream":"requests"}`))
	}))
	defer server.Close()

	fetcher := &runtimeIndexFetcher{
		url:    server.URL,
		client: server.Client(),
		token:  "runtime-secret",
	}
	if _, err := fetcher.Fetch("requests", 10, 0); err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if received != "runtime-secret" {
		t.Fatalf("expected runtime token header, got %q", received)
	}
}

func TestRuntimeIndexFetcherDeleteIncludesRuntimeToken(t *testing.T) {
	received := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r.Header.Get("X-WAF-Runtime-Token")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	fetcher := &runtimeIndexFetcher{
		url:    server.URL,
		client: server.Client(),
		token:  "runtime-secret",
	}
	if err := fetcher.Delete("requests", "2026-06-24"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if received != "runtime-secret" {
		t.Fatalf("expected runtime token header, got %q", received)
	}
}

func TestRuntimeIndexFetcherFetchFallsBackFromRuntimeHost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"date":"2026-06-24"}],"total":1,"limit":10,"offset":0,"stream":"requests"}`))
	}))
	defer server.Close()

	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	fetcher := &runtimeIndexFetcher{
		url:    "http://runtime:" + parsed.Port() + "/requests/indexes",
		client: server.Client(),
		token:  "runtime-secret",
	}
	payload, err := fetcher.Fetch("requests", 10, 0)
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if got := payload["total"]; got != float64(1) && got != 1 {
		t.Fatalf("expected total=1, got %v", got)
	}
}
