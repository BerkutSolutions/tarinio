package services

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"waf/compiler/pipeline"
)

func TestRuntimeEndpointCandidates(t *testing.T) {
	candidates := runtimeEndpointCandidates("http://runtime:8081/reload", "http://127.0.0.1:8081/reload")
	if len(candidates) < 3 {
		t.Fatalf("expected fallback candidates, got %+v", candidates)
	}
	if candidates[0] != "http://runtime:8081/reload" {
		t.Fatalf("expected original url first, got %q", candidates[0])
	}

	matchedLoopback := false
	for _, candidate := range candidates {
		if strings.Contains(candidate, "127.0.0.1:8081") || strings.Contains(candidate, "localhost:8081") {
			matchedLoopback = true
			break
		}
	}
	if !matchedLoopback {
		t.Fatalf("expected loopback candidate, got %+v", candidates)
	}
}

func TestHTTPReloadExecutorFallsBackFromRuntimeHost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/reload" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	executor := HTTPReloadExecutor{
		URL: "http://runtime:" + parsed.Port() + "/reload",
	}
	if err := executor.Run("nginx", []string{"reload"}, ""); err != nil {
		t.Fatalf("expected fallback reload success, got: %v", err)
	}
}

func TestHTTPReloadExecutorRetriesOneTransportInterruption(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			hijacker := w.(http.Hijacker)
			conn, _, err := hijacker.Hijack()
			if err != nil {
				t.Fatalf("hijack first request: %v", err)
			}
			_ = conn.Close()
			return
		}
		if r.URL.Path != "/reload" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	if err := (HTTPReloadExecutor{URL: server.URL + "/reload"}).Run("nginx", nil, ""); err != nil {
		t.Fatalf("transport retry must recover: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected one retry, got %d attempts", attempts)
	}
}

func TestHTTPReloadExecutorDoesNotRetryRuntimeResponseFailure(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		http.Error(w, "invalid candidate", http.StatusBadGateway)
	}))
	defer server.Close()
	if err := (HTTPReloadExecutor{URL: server.URL + "/reload"}).Run("nginx", nil, ""); err == nil {
		t.Fatal("runtime response failure must fail apply")
	}
	if attempts != 1 {
		t.Fatalf("runtime response failure must not retry, got %d attempts", attempts)
	}
}

func TestHTTPHealthCheckerFallsBackFromRuntimeHost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/readyz" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("X-WAF-Runtime-Revision", "rev-expected")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	checker := HTTPHealthChecker{
		URL: "http://runtime:" + parsed.Port() + "/readyz",
	}
	if err := checker.Check(&pipeline.ActivePointer{RevisionID: "rev-expected"}); err != nil {
		t.Fatalf("expected fallback health success, got: %v", err)
	}
}

func TestHTTPHealthCheckerRejectsDifferentRuntimeRevision(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-WAF-Runtime-Revision", "rev-stale")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	checker := HTTPHealthChecker{URL: server.URL}
	err := checker.Check(&pipeline.ActivePointer{RevisionID: "rev-expected"})
	if err == nil || !strings.Contains(err.Error(), "rev-stale") {
		t.Fatalf("expected stale revision error, got %v", err)
	}
}
