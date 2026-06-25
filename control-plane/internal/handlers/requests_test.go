package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

type fakeRequestCollector struct {
	items     []map[string]any
	err       error
	probeErr  error
	probeRuns int
}

func (f *fakeRequestCollector) Collect() ([]map[string]any, error) {
	if f.err != nil {
		return nil, f.err
	}
	return append([]map[string]any(nil), f.items...), nil
}

func (f *fakeRequestCollector) CollectWithOptions(_ url.Values) ([]map[string]any, error) {
	return f.Collect()
}

func (f *fakeRequestCollector) Probe(_ url.Values) error {
	f.probeRuns++
	return f.probeErr
}

func TestRequestsHandler_Probe(t *testing.T) {
	collector := &fakeRequestCollector{}
	handler := NewRequestsHandler(collector)
	req := httptest.NewRequest(http.MethodGet, "/api/requests?probe=1", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	if collector.probeRuns != 1 {
		t.Fatalf("expected probe to run once, got %d", collector.probeRuns)
	}
}

func TestRequestsHandler_UsesCachedRowsOnCollectorFailure(t *testing.T) {
	collector := &fakeRequestCollector{
		items: []map[string]any{
			{"id": "req-1"},
		},
	}
	handler := NewRequestsHandler(collector)

	firstReq := httptest.NewRequest(http.MethodGet, "/api/requests?limit=5", nil)
	firstResp := httptest.NewRecorder()
	handler.ServeHTTP(firstResp, firstReq)
	if firstResp.Code != http.StatusOK {
		t.Fatalf("expected first response 200, got %d", firstResp.Code)
	}

	collector.err = errors.New("boom")
	collector.items = nil

	secondReq := httptest.NewRequest(http.MethodGet, "/api/requests?limit=5", nil)
	secondResp := httptest.NewRecorder()
	handler.ServeHTTP(secondResp, secondReq)
	if secondResp.Code != http.StatusOK {
		t.Fatalf("expected cached response 200, got %d", secondResp.Code)
	}
}
