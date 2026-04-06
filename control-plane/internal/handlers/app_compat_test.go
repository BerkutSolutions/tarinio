package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppCompatHandler_ReportOKWithoutLegacyPending(t *testing.T) {
	runtimeRoot := t.TempDir()
	revisionStoreDir := filepath.Join(runtimeRoot, "control-plane")
	if err := os.MkdirAll(revisionStoreDir, 0o755); err != nil {
		t.Fatalf("mkdir revision store: %v", err)
	}
	handler := NewAppCompatHandler(runtimeRoot, revisionStoreDir)
	req := httptest.NewRequest(http.MethodGet, "/api/app/compat", nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
	body := resp.Body.String()
	if !strings.Contains(body, `"module_id":"dashboard"`) {
		t.Fatalf("expected compat modules in response, got: %s", body)
	}
	if strings.Contains(body, `"status":"needs_attention"`) {
		t.Fatalf("expected no pending status, got: %s", body)
	}
}

func TestAppCompatHandler_ReportNeedsAttentionWhenLegacyPending(t *testing.T) {
	runtimeRoot := t.TempDir()
	legacySites := filepath.Join(runtimeRoot, "data", "control-plane", "sites")
	if err := os.MkdirAll(legacySites, 0o755); err != nil {
		t.Fatalf("mkdir legacy path: %v", err)
	}
	revisionStoreDir := filepath.Join(runtimeRoot, "control-plane")
	if err := os.MkdirAll(revisionStoreDir, 0o755); err != nil {
		t.Fatalf("mkdir revision store: %v", err)
	}
	handler := NewAppCompatHandler(runtimeRoot, revisionStoreDir)
	req := httptest.NewRequest(http.MethodGet, "/api/app/compat", nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
	body := resp.Body.String()
	if !strings.Contains(body, `"status":"needs_attention"`) {
		t.Fatalf("expected needs_attention in response, got: %s", body)
	}
}

func TestAppCompatHandler_FixModule(t *testing.T) {
	runtimeRoot := t.TempDir()
	legacySiteDir := filepath.Join(runtimeRoot, "data", "control-plane", "sites")
	if err := os.MkdirAll(legacySiteDir, 0o755); err != nil {
		t.Fatalf("mkdir legacy sites: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacySiteDir, "site-a.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write legacy payload: %v", err)
	}
	revisionStoreDir := filepath.Join(runtimeRoot, "control-plane")
	if err := os.MkdirAll(revisionStoreDir, 0o755); err != nil {
		t.Fatalf("mkdir revision store: %v", err)
	}
	handler := NewAppCompatHandler(runtimeRoot, revisionStoreDir)
	req := httptest.NewRequest(http.MethodPost, "/api/app/compat/fix", bytes.NewBufferString(`{"module_id":"sites"}`))
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", resp.Code, resp.Body.String())
	}
	if _, err := os.Stat(filepath.Join(revisionStoreDir, "sites", "site-a.json")); err != nil {
		t.Fatalf("expected transferred file, stat error: %v", err)
	}
	body := resp.Body.String()
	if !strings.Contains(body, `"ok":true`) {
		t.Fatalf("expected ok=true, got %s", body)
	}
}
