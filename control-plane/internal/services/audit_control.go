package services

import "context"

type auditDisabledContextKey struct{}

// WithAuditDisabled disables audit side effects for operations executed within
// the provided context.
func WithAuditDisabled(ctx context.Context) context.Context {
	return withAuditDisabled(ctx)
}

func withAuditDisabled(ctx context.Context) context.Context {
	return context.WithValue(ctx, auditDisabledContextKey{}, true)
}

func isAuditDisabled(ctx context.Context) bool {
	disabled, _ := ctx.Value(auditDisabledContextKey{}).(bool)
	return disabled
}
