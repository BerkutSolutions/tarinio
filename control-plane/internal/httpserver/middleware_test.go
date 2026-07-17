package httpserver

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"waf/control-plane/internal/auth"
	"waf/control-plane/internal/handlers"
	"waf/control-plane/internal/rbac"
)

type middlewareAuthenticator struct{}

func (middlewareAuthenticator) Authenticate(sessionID string) (auth.SessionView, error) {
	if sessionID != "fresh-http-bootstrap-session" {
		return auth.SessionView{}, errors.New("session not found")
	}
	return auth.SessionView{SessionID: sessionID, UserID: "admin"}, nil
}

func (middlewareAuthenticator) RequirePermission(auth.SessionView, rbac.Permission) bool { return true }

func TestMiddlewareAcceptsFreshHTTPBootstrapCookiesAfterStaleHTTPSCookies(t *testing.T) {
	previousToken := handlers.SessionBootToken()
	handlers.SetSessionBootToken("fresh-bootstrap-token")
	t.Cleanup(func() { handlers.SetSessionBootToken(previousToken) })

	handler := withAuth(middlewareAuthenticator{}, "", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/api/sites", nil)
	request.AddCookie(&http.Cookie{Name: handlers.SessionCookieName, Value: "stale-https-session"})
	request.AddCookie(&http.Cookie{Name: handlers.SessionBootCookieName, Value: "stale-https-token"})
	request.AddCookie(&http.Cookie{Name: handlers.SessionHTTPBootstrapCookieName, Value: "fresh-http-bootstrap-session"})
	request.AddCookie(&http.Cookie{Name: handlers.SessionHTTPBootstrapBootCookieName, Value: "fresh-bootstrap-token"})
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("fresh HTTP bootstrap session must win over stale HTTPS cookies, got %d: %s", response.Code, response.Body.String())
	}
}
