package handlers

import (
	"encoding/json"
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
	count     int
	countErr  error
	countRuns int
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

func (f *fakeRequestCollector) CollectCount(_ url.Values) (int, error) {
	f.countRuns++
	if f.countErr != nil {
		return 0, f.countErr
	}
	if f.count > 0 {
		return f.count, nil
	}
	return len(f.items), nil
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

func TestRequestsHandler_CountUsesCollectorCounter(t *testing.T) {
	collector := &fakeRequestCollector{count: 17}
	handler := NewRequestsHandler(collector)
	req := httptest.NewRequest(http.MethodGet, "/api/requests?count=1", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	if collector.countRuns != 1 {
		t.Fatalf("expected count to run once, got %d", collector.countRuns)
	}
	if collector.probeRuns != 0 {
		t.Fatalf("expected probe not to run, got %d", collector.probeRuns)
	}
	var payload map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if got := payload["count"]; got != float64(17) {
		t.Fatalf("expected count=17, got %#v", got)
	}
}

func TestRequestsHandler_CountFallsBackToItemsLength(t *testing.T) {
	collector := &fakeRequestCollector{items: []map[string]any{{"id": "a"}, {"id": "b"}}}
	handler := NewRequestsHandler(struct{ requestCollector }{requestCollector: collector})
	req := httptest.NewRequest(http.MethodGet, "/api/requests?count=1", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if got := payload["count"]; got != float64(2) {
		t.Fatalf("expected count=2, got %#v", got)
	}
}

func TestRequestsHandler_UsesCachedRowsOnCollectorFailure(t *testing.T) {
	collector := &fakeRequestCollector{
		items: []map[string]any{{"id": "req-1"}},
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

func TestRequestsHandler_AddsRequestRowType(t *testing.T) {
	collector := &fakeRequestCollector{
		items: []map[string]any{
			{"stream": "runtime", "entry": map[string]any{"request_id": "req-1"}},
			{"stream": "security_waf", "entry": map[string]any{"request_id": "req-2"}},
		},
	}
	handler := NewRequestsHandler(collector)

	req := httptest.NewRequest(http.MethodGet, "/api/requests?limit=5", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	var rows []map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode rows: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if got := rows[0]["row_type"]; got != requestRowTypeRequest {
		t.Fatalf("expected first row_type=%q, got %#v", requestRowTypeRequest, got)
	}
	if got := rows[1]["row_type"]; got != requestRowTypeSecurity {
		t.Fatalf("expected second row_type=%q, got %#v", requestRowTypeSecurity, got)
	}
	if got := rows[0]["legacy_row_type_support"]; got != true {
		t.Fatalf("expected legacy_row_type_support=true, got %#v", got)
	}
}

func TestRequestsHandler_KeepsExplicitLegacyRowTypeCompatibility(t *testing.T) {
	collector := &fakeRequestCollector{
		items: []map[string]any{
			{"row_type": "security", "stream": "runtime", "entry": map[string]any{"request_id": "req-legacy-security"}},
			{"rowType": "request", "stream": "security_modsec", "entry": map[string]any{"request_id": "req-legacy-request"}},
		},
	}
	handler := NewRequestsHandler(collector)

	req := httptest.NewRequest(http.MethodGet, "/api/requests?limit=5", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	var rows []map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode rows: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if got := rows[0]["row_type"]; got != requestRowTypeSecurity {
		t.Fatalf("expected explicit row_type security to win, got %#v", got)
	}
	if got := rows[1]["row_type"]; got != requestRowTypeRequest {
		t.Fatalf("expected explicit rowType request to win, got %#v", got)
	}
}

func TestRequestsHandler_InfersLegacySecurityRowsWithoutRowType(t *testing.T) {
	collector := &fakeRequestCollector{
		items: []map[string]any{
			{"type": "modsecurity_sql_injection", "entry": map[string]any{"request_id": "req-modsec"}},
			{"source_component": "security_runtime", "entry": map[string]any{"request_id": "req-source"}},
		},
	}
	handler := NewRequestsHandler(collector)

	req := httptest.NewRequest(http.MethodGet, "/api/requests?limit=5", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	var rows []map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode rows: %v", err)
	}
	for idx, row := range rows {
		if got := row["row_type"]; got != requestRowTypeSecurity {
			t.Fatalf("expected inferred security row at %d, got %#v", idx, got)
		}
	}
}

func TestRequestsHandler_ExposesNormalizedSecurityCategoryFields(t *testing.T) {
	collector := &fakeRequestCollector{
		items: []map[string]any{
			{
				"stream": "security_access",
				"summary": "legacy summary should not win",
				"details": map[string]any{
					"event_type": "auth",
					"path":       "/auth/login",
				},
			},
			{
				"stream": "security_access",
				"type":   "modsecurity_sql_injection",
				"details": map[string]any{
					"path": "/checkout",
				},
			},
		},
	}
	handler := NewRequestsHandler(collector)

	req := httptest.NewRequest(http.MethodGet, "/api/requests?limit=5", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	var rows []map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode rows: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if got := rows[0]["event_type"]; got != "auth" {
		t.Fatalf("expected first event_type=auth, got %#v", got)
	}
	if got := rows[0]["security_reason"]; got != "auth" {
		t.Fatalf("expected first security_reason=auth, got %#v", got)
	}
	if got := rows[1]["security_reason"]; got != "modsecurity_sql_injection" {
		t.Fatalf("expected second security_reason from legacy fallback, got %#v", got)
	}
}
