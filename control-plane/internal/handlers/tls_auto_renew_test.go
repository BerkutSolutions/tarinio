package handlers

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"waf/control-plane/internal/services"
)

type fakeTLSAutoRenewService struct {
	settings services.TLSAutoRenewSettings
	err      error
}

func (f *fakeTLSAutoRenewService) Settings() (services.TLSAutoRenewSettings, error) {
	if f.err != nil {
		return services.TLSAutoRenewSettings{}, f.err
	}
	return f.settings, nil
}

func (f *fakeTLSAutoRenewService) UpdateSettings(input services.TLSAutoRenewSettings) (services.TLSAutoRenewSettings, error) {
	if f.err != nil {
		return services.TLSAutoRenewSettings{}, f.err
	}
	f.settings.Enabled = input.Enabled
	f.settings.RenewBeforeDays = input.RenewBeforeDays
	return f.settings, nil
}

func TestTLSAutoRenewHandler_Get(t *testing.T) {
	h := NewTLSAutoRenewHandler(&fakeTLSAutoRenewService{settings: services.TLSAutoRenewSettings{Enabled: true, RenewBeforeDays: 20}})
	req := httptest.NewRequest(http.MethodGet, "/api/tls/auto-renew", nil)
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}

func TestTLSAutoRenewHandler_Put(t *testing.T) {
	h := NewTLSAutoRenewHandler(&fakeTLSAutoRenewService{settings: services.TLSAutoRenewSettings{Enabled: false, RenewBeforeDays: 30}})
	req := httptest.NewRequest(http.MethodPut, "/api/tls/auto-renew", bytes.NewBufferString(`{"enabled":true,"renew_before_days":15}`))
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}

func TestTLSAutoRenewHandler_BadPayload(t *testing.T) {
	h := NewTLSAutoRenewHandler(&fakeTLSAutoRenewService{})
	req := httptest.NewRequest(http.MethodPut, "/api/tls/auto-renew", bytes.NewBufferString(`{"enabled":"yes","renew_before_days":15}`))
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.Code)
	}
}

func TestTLSAutoRenewHandler_Error(t *testing.T) {
	h := NewTLSAutoRenewHandler(&fakeTLSAutoRenewService{err: errors.New("boom")})
	req := httptest.NewRequest(http.MethodGet, "/api/tls/auto-renew", nil)
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.Code)
	}
}
