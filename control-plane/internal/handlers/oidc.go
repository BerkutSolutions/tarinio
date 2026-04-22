package handlers

import (
	"context"
	"net/http"
	"net/url"

	"waf/control-plane/internal/services"
)

type oidcService interface {
	ProviderStatus() (services.OIDCProviderStatus, error)
	BeginOIDCLogin(nextPath string) (string, error)
	HandleOIDCCallback(ctx context.Context, state, code string) (services.OIDCCallbackResult, error)
}

type OIDCHandler struct {
	service oidcService
}

func NewOIDCHandler(service oidcService) *OIDCHandler {
	return &OIDCHandler{service: service}
}

func (h *OIDCHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/api/auth/providers" && r.Method == http.MethodGet:
		h.providers(w, r)
	case r.URL.Path == "/api/auth/oidc/start" && r.Method == http.MethodGet:
		h.start(w, r)
	case r.URL.Path == "/api/auth/oidc/callback" && r.Method == http.MethodGet:
		h.callback(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (h *OIDCHandler) providers(w http.ResponseWriter, _ *http.Request) {
	status, err := h.service.ProviderStatus()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"oidc": status,
	})
}

func (h *OIDCHandler) start(w http.ResponseWriter, r *http.Request) {
	redirectURL, err := h.service.BeginOIDCLogin(r.URL.Query().Get("next"))
	if err != nil {
		http.Redirect(w, r, "/login?reason="+url.QueryEscape(err.Error()), http.StatusFound)
		return
	}
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (h *OIDCHandler) callback(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.HandleOIDCCallback(withActorIP(r), r.URL.Query().Get("state"), r.URL.Query().Get("code"))
	if err != nil {
		http.Redirect(w, r, "/login?reason="+url.QueryEscape(err.Error()), http.StatusFound)
		return
	}
	SetSessionCookieForRequest(w, r, result.SessionID)
	http.Redirect(w, r, result.NextPath, http.StatusFound)
}
