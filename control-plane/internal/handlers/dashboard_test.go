package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"waf/control-plane/internal/services"
)

type fakeDashboardService struct{}

func (f *fakeDashboardService) Stats() (services.DashboardStats, error) {
	return services.DashboardStats{
		ServicesUp:   1,
		RequestsDay:  10,
		AttacksDay:   2,
		GeneratedAt:  "2026-01-01T00:00:00Z",
		ServicesDown: 0,
	}, nil
}

func TestDashboardHandler_Stats(t *testing.T) {
	handler := NewDashboardHandler(&fakeDashboardService{})

	req := httptest.NewRequest(http.MethodGet, "/api/dashboard/stats", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}
