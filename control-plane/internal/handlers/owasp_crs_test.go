package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"waf/control-plane/internal/services"
)

type fakeOWASPCRSService struct{}

type failingOWASPCRSService struct{ fakeOWASPCRSService }

func (f *failingOWASPCRSService) CheckUpdates(ctx context.Context, dryRun bool) (services.RuntimeCRSStatus, error) {
	return services.RuntimeCRSStatus{}, &services.RuntimeCRSError{Code: "crs_release_digest_invalid", Message: "internal detail"}
}

func (f *fakeOWASPCRSService) Status(ctx context.Context) (services.RuntimeCRSStatus, error) {
	return services.RuntimeCRSStatus{ActiveVersion: "4.0.0"}, nil
}

func TestOWASPCRSHandler_MapsRuntimeErrorCode(t *testing.T) {
	handler := NewOWASPCRSHandler(&failingOWASPCRSService{})
	req := httptest.NewRequest(http.MethodPost, "/api/owasp-crs/check-updates", strings.NewReader(`{"dry_run":true}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", resp.Code)
	}
	if !strings.Contains(resp.Body.String(), `"code":"crs_release_digest_invalid"`) {
		t.Fatalf("expected stable code, got %s", resp.Body.String())
	}
	if strings.Contains(resp.Body.String(), "internal detail") {
		t.Fatalf("internal runtime detail must not reach UI: %s", resp.Body.String())
	}
}

func (f *fakeOWASPCRSService) CheckUpdates(ctx context.Context, dryRun bool) (services.RuntimeCRSStatus, error) {
	return services.RuntimeCRSStatus{LatestVersion: "4.1.0", HasUpdate: true}, nil
}

func (f *fakeOWASPCRSService) Update(ctx context.Context) (services.RuntimeCRSStatus, error) {
	return services.RuntimeCRSStatus{ActiveVersion: "4.1.0"}, nil
}

func (f *fakeOWASPCRSService) SetHourlyAutoUpdate(ctx context.Context, enabled bool) (services.RuntimeCRSStatus, error) {
	return services.RuntimeCRSStatus{HourlyAutoUpdateEnabled: enabled}, nil
}

func TestOWASPCRSHandler_StatusAndUpdate(t *testing.T) {
	handler := NewOWASPCRSHandler(&fakeOWASPCRSService{})

	statusReq := httptest.NewRequest(http.MethodGet, "/api/owasp-crs/status", nil)
	statusResp := httptest.NewRecorder()
	handler.ServeHTTP(statusResp, statusReq)
	if statusResp.Code != http.StatusOK {
		t.Fatalf("expected status 200 for status endpoint, got %d", statusResp.Code)
	}
	if !strings.Contains(statusResp.Body.String(), "\"active_version\":\"4.0.0\"") {
		t.Fatalf("unexpected status response body: %s", statusResp.Body.String())
	}

	updateReq := httptest.NewRequest(http.MethodPost, "/api/owasp-crs/update", strings.NewReader(`{}`))
	updateResp := httptest.NewRecorder()
	handler.ServeHTTP(updateResp, updateReq)
	if updateResp.Code != http.StatusOK {
		t.Fatalf("expected status 200 for update endpoint, got %d", updateResp.Code)
	}
	if !strings.Contains(updateResp.Body.String(), "\"active_version\":\"4.1.0\"") {
		t.Fatalf("unexpected update response body: %s", updateResp.Body.String())
	}
}
