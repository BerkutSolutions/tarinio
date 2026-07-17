package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"waf/internal/observability"
)

func TestMetricsHandlerFailsClosedWithoutToken(t *testing.T) {
	response := httptest.NewRecorder()
	metricsHandler(observability.NewRegistry(), "").ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected unconfigured metrics endpoint to fail closed, got %d", response.Code)
	}
}

func TestMetricsHandlerRequiresConfiguredToken(t *testing.T) {
	handler := metricsHandler(observability.NewRegistry(), "metrics-secret")
	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	request.Header.Set(metricsTokenHeader, "metrics-secret")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected configured token to retain metrics access, got %d", response.Code)
	}
}
