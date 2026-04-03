package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"waf/control-plane/internal/accesspolicies"
	"waf/control-plane/internal/sites"
)

type fakeSiteService struct {
	items []sites.Site
}

func (f *fakeSiteService) Create(ctx context.Context, site sites.Site) (sites.Site, error) {
	site.CreatedAt = "2026-04-01T00:00:00Z"
	site.UpdatedAt = "2026-04-01T00:00:00Z"
	f.items = append(f.items, site)
	return site, nil
}

func (f *fakeSiteService) List() ([]sites.Site, error) {
	return append([]sites.Site(nil), f.items...), nil
}

func (f *fakeSiteService) Update(ctx context.Context, site sites.Site) (sites.Site, error) {
	site.UpdatedAt = "2026-04-01T01:00:00Z"
	return site, nil
}

func (f *fakeSiteService) Delete(ctx context.Context, id string) error {
	return nil
}

type fakeSiteBanService struct{}

func (f *fakeSiteBanService) Ban(ctx context.Context, siteID string, address string) (accesspolicies.AccessPolicy, error) {
	return accesspolicies.AccessPolicy{SiteID: siteID, DenyList: []string{address}}, nil
}

func (f *fakeSiteBanService) Unban(ctx context.Context, siteID string, address string) (accesspolicies.AccessPolicy, error) {
	return accesspolicies.AccessPolicy{SiteID: siteID}, nil
}

func TestSitesHandler_CreateAndList(t *testing.T) {
	handler := NewSitesHandler(&fakeSiteService{}, &fakeSiteBanService{})

	createReq := httptest.NewRequest(http.MethodPost, "/api/sites", bytes.NewBufferString(`{"id":"site-a","primary_host":"a.example.com","enabled":true}`))
	createResp := httptest.NewRecorder()
	handler.ServeHTTP(createResp, createReq)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", createResp.Code)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/sites", nil)
	listResp := httptest.NewRecorder()
	handler.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", listResp.Code)
	}
}

func TestSitesHandler_Delete(t *testing.T) {
	handler := NewSitesHandler(&fakeSiteService{}, &fakeSiteBanService{})
	req := httptest.NewRequest(http.MethodDelete, "/api/sites/site-a", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.Code)
	}
}

func TestSitesHandler_BanAndUnban(t *testing.T) {
	handler := NewSitesHandler(&fakeSiteService{}, &fakeSiteBanService{})

	banReq := httptest.NewRequest(http.MethodPost, "/api/sites/site-a/ban", bytes.NewBufferString(`{"ip":"10.0.0.1"}`))
	banResp := httptest.NewRecorder()
	handler.ServeHTTP(banResp, banReq)
	if banResp.Code != http.StatusOK {
		t.Fatalf("expected 200 for ban, got %d", banResp.Code)
	}

	unbanReq := httptest.NewRequest(http.MethodPost, "/api/sites/site-a/unban", bytes.NewBufferString(`{"ip":"10.0.0.1"}`))
	unbanResp := httptest.NewRecorder()
	handler.ServeHTTP(unbanResp, unbanReq)
	if unbanResp.Code != http.StatusOK {
		t.Fatalf("expected 200 for unban, got %d", unbanResp.Code)
	}
}
