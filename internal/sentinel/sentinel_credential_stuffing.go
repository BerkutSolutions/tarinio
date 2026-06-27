package sentinel

import "strings"

// defaultAuthPaths is the built-in list of authentication endpoint prefixes
// used by signal_credential_stuffing when no custom list is configured.
var defaultAuthPaths = []string{
	"/login",
	"/auth",
	"/signin",
	"/sign-in",
	"/api/auth",
	"/api/login",
	"/api/token",
	"/api/signin",
	"/account/login",
	"/user/login",
	"/users/sign_in",
	"/session",
	"/sessions",
}

// effectiveAuthPaths returns the operator-configured list or the built-in default.
func effectiveAuthPaths(cfg Config) []string {
	if len(cfg.AuthPaths) > 0 {
		return cfg.AuthPaths
	}
	return defaultAuthPaths
}

// isAuthPath reports whether the given path matches a known auth endpoint.
func isAuthPath(path string, authPaths []string) bool {
	p := normalizePath(path)
	if p == "" {
		return false
	}
	for _, prefix := range authPaths {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}
	return false
}

// isBadBehaviorHit reports whether the response status indicates a bad-behavior
// zone hit (429 from the dedicated bad-behavior nginx zone).
// We identify bad-behavior 429s by status code — the same status the nginx
// bad_behavior zone returns.  Future: add a dedicated log field.
func isBadBehaviorHit(status int) bool {
	return status == 429
}
