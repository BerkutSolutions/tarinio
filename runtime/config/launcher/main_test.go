package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveCandidatePath(t *testing.T) {
	absolute := filepath.Join("C:\\runtime", "candidates", "rev-001")
	resolved, err := resolveCandidatePath(filepath.Join("C:\\runtime"), absolute)
	if err != nil {
		t.Fatalf("resolve absolute failed: %v", err)
	}
	if resolved != absolute {
		t.Fatalf("unexpected absolute resolution: %s", resolved)
	}

	relative, err := resolveCandidatePath(filepath.Join("C:\\runtime"), filepath.Join("candidates", "rev-002"))
	if err != nil {
		t.Fatalf("resolve relative failed: %v", err)
	}
	if relative != filepath.Join("C:\\runtime", "candidates", "rev-002") {
		t.Fatalf("unexpected relative resolution: %s", relative)
	}
}

func TestSelectFirstExisting(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "missing")
	second := filepath.Join(root, "present")
	if err := os.WriteFile(second, []byte("ok"), 0o644); err != nil {
		t.Fatalf("write fixture failed: %v", err)
	}

	selected, err := selectFirstExisting(first, second)
	if err != nil {
		t.Fatalf("select path failed: %v", err)
	}
	if selected != second {
		t.Fatalf("unexpected selected path: %s", selected)
	}
}

func TestValidateCandidateBundle(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "nginx"), 0o755); err != nil {
		t.Fatalf("mkdir nginx failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "modsecurity"), 0o755); err != nil {
		t.Fatalf("mkdir modsecurity failed: %v", err)
	}
	for _, rel := range []string{
		"manifest.json",
		filepath.Join("nginx", "nginx.conf"),
		filepath.Join("modsecurity", "modsecurity.conf"),
	} {
		if err := os.WriteFile(filepath.Join(root, rel), []byte("ok"), 0o644); err != nil {
			t.Fatalf("write %s failed: %v", rel, err)
		}
	}

	if err := validateCandidateBundle(root); err != nil {
		t.Fatalf("expected valid bundle, got %v", err)
	}
}

func TestRuntimeStatusHealthHandlers(t *testing.T) {
	status := &runtimeStatus{}
	logPath := filepath.Join(t.TempDir(), "access.log")
	server := httptest.NewServer(status.handlers(
		newRuntimeProcess(t.TempDir(), "/tmp/crs", status, nil, "/tmp/module.so"),
		newSecurityEventSource(logPath),
		newRequestStreamSource(logPath, 100, filepath.Join(t.TempDir(), "requests-archive"), 30),
	))
	defer server.Close()

	resp, err := http.Get(server.URL + "/healthz")
	if err != nil {
		t.Fatalf("health request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected liveness success while process is alive, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	resp, err = http.Get(server.URL + "/readyz")
	if err != nil {
		t.Fatalf("ready request failed: %v", err)
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected readiness failure before bundle load, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	status.setActiveBundle(&activePointer{
		RevisionID:    "rev-001",
		CandidatePath: "/var/lib/waf/candidates/rev-001",
	}, "/var/lib/waf/candidates/rev-001")
	status.setNginxRunning(true)

	resp, err = http.Get(server.URL + "/healthz")
	if err != nil {
		t.Fatalf("health request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected liveness success, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	resp, err = http.Get(server.URL + "/readyz")
	if err != nil {
		t.Fatalf("ready request failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected readiness success, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestRuntimeStatusReadinessRequiresRunningNginx(t *testing.T) {
	status := &runtimeStatus{}
	status.setActiveBundle(&activePointer{
		RevisionID:    "rev-001",
		CandidatePath: "/var/lib/waf/candidates/rev-001",
	}, "/var/lib/waf/candidates/rev-001")

	logPath := filepath.Join(t.TempDir(), "access.log")
	server := httptest.NewServer(status.handlers(
		newRuntimeProcess(t.TempDir(), "/tmp/crs", status, nil, "/tmp/module.so"),
		newSecurityEventSource(logPath),
		newRequestStreamSource(logPath, 100, filepath.Join(t.TempDir(), "requests-archive"), 30),
	))
	defer server.Close()

	resp, err := http.Get(server.URL + "/readyz")
	if err != nil {
		t.Fatalf("ready request failed: %v", err)
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected readiness failure without nginx running, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestRuntimeHandlers_RequireTokenWhenConfigured(t *testing.T) {
	t.Setenv("WAF_RUNTIME_API_TOKEN", "runtime-secret")
	status := &runtimeStatus{}
	status.setActiveBundle(&activePointer{
		RevisionID:    "rev-001",
		CandidatePath: "/var/lib/waf/candidates/rev-001",
	}, "/var/lib/waf/candidates/rev-001")
	status.setNginxRunning(true)

	logPath := filepath.Join(t.TempDir(), "access.log")
	server := httptest.NewServer(status.handlers(
		newRuntimeProcess(t.TempDir(), "/tmp/crs", status, nil, "/tmp/module.so"),
		newSecurityEventSource(logPath),
		newRequestStreamSource(logPath, 100, filepath.Join(t.TempDir(), "requests-archive"), 30),
	))
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL+"/readyz", nil)
	if err != nil {
		t.Fatalf("new request failed: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 without runtime token, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	req, err = http.NewRequest(http.MethodGet, server.URL+"/readyz", nil)
	if err != nil {
		t.Fatalf("new request failed: %v", err)
	}
	req.Header.Set(runtimeAuthHeader, "runtime-secret")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request with token failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 with runtime token, got %d", resp.StatusCode)
	}
	_ = resp.Body.Close()
}
