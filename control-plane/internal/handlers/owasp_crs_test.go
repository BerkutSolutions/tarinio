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

func (f *fakeOWASPCRSService) Status(ctx context.Context) (services.RuntimeCRSStatus, error) {
	return services.RuntimeCRSStatus{ActiveVersion: "4.0.0"}, nil
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

