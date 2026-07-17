package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"waf/control-plane/internal/services"
)

type staticSetupStatusService struct{ status services.SetupStatus }

func (s staticSetupStatusService) Status() (services.SetupStatus, error) { return s.status, nil }

func TestSetupHandler_PublicResponseOnlyDisclosesBootstrapNeed(t *testing.T) {
	handler := NewSetupHandler(staticSetupStatusService{status: services.SetupStatus{
		NeedsBootstrap:    false,
		HasUsers:          true,
		HasSites:          true,
		HasActiveRevision: true,
	}})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/setup/status", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.Code)
	}
	body := response.Body.String()
	if !strings.Contains(body, `"needs_bootstrap":false`) {
		t.Fatalf("expected bootstrap state, got %s", body)
	}
	for _, privateField := range []string{"has_users", "has_sites", "has_active_revision"} {
		if strings.Contains(body, privateField) {
			t.Fatalf("public setup response leaked %s: %s", privateField, body)
		}
	}
}
