package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"waf/control-plane/internal/tlsconfigs"
)

type fakeTLSConfigService struct {
	items []tlsconfigs.TLSConfig
}

func (f *fakeTLSConfigService) Create(ctx context.Context, item tlsconfigs.TLSConfig) (tlsconfigs.TLSConfig, error) {
	item.CreatedAt = "2026-04-01T00:00:00Z"
	item.UpdatedAt = "2026-04-01T00:00:00Z"
	f.items = append(f.items, item)
	return item, nil
}

func (f *fakeTLSConfigService) List() ([]tlsconfigs.TLSConfig, error) {
	return append([]tlsconfigs.TLSConfig(nil), f.items...), nil
}

func (f *fakeTLSConfigService) Update(ctx context.Context, item tlsconfigs.TLSConfig) (tlsconfigs.TLSConfig, error) {
	item.UpdatedAt = "2026-04-01T01:00:00Z"
	return item, nil
}

func (f *fakeTLSConfigService) Delete(ctx context.Context, siteID string) error {
	return nil
}

func TestTLSConfigsHandler_CreateAndList(t *testing.T) {
	handler := NewTLSConfigsHandler(&fakeTLSConfigService{})

	createReq := httptest.NewRequest(http.MethodPost, "/api/tls-configs", bytes.NewBufferString(`{"site_id":"site-a","certificate_id":"cert-a"}`))
	createResp := httptest.NewRecorder()
	handler.ServeHTTP(createResp, createReq)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", createResp.Code)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/tls-configs", nil)
	listResp := httptest.NewRecorder()
	handler.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", listResp.Code)
	}
}

func TestTLSConfigsHandler_Delete(t *testing.T) {
	handler := NewTLSConfigsHandler(&fakeTLSConfigService{})
	req := httptest.NewRequest(http.MethodDelete, "/api/tls-configs/site-a", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.Code)
	}
}
