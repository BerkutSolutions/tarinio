package httpserver

import (
	"net/http"

	"waf/control-plane/internal/auth"
	"waf/control-plane/internal/handlers"
	"waf/control-plane/internal/rbac"
)

type authenticator interface {
	Authenticate(sessionID string) (auth.SessionView, error)
	RequirePermission(session auth.SessionView, permission rbac.Permission) bool
}

func withAuth(authService authenticator, permission rbac.Permission, next http.Handler) http.Handler {
	return withMethodPermissions(authService, map[string]rbac.Permission{
		http.MethodGet:    permission,
		http.MethodPost:   permission,
		http.MethodPut:    permission,
		http.MethodDelete: permission,
	}, next)
}

func withMethodPermissions(authService authenticator, permissions map[string]rbac.Permission, next http.Handler) http.Handler {
	methodRequirements := make(map[string][]rbac.Permission, len(permissions))
	for method, permission := range permissions {
		if permission == "" {
			methodRequirements[method] = nil
			continue
		}
		methodRequirements[method] = []rbac.Permission{permission}
	}
	return withMethodAllPermissions(authService, methodRequirements, next)
}

func withMethodAllPermissions(authService authenticator, permissions map[string][]rbac.Permission, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, status, ok := authenticateRequestSession(r, authService)
		if !ok {
			handlers.ClearSessionCookie(w)
			handlers.WriteJSON(w, http.StatusUnauthorized, map[string]any{"error": status})
			return
		}
		for _, permission := range permissions[r.Method] {
			if permission == "" {
				continue
			}
			if !authService.RequirePermission(session, permission) {
				handlers.WriteJSON(w, http.StatusForbidden, map[string]any{"error": "permission denied"})
				return
			}
		}
		handlers.SetSessionCookieForRequest(w, r, session.SessionID)
		next.ServeHTTP(w, r.WithContext(auth.ContextWithSession(r.Context(), session)))
	})
}

func authenticateRequestSession(r *http.Request, authService authenticator) (auth.SessionView, string, bool) {
	candidates := [][2]string{
		{handlers.SessionCookieName, handlers.SessionBootCookieName},
		{handlers.SessionHTTPBootstrapCookieName, handlers.SessionHTTPBootstrapBootCookieName},
	}
	bootMismatch := false
	for _, candidate := range candidates {
		cookie, cookieErr := r.Cookie(candidate[0])
		bootCookie, bootErr := r.Cookie(candidate[1])
		if cookieErr != nil || bootErr != nil || cookie.Value == "" || bootCookie.Value == "" {
			continue
		}
		if bootCookie.Value != handlers.SessionBootToken() {
			bootMismatch = true
			continue
		}
		session, err := authService.Authenticate(cookie.Value)
		if err == nil {
			return session, "", true
		}
	}
	if bootMismatch {
		return auth.SessionView{}, "session expired after restart", false
	}
	return auth.SessionView{}, "authentication required", false
}
