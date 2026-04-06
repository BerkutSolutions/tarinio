package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAppPingHandler_OK(t *testing.T) {
	handler := NewAppPingHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/app/ping", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	body := resp.Body.String()
	if !strings.Contains(body, `"ok":true`) {
		t.Fatalf("expected ok=true, got: %s", body)
	}
}

func TestAppPingHandler_NotFound(t *testing.T) {
	handler := NewAppPingHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/app/ping", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.Code)
	}
}
