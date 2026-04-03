package auth

import "context"

type contextKey string

const sessionContextKey contextKey = "auth-session"

type SessionView struct {
	SessionID   string   `json:"session_id"`
	UserID      string   `json:"user_id"`
	Username    string   `json:"username"`
	RoleIDs     []string `json:"role_ids"`
	Permissions []string `json:"permissions"`
}

func ContextWithSession(ctx context.Context, session SessionView) context.Context {
	return context.WithValue(ctx, sessionContextKey, session)
}

func SessionFromContext(ctx context.Context) (SessionView, bool) {
	value := ctx.Value(sessionContextKey)
	session, ok := value.(SessionView)
	return session, ok
}
