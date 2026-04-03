package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeHealthService struct {
	count int
	err   error
}

func (f *fakeHealthService) RevisionCount() (int, error) {
	return f.count, f.err
}

func TestHealthHandler_OK(t *testing.T) {
	handler := NewHealthHandler(&fakeHealthService{count: 2})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}

func TestHealthHandler_Degraded(t *testing.T) {
	handler := NewHealthHandler(&fakeHealthService{err: errors.New("store down")})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", resp.Code)
	}
}
