package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"waf/control-plane/internal/accesspolicies"
	"waf/control-plane/internal/sites"
)

type stubSiteService struct {
	createFn func(context.Context, sites.Site) (sites.Site, error)
	listFn   func() ([]sites.Site, error)
	updateFn func(context.Context, sites.Site) (sites.Site, error)
	deleteFn func(context.Context, string) error
}

func (s stubSiteService) Create(ctx context.Context, site sites.Site) (sites.Site, error) {
	if s.createFn != nil {
		return s.createFn(ctx, site)
	}
	return sites.Site{}, nil
}

func (s stubSiteService) List() ([]sites.Site, error) {
	if s.listFn != nil {
		return s.listFn()
	}
	return nil, nil
}

func (s stubSiteService) Update(ctx context.Context, site sites.Site) (sites.Site, error) {
	if s.updateFn != nil {
		return s.updateFn(ctx, site)
	}
	return site, nil
}

func (s stubSiteService) Delete(ctx context.Context, id string) error {
	if s.deleteFn != nil {
		return s.deleteFn(ctx, id)
	}
	return nil
}

type stubSiteBanService struct{}

func (stubSiteBanService) Ban(context.Context, string, string) (accesspolicies.AccessPolicy, error) {
	return accesspolicies.AccessPolicy{}, nil
}

func (stubSiteBanService) Unban(context.Context, string, string) (accesspolicies.AccessPolicy, error) {
	return accesspolicies.AccessPolicy{}, nil
}

func TestSitesHandlerUpdateRejectsSiteIDRename(t *testing.T) {
	called := false
	handler := NewSitesHandler(stubSiteService{
		updateFn: func(ctx context.Context, site sites.Site) (sites.Site, error) {
			called = true
			return site, nil
		},
	}, stubSiteBanService{})

	body, err := json.Marshal(map[string]any{
		"id":           "panel.example.test",
		"primary_host": "panel.example.test",
		"enabled":      true,
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/api/sites/135.136.191.54", bytes.NewReader(body))
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", res.Code, res.Body.String())
	}
	if called {
		t.Fatal("site service update should not be called for rename payload")
	}
}
