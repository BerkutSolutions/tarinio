package services

import "context"

type autoApplyDisabledContextKey struct{}

// WithAutoApplyDisabled disables compile/apply side effects for control-plane
// operations executed within the provided context.
func WithAutoApplyDisabled(ctx context.Context) context.Context {
	return withAutoApplyDisabled(ctx)
}

func withAutoApplyDisabled(ctx context.Context) context.Context {
	return context.WithValue(ctx, autoApplyDisabledContextKey{}, true)
}

func isAutoApplyDisabled(ctx context.Context) bool {
	disabled, _ := ctx.Value(autoApplyDisabledContextKey{}).(bool)
	return disabled
}
