package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

func TestWriteBootstrapNginxConfig(t *testing.T) {
	root := t.TempDir()
	if err := writeBootstrapNginxConfig(root, bootstrapUIUpstream); err != nil {
		t.Fatalf("write bootstrap config failed: %v", err)
	}

	rawConf, err := os.ReadFile(filepath.Join(root, "nginx.conf"))
	if err != nil {
		t.Fatalf("read nginx.conf failed: %v", err)
	}
	if !strings.Contains(string(rawConf), "include /etc/waf/nginx/conf.d/*.conf;") {
		t.Fatalf("unexpected nginx.conf content: %s", string(rawConf))
	}

	rawBootstrap, err := os.ReadFile(filepath.Join(root, "conf.d", "bootstrap.conf"))
	if err != nil {
		t.Fatalf("read bootstrap.conf failed: %v", err)
	}
	text := string(rawBootstrap)
	for _, fragment := range []string{
		"listen 80 default_server;",
		"proxy_pass http://ui:80;",
		"proxy_set_header X-Forwarded-Proto $scheme;",
	} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("expected bootstrap config to contain %q, got %s", fragment, text)
		}
	}
	if strings.Contains(text, "listen 443") {
		t.Fatalf("bootstrap config must not expose 443 before first apply: %s", text)
	}
}

func TestRelinkReplacesPopulatedDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated privileges on Windows test environments")
	}
	root := t.TempDir()
	target := filepath.Join(root, "candidate", "nginx", "conf.d")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir target failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "site.conf"), []byte("server {}"), 0o644); err != nil {
		t.Fatalf("write target file failed: %v", err)
	}

	linkPath := filepath.Join(root, "runtime", "conf.d")
	if err := os.MkdirAll(linkPath, 0o755); err != nil {
		t.Fatalf("mkdir existing link path failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(linkPath, "bootstrap.conf"), []byte("bootstrap"), 0o644); err != nil {
		t.Fatalf("write bootstrap file failed: %v", err)
	}

	if err := relink(target, linkPath); err != nil {
		t.Fatalf("relink failed: %v", err)
	}

	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("stat relinked path failed: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected symlink, got mode %v", info.Mode())
	}
	resolved, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("readlink failed: %v", err)
	}
	if resolved != target {
		t.Fatalf("expected symlink target %s, got %s", target, resolved)
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
