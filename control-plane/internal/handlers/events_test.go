package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"waf/control-plane/internal/events"
)

type fakeEventService struct {
	items    []events.Event
	err      error
	probeErr error
}

func (f *fakeEventService) List() ([]events.Event, error) {
	if f.err != nil {
		return nil, f.err
	}
	return append([]events.Event(nil), f.items...), nil
}

func (f *fakeEventService) Probe() error {
	return f.probeErr
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

func TestEventsHandler_Probe(t *testing.T) {
	handler := NewEventsHandler(&fakeEventService{})
	req := httptest.NewRequest(http.MethodGet, "/api/events?probe=1", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}

func TestEventsHandler_UsesCachedEventsOnServiceFailure(t *testing.T) {
	now := time.Now().UTC().Format(time.RFC3339)
	service := &fakeEventService{
		items: []events.Event{
			{
				ID:              "evt-1",
				Type:            events.TypeApplySucceeded,
				Severity:        events.SeverityInfo,
				SourceComponent: "apply-runner",
				OccurredAt:      now,
				Summary:         "apply done",
			},
		},
	}
	handler := NewEventsHandler(service)

	firstReq := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	firstResp := httptest.NewRecorder()
	handler.ServeHTTP(firstResp, firstReq)
	if firstResp.Code != http.StatusOK {
		t.Fatalf("expected first response 200, got %d", firstResp.Code)
	}

	service.err = errors.New("boom")
	service.items = nil

	secondReq := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	secondResp := httptest.NewRecorder()
	handler.ServeHTTP(secondResp, secondReq)
	if secondResp.Code != http.StatusOK {
		t.Fatalf("expected cached response 200, got %d", secondResp.Code)
	}
}

func TestEventsHandler_AppliesFiltersToCachedFallback(t *testing.T) {
	service := &fakeEventService{items: []events.Event{
		{ID: "evt-b", Type: "apply_started", Severity: "info", SiteID: "site-b", OccurredAt: "2026-07-23T12:00:00Z"},
		{ID: "evt-a", Type: "apply_succeeded", Severity: "info", SiteID: "site-a", OccurredAt: "2026-07-23T13:00:00Z"},
	}}
	handler := NewEventsHandler(service)
	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/api/events", nil))
	if first.Code != http.StatusOK {
		t.Fatalf("seed cache status=%d body=%s", first.Code, first.Body.String())
	}
	service.err = errors.New("backend unavailable")
	filtered := httptest.NewRecorder()
	handler.ServeHTTP(filtered, httptest.NewRequest(http.MethodGet, "/api/events?type=apply_succeeded&site_id=site-a&limit=1", nil))
	var payload struct {
		Events []events.Event `json:"events"`
		Total  int            `json:"total"`
	}
	if err := json.NewDecoder(filtered.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if filtered.Code != http.StatusOK || payload.Total != 1 || len(payload.Events) != 1 || payload.Events[0].ID != "evt-a" {
		t.Fatalf("cached filter mismatch: status=%d payload=%+v", filtered.Code, payload)
	}
}

func TestEventsHandlerClampsLargePage(t *testing.T) {
	items := make([]events.Event, maxMonitoringEventsPageSize+20)
	for i := range items {
		items[i] = events.Event{ID: "evt", OccurredAt: time.Now().UTC().Format(time.RFC3339)}
	}
	handler := NewEventsHandler(&fakeEventService{items: items})
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/api/events?limit=999999", nil))
	var payload struct {
		Events []events.Event `json:"events"`
		Total  int            `json:"total"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Events) != maxMonitoringEventsPageSize || payload.Total != len(items) {
		t.Fatalf("unexpected pagination: got=%d total=%d", len(payload.Events), payload.Total)
	}
}

func TestEventsHandlerFiltersSortsAndPaginates(t *testing.T) {
	handler := NewEventsHandler(&fakeEventService{items: []events.Event{
		{ID: "evt-b", Type: events.TypeSecurityWAF, Severity: events.SeverityError, SiteID: "site-a", OccurredAt: "2026-07-23T12:00:00Z"},
		{ID: "evt-a", Type: events.TypeSecurityWAF, Severity: events.SeverityError, SiteID: "site-a", OccurredAt: "2026-07-23T13:00:00Z"},
		{ID: "evt-c", Type: events.TypeApplySucceeded, Severity: events.SeverityInfo, SiteID: "site-b", OccurredAt: "2026-07-23T14:00:00Z"},
	}})
	request := httptest.NewRequest(http.MethodGet, "/api/events?type=security&severity=error&site_id=site-a&from=2026-07-23T11:00:00Z&to=2026-07-23T14:00:00Z&limit=1&offset=0", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	var payload struct {
		Events []events.Event `json:"events"`
		Total  int            `json:"total"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK || payload.Total != 2 || len(payload.Events) != 1 || payload.Events[0].ID != "evt-a" {
		t.Fatalf("unexpected filtered page: code=%d payload=%+v", response.Code, payload)
	}
}

func TestEventsHandlerRejectsInvalidDateFilter(t *testing.T) {
	handler := NewEventsHandler(&fakeEventService{})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/events?from=invalid", nil))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", response.Code)
	}
}
