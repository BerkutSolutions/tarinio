package handlers

import (
	"context"
	"net/http"

	"waf/control-plane/internal/audits"
)

func withActorIP(r *http.Request) context.Context {
	return audits.ContextWithActorIP(r.Context(), audits.NormalizeRemoteIP(r.RemoteAddr))
}
