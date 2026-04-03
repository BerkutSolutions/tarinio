package handlers

import (
	"net/http"
	"net/http/httptest"
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
