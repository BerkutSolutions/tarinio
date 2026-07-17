package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"waf/control-plane/internal/auth"
	"waf/control-plane/internal/services"
)

type fakeDashboardService struct {
	probeKind string
}

func (f *fakeDashboardService) Stats() (services.DashboardStats, error) {
	return services.DashboardStats{
		ServicesUp:   1,
		RequestsDay:  10,
		AttacksDay:   2,
		GeneratedAt:  "2026-01-01T00:00:00Z",
		ServicesDown: 0,
	}, nil
}

func (f *fakeDashboardService) StatsForActor(_ string) (services.DashboardStats, error) { return f.Stats() }

func (f *fakeDashboardService) StatsForActorWithProcessDetails(_ string, _ bool) (services.DashboardStats, error) {
	return f.Stats()
}

func (f *fakeDashboardService) Probe(kind string, _ url.Values) error {
	f.probeKind = kind
	return nil
}

func (f *fakeDashboardService) DismissServiceErrors(_ string, _ []string) {}

func TestDashboardHandler_Stats(t *testing.T) {
	handler := NewDashboardHandler(&fakeDashboardService{})

	req := httptest.NewRequest(http.MethodGet, "/api/dashboard/stats", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}

func TestDashboardHandler_DismissRequiresAuthenticatedActor(t *testing.T) {
	handler := NewDashboardHandler(&fakeDashboardService{})
	req := httptest.NewRequest(http.MethodDelete, "/api/dashboard/services/runtime/errors", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.Code)
	}
}

func TestDashboardHandler_DismissAcceptsAuthenticatedActor(t *testing.T) {
	handler := NewDashboardHandler(&fakeDashboardService{})
	req := httptest.NewRequest(http.MethodDelete, "/api/dashboard/services/runtime/errors", nil)
	req = req.WithContext(auth.ContextWithSession(req.Context(), auth.SessionView{UserID: "operator-1"}))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}

func TestDashboardCanReadProcessDetailsRequiresAdministrationRead(t *testing.T) {
	for _, check := range []struct {
		name        string
		permissions []string
		want        bool
	}{
		{name: "dashboard reader", permissions: []string{"dashboard.read"}, want: false},
		{name: "administrator", permissions: []string{"administration.read"}, want: true},
	} {
		t.Run(check.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/dashboard/stats", nil)
			req = req.WithContext(auth.ContextWithSession(req.Context(), auth.SessionView{UserID: "u1", Permissions: check.permissions}))
			if got := dashboardCanReadProcessDetails(req); got != check.want {
				t.Fatalf("expected %v, got %v", check.want, got)
			}
		})
	}
}

func TestDashboardHandler_Probe(t *testing.T) {
	service := &fakeDashboardService{}
	handler := NewDashboardHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/api/dashboard/stats?probe=requests", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	if service.probeKind != "requests" {
		t.Fatalf("expected probe kind requests, got %q", service.probeKind)
	}
}
