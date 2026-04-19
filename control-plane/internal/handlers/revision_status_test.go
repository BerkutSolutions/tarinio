package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeRevisionStatusService struct {
	cleared bool
}

func (f *fakeRevisionStatusService) ClearTimeline(ctx context.Context) error {
	f.cleared = true
	return nil
}

func TestRevisionStatusHandler_Clear(t *testing.T) {
	service := &fakeRevisionStatusService{}
	handler := NewRevisionStatusHandler(service)

	req := httptest.NewRequest(http.MethodDelete, "/api/revisions/statuses", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	if !service.cleared {
		t.Fatal("expected clear timeline to be called")
	}
}
