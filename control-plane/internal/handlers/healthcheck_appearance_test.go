package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthcheckAppearanceSettingAndPublicRead(t *testing.T) {
	runtimeSettingsState.mu.Lock()
	previous := runtimeSettingsState.healthcheckAppearance
	previousPath := runtimeSettingsState.statePath
	runtimeSettingsState.healthcheckAppearance = defaultHealthcheckAppearance
	runtimeSettingsState.statePath = ""
	runtimeSettingsState.mu.Unlock()
	t.Cleanup(func() {
		runtimeSettingsState.mu.Lock()
		runtimeSettingsState.healthcheckAppearance = previous
		runtimeSettingsState.statePath = previousPath
		runtimeSettingsState.mu.Unlock()
	})

	for _, appearance := range []string{"variant-1", "variant-2", "variant-3", "variant-4", "variant-5"} {
		request := httptest.NewRequest(http.MethodPut, "/api/settings/runtime", bytes.NewBufferString(`{"healthcheck_appearance":"`+appearance+`"}`))
		response := httptest.NewRecorder()
		(&SettingsRuntimeHandler{}).ServeHTTP(response, request)
		if response.Code != http.StatusOK || CurrentHealthcheckAppearance() != appearance {
			t.Fatalf("appearance was not saved: status=%d value=%q", response.Code, CurrentHealthcheckAppearance())
		}
	}

	response := httptest.NewRecorder()
	(&HealthcheckAppearanceHandler{}).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/public/healthcheck-appearance", nil))
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"healthcheck_appearance":"variant-5"`)) {
		t.Fatalf("unexpected public appearance response: status=%d body=%s", response.Code, response.Body.String())
	}

	invalid := httptest.NewRecorder()
	(&SettingsRuntimeHandler{}).ServeHTTP(invalid, httptest.NewRequest(http.MethodPut, "/api/settings/runtime", bytes.NewBufferString(`{"healthcheck_appearance":"unknown"}`)))
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid appearance status=%d, want %d", invalid.Code, http.StatusBadRequest)
	}
}
