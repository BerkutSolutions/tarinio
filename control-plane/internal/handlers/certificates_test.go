package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"waf/control-plane/internal/certificates"
)

type fakeCertificateService struct {
	items []certificates.Certificate
}

func (f *fakeCertificateService) Create(ctx context.Context, item certificates.Certificate) (certificates.Certificate, error) {
	item.CreatedAt = "2026-04-01T00:00:00Z"
	item.UpdatedAt = "2026-04-01T00:00:00Z"
	f.items = append(f.items, item)
	return item, nil
}

func (f *fakeCertificateService) List() ([]certificates.Certificate, error) {
	return append([]certificates.Certificate(nil), f.items...), nil
}

func (f *fakeCertificateService) Update(ctx context.Context, item certificates.Certificate) (certificates.Certificate, error) {
	item.UpdatedAt = "2026-04-01T01:00:00Z"
	return item, nil
}

func (f *fakeCertificateService) Delete(ctx context.Context, id string) error {
	return nil
}

func TestCertificatesHandler_CreateAndList(t *testing.T) {
	handler := NewCertificatesHandler(&fakeCertificateService{})

	createReq := httptest.NewRequest(http.MethodPost, "/api/certificates", bytes.NewBufferString(`{"id":"cert-a","common_name":"example.com","san_list":["www.example.com"],"not_before":"2026-04-01T00:00:00Z","not_after":"2026-10-01T00:00:00Z","status":"active"}`))
	createResp := httptest.NewRecorder()
	handler.ServeHTTP(createResp, createReq)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", createResp.Code)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/certificates", nil)
	listResp := httptest.NewRecorder()
	handler.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", listResp.Code)
	}
}

func TestCertificatesHandler_Delete(t *testing.T) {
	handler := NewCertificatesHandler(&fakeCertificateService{})
	req := httptest.NewRequest(http.MethodDelete, "/api/certificates/cert-a", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.Code)
	}
}
