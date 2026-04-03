package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"waf/control-plane/internal/services"
)

type fakeReportService struct{}

func (f *fakeReportService) RevisionSummary() (services.ReportSummary, error) {
	return services.ReportSummary{ApplySuccessCount: 1}, nil
}

func TestReportsHandler_RevisionSummary(t *testing.T) {
	handler := NewReportsHandler(&fakeReportService{})

	req := httptest.NewRequest(http.MethodGet, "/api/reports/revisions", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}
