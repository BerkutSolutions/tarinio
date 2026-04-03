package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"waf/control-plane/internal/events"
)

type fakeEventService struct {
	items []events.Event
	err   error
}

func (f *fakeEventService) List() ([]events.Event, error) {
	if f.err != nil {
		return nil, f.err
	}
	return append([]events.Event(nil), f.items...), nil
}

func TestEventsHandler_Get(t *testing.T) {
	handler := NewEventsHandler(&fakeEventService{
		items: []events.Event{
			{
				ID:              "evt-1",
				Type:            events.TypeApplySucceeded,
				Severity:        events.SeverityInfo,
				SourceComponent: "apply-runner",
				OccurredAt:      "2026-04-02T00:00:00Z",
				Summary:         "apply done",
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}

func TestEventsHandler_NotFound(t *testing.T) {
	handler := NewEventsHandler(&fakeEventService{})
	req := httptest.NewRequest(http.MethodGet, "/api/events/unknown", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.Code)
	}
}

