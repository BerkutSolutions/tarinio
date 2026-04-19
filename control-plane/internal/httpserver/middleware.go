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
		cookie, err := r.Cookie(handlers.SessionCookieName)
		if err != nil || cookie.Value == "" {
			handlers.ClearSessionCookie(w)
			handlers.WriteJSON(w, http.StatusUnauthorized, map[string]any{"error": "authentication required"})
			return
		}
		bootCookie, bootErr := r.Cookie(handlers.SessionBootCookieName)
		if bootErr != nil || bootCookie.Value == "" || bootCookie.Value != handlers.SessionBootToken() {
			handlers.ClearSessionCookie(w)
			handlers.WriteJSON(w, http.StatusUnauthorized, map[string]any{"error": "session expired after restart"})
			return
		}
		session, err := authService.Authenticate(cookie.Value)
		if err != nil {
			handlers.ClearSessionCookie(w)
			handlers.WriteJSON(w, http.StatusUnauthorized, map[string]any{"error": "authentication required"})
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
		handlers.SetSessionCookieForRequest(w, r, cookie.Value)
		next.ServeHTTP(w, r.WithContext(auth.ContextWithSession(r.Context(), session)))
	})
}
