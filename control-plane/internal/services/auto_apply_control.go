package services

import "context"

type autoApplyDisabledContextKey struct{}

func withAutoApplyDisabled(ctx context.Context) context.Context {
	return context.WithValue(ctx, autoApplyDisabledContextKey{}, true)
}

func isAutoApplyDisabled(ctx context.Context) bool {
	disabled, _ := ctx.Value(autoApplyDisabledContextKey{}).(bool)
	return disabled
}
