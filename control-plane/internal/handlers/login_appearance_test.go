package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoginAppearanceSettingAndPreview(t *testing.T) {
	runtimeSettingsState.mu.Lock()
	previous := runtimeSettingsState.loginAppearance
	previousPath := runtimeSettingsState.statePath
	runtimeSettingsState.loginAppearance = defaultLoginAppearance
	runtimeSettingsState.statePath = ""
	runtimeSettingsState.mu.Unlock()
	t.Cleanup(func() {
		runtimeSettingsState.mu.Lock()
		runtimeSettingsState.loginAppearance = previous
		runtimeSettingsState.statePath = previousPath
		runtimeSettingsState.mu.Unlock()
	})

	for _, appearance := range []string{"command-center", "incident-console", "command-center-classic", "security-card", "incident-console-classic"} {
		request := httptest.NewRequest(http.MethodPut, "/api/settings/runtime", bytes.NewBufferString(`{"login_appearance":"`+appearance+`"}`))
		response := httptest.NewRecorder()
		(&SettingsRuntimeHandler{}).ServeHTTP(response, request)
		if response.Code != http.StatusOK || CurrentLoginAppearance() != appearance {
			t.Fatalf("appearance was not saved: status=%d value=%q", response.Code, CurrentLoginAppearance())
		}
	}

	for _, appearance := range []string{"command-center", "incident-console", "command-center-classic", "security-card", "incident-console-classic"} {
		preview := httptest.NewRecorder()
		(&LoginAppearancePreviewHandler{}).ServeHTTP(preview, httptest.NewRequest(http.MethodGet, "/api/login-appearance/preview/"+appearance+"?screen=2fa", nil))
		if preview.Code != http.StatusOK || !strings.Contains(preview.Body.String(), "ключом доступа") || !strings.Contains(preview.Body.String(), "Код двухфакторной") || !strings.Contains(preview.Body.String(), `data-login-appearance="`+appearance+`"`) {
			t.Fatalf("2FA preview misses required controls for %s: status=%d body=%s", appearance, preview.Code, preview.Body.String())
		}
	}

	invalid := httptest.NewRecorder()
	(&SettingsRuntimeHandler{}).ServeHTTP(invalid, httptest.NewRequest(http.MethodPut, "/api/settings/runtime", bytes.NewBufferString(`{"login_appearance":"unknown"}`)))
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("unknown appearance status=%d, want %d", invalid.Code, http.StatusBadRequest)
	}
}
