package audits

import (
	"context"
	"net"
	"strings"
)

type contextKey string

const actorIPContextKey contextKey = "audit-actor-ip"

func ContextWithActorIP(ctx context.Context, ip string) context.Context {
	ip = normalizeIP(ip)
	if ip == "" {
		return ctx
	}
	return context.WithValue(ctx, actorIPContextKey, ip)
}

func ActorIPFromContext(ctx context.Context) string {
	value := ctx.Value(actorIPContextKey)
	ip, _ := value.(string)
	return strings.TrimSpace(ip)
}

func NormalizeRemoteIP(value string) string {
	return normalizeIP(value)
}

func normalizeIP(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	host, _, err := net.SplitHostPort(value)
	if err == nil {
		value = host
	}
	value = strings.TrimSpace(value)
	if net.ParseIP(value) == nil {
		return ""
	}
	return value
}
