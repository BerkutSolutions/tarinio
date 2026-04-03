package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"waf/control-plane/internal/jobs"
)

type fakeCertificateACMEService struct{}

func (f *fakeCertificateACMEService) Issue(ctx context.Context, certificateID string, commonName string, sanList []string) (jobs.Job, error) {
	return jobs.Job{ID: "job-a", Status: jobs.StatusSucceeded}, nil
}

func (f *fakeCertificateACMEService) Renew(ctx context.Context, certificateID string) (jobs.Job, error) {
	return jobs.Job{ID: "job-b", Status: jobs.StatusSucceeded}, nil
}

func TestCertificateACMEHandler_Issue(t *testing.T) {
	handler := NewCertificateACMEHandler(&fakeCertificateACMEService{})
	req := httptest.NewRequest(http.MethodPost, "/api/certificates/acme/issue", bytes.NewBufferString(`{"certificate_id":"cert-a","common_name":"example.com","san_list":["www.example.com"]}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.Code)
	}
}

func TestCertificateACMEHandler_Renew(t *testing.T) {
	handler := NewCertificateACMEHandler(&fakeCertificateACMEService{})
	req := httptest.NewRequest(http.MethodPost, "/api/certificates/acme/renew/cert-a", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.Code)
	}
}
