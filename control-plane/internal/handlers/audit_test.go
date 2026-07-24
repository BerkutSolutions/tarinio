package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"waf/control-plane/internal/audits"
)

type fakeAuditService struct{}

func (f *fakeAuditService) List(query audits.Query) (audits.ListResult, error) {
	return audits.ListResult{Items: []audits.AuditEvent{{ID: "a1"}}, Total: 1, Limit: query.Limit, Offset: query.Offset}, nil
}

func TestAuditHandler_List(t *testing.T) {
	handler := NewAuditHandler(&fakeAuditService{})

	req := httptest.NewRequest(http.MethodGet, "/api/audit?action=site.create&limit=10", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}

func TestAuditHandlerValidatesAndClampsQuery(t *testing.T) {
	handler := NewAuditHandler(&fakeAuditService{})
	for _, path := range []string{"/api/audit?from=invalid", "/api/audit?to=invalid", "/api/audit?status=unknown", "/api/audit?offset=-1"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("%s: expected 400, got %d", path, response.Code)
		}
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/audit?limit=999999", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"limit":500`) {
		t.Fatalf("expected clamped limit, got %d %s", response.Code, response.Body.String())
	}
}
