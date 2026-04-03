package handlers

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"waf/control-plane/internal/upstreams"
)

type fakeUpstreamService struct {
	items []upstreams.Upstream
	err   error
}

func (f *fakeUpstreamService) Create(ctx context.Context, item upstreams.Upstream) (upstreams.Upstream, error) {
	if f.err != nil {
		return upstreams.Upstream{}, f.err
	}
	item.CreatedAt = "2026-04-01T00:00:00Z"
	item.UpdatedAt = "2026-04-01T00:00:00Z"
	f.items = append(f.items, item)
	return item, nil
}

func (f *fakeUpstreamService) List() ([]upstreams.Upstream, error) {
	if f.err != nil {
		return nil, f.err
	}
	return append([]upstreams.Upstream(nil), f.items...), nil
}

func (f *fakeUpstreamService) Update(ctx context.Context, item upstreams.Upstream) (upstreams.Upstream, error) {
	if f.err != nil {
		return upstreams.Upstream{}, f.err
	}
	item.UpdatedAt = "2026-04-01T01:00:00Z"
	return item, nil
}

func (f *fakeUpstreamService) Delete(ctx context.Context, id string) error {
	return f.err
}

func TestUpstreamsHandler_CreateAndList(t *testing.T) {
	handler := NewUpstreamsHandler(&fakeUpstreamService{})

	createReq := httptest.NewRequest(http.MethodPost, "/api/upstreams", bytes.NewBufferString(`{"id":"up-a","site_id":"site-a","host":"app.internal","port":8080,"scheme":"http"}`))
	createResp := httptest.NewRecorder()
	handler.ServeHTTP(createResp, createReq)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", createResp.Code)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/upstreams", nil)
	listResp := httptest.NewRecorder()
	handler.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", listResp.Code)
	}
}

func TestUpstreamsHandler_CreateSiteMissing(t *testing.T) {
	handler := NewUpstreamsHandler(&fakeUpstreamService{err: errors.New("site site-a not found")})
	req := httptest.NewRequest(http.MethodPost, "/api/upstreams", bytes.NewBufferString(`{"id":"up-a","site_id":"site-a","host":"app.internal","port":8080,"scheme":"http"}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}
