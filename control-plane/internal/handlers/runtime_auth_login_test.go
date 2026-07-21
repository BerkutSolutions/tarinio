package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type fakeBasicAuthLoginRecorder struct {
	siteID, username string
	when             time.Time
}

func (f *fakeBasicAuthLoginRecorder) MarkBasicAuthLogin(siteID, username string, when time.Time) error {
	f.siteID, f.username, f.when = siteID, username, when
	return nil
}

func TestRuntimeBasicAuthLoginHandlerRequiresRuntimeToken(t *testing.T) {
	recorder := &fakeBasicAuthLoginRecorder{}
	handler := NewRuntimeBasicAuthLoginHandler("runtime-token", recorder)
	body := []byte(`{"site_id":"site-a","username":"alice","occurred_at":"2026-07-21T12:00:00Z"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/internal/runtime/basic-auth-login", bytes.NewReader(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("missing token status=%d", response.Code)
	}
	request = httptest.NewRequest(http.MethodPost, "/api/internal/runtime/basic-auth-login", bytes.NewReader(body))
	request.Header.Set(runtimeAuthHeader, "runtime-token")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent || recorder.siteID != "site-a" || recorder.username != "alice" {
		t.Fatalf("accepted callback state: status=%d recorder=%+v", response.Code, recorder)
	}
}
