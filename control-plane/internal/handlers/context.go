package handlers

import (
	"context"
	"net/http"
	"strings"

	"waf/control-plane/internal/audits"
	"waf/control-plane/internal/services"
)

func withActorIP(r *http.Request) context.Context {
	ctx := audits.ContextWithActorIP(r.Context(), audits.NormalizeRemoteIP(r.RemoteAddr))
	if autoApplyDisabled(r) {
		ctx = services.WithAutoApplyDisabled(ctx)
	}
	return ctx
}

func autoApplyDisabled(r *http.Request) bool {
	if r == nil {
		return false
	}
	if raw := strings.TrimSpace(r.Header.Get("X-WAF-Auto-Apply-Disabled")); raw != "" {
		switch strings.ToLower(raw) {
		case "1", "true", "yes", "on":
			return true
		}
	}
	switch strings.ToLower(strings.TrimSpace(r.URL.Query().Get("auto_apply"))) {
	case "0", "false", "no", "off":
		return true
	}
	return false
}
