package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"waf/control-plane/internal/rbac"
	"waf/control-plane/internal/revisionsnapshots"
	"waf/control-plane/internal/roles"
	"waf/control-plane/internal/services"
	"waf/control-plane/internal/users"
)

type fakeHealthService struct {
	count int
	err   error
}

func (f *fakeHealthService) RevisionCount() (int, error) { return f.count, f.err }

type fakeHealthCatalogService struct {
	probe services.RevisionCatalogProbe
	err   error
}

func (f *fakeHealthCatalogService) Probe(_ context.Context) (services.RevisionCatalogProbe, error) {
	return f.probe, f.err
}

type fakeHealthSetupService struct {
	status services.SetupStatus
	err    error
}

func (f *fakeHealthSetupService) Status() (services.SetupStatus, error) { return f.status, f.err }

type fakeHealthSessionStore struct {
	count int
	err   error
}

func (f *fakeHealthSessionStore) Count() (int, error) { return f.count, f.err }

type fakeHealthUserStore struct {
	count int
	items []users.User
	err   error
}

func (f *fakeHealthUserStore) Count() (int, error) { return f.count, f.err }
func (f *fakeHealthUserStore) List() ([]users.User, error) {
	return append([]users.User(nil), f.items...), f.err
}

type fakeHealthRoleStore struct {
	items []roles.Role
	err   error
}

func (f *fakeHealthRoleStore) List() ([]roles.Role, error) {
	return append([]roles.Role(nil), f.items...), f.err
}

type fakeHealthCompiler struct{ err error }

func (f *fakeHealthCompiler) Preview() (revisionsnapshots.Snapshot, error) {
	return revisionsnapshots.Snapshot{}, f.err
}

type fakeSimpleProbe struct{ err error }

func (f *fakeSimpleProbe) Probe() error { return f.err }

type fakeRequestProbe struct{ err error }

func (f *fakeRequestProbe) Probe(query url.Values) error { return f.err }

type fakeCRSStatusService struct {
	status services.RuntimeCRSStatus
	err    error
}

func (f *fakeCRSStatusService) Status(ctx context.Context) (services.RuntimeCRSStatus, error) {
	return f.status, f.err
}

func TestHealthHandler_OK(t *testing.T) {
	handler := NewHealthHandler(
		&fakeHealthService{count: 2},
		&fakeHealthCatalogService{probe: services.RevisionCatalogProbe{ServiceCount: 1, RevisionCount: 2, TimelineCount: 3}},
		&fakeHealthSetupService{status: services.SetupStatus{HasUsers: true, HasSites: true, HasActiveRevision: true}},
		&fakeHealthSessionStore{count: 3},
		&fakeHealthUserStore{count: 1, items: []users.User{{ID: "admin", Username: "admin", IsActive: true, RoleIDs: []string{"admin"}}}},
		&fakeHealthRoleStore{items: []roles.Role{
			{ID: "admin", Permissions: rbac.AllPermissions()},
			{ID: "auditor"},
			{ID: "manager"},
			{ID: "soc"},
		}},
		&fakeHealthCompiler{},
		&fakeSimpleProbe{},
		&fakeSimpleProbe{},
		&fakeRequestProbe{},
		&fakeCRSStatusService{status: services.RuntimeCRSStatus{ActiveVersion: "4.0.0"}},
	)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}

func TestHealthHandler_DegradedOnRevisionStore(t *testing.T) {
	handler := NewHealthHandler(
		&fakeHealthService{err: errors.New("store down")},
		&fakeHealthCatalogService{},
		nil, nil, nil, nil, nil, nil, nil, nil, nil,
	)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", resp.Code)
	}
}

func TestHealthHandler_DegradedOnOperationalProbeFailure(t *testing.T) {
	handler := NewHealthHandler(
		&fakeHealthService{count: 2},
		&fakeHealthCatalogService{probe: services.RevisionCatalogProbe{ServiceCount: 1, RevisionCount: 2, TimelineCount: 3}},
		&fakeHealthSetupService{status: services.SetupStatus{HasUsers: true, HasSites: true, HasActiveRevision: true}},
		&fakeHealthSessionStore{count: 1},
		&fakeHealthUserStore{count: 1, items: []users.User{{ID: "admin", Username: "admin", IsActive: true, RoleIDs: []string{"admin"}}}},
		&fakeHealthRoleStore{items: []roles.Role{
			{ID: "admin", Permissions: rbac.AllPermissions()},
			{ID: "auditor"},
			{ID: "manager"},
			{ID: "soc"},
		}},
		&fakeHealthCompiler{},
		&fakeSimpleProbe{err: errors.New("runtime down")},
		&fakeSimpleProbe{},
		&fakeRequestProbe{},
		&fakeCRSStatusService{status: services.RuntimeCRSStatus{ActiveVersion: "4.0.0"}},
	)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["status"] != "degraded" {
		t.Fatalf("expected degraded payload status, got %#v", payload["status"])
	}
}
