package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLivenessHandlerDoesNotDiscloseOperationalState(t *testing.T) {
	response := httptest.NewRecorder()
	NewLivenessHandler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if response.Code != http.StatusOK || strings.TrimSpace(response.Body.String()) != `{"status":"ok"}` {
		t.Fatalf("unexpected liveness response: status=%d body=%s", response.Code, response.Body.String())
	}
}
