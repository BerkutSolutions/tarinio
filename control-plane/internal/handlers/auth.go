package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"waf/control-plane/internal/services"
)

const SessionCookieName = "waf_session"
const SessionBootCookieName = "waf_session_boot"
const sessionCookieMaxAgeSeconds = 12 * 60 * 60

var (
	sessionBootToken   = generateSessionBootToken()
	sessionBootTokenMu sync.RWMutex
	loginRateLimiter   = &authLoginRateLimiter{state: map[string]authLoginAttemptState{}}
)

type authLoginAttemptState struct {
	FailTimestamps []time.Time
	BlockedUntil   time.Time
}

type authLoginRateLimiter struct {
	mu    sync.Mutex
	state map[string]authLoginAttemptState
}

type authService interface {
	Bootstrap(ctx context.Context, username, email, password string) (services.SessionResult, error)
	Login(ctx context.Context, username, password string) (services.LoginResult, error)
	Login2FA(ctx context.Context, challengeID, code, recoveryCode string) (services.SessionResult, error)
	BeginPasskeyLogin(ctx context.Context, username string, req *http.Request) (services.PasskeyBeginResult, error)
	FinishPasskeyLogin(ctx context.Context, challengeID string, credentialJSON json.RawMessage, req *http.Request) (services.LoginResult, error)
	BeginPasskey2FA(ctx context.Context, loginChallengeID string, req *http.Request) (services.Passkey2FABeginResult, error)
	FinishPasskey2FA(ctx context.Context, loginChallengeID, webAuthnChallengeID string, credentialJSON json.RawMessage, req *http.Request) (services.SessionResult, error)
	Logout(ctx context.Context, sessionID string) error
	Me(sessionID string) (services.AuthUser, error)
	SetupTOTP(ctx context.Context, sessionID string) (services.TOTPSetupResult, error)
	EnableTOTP(ctx context.Context, sessionID, challengeID, code string) (services.TOTPEnableResult, error)
	DisableTOTP(ctx context.Context, sessionID, password, recoveryCode string) (services.AuthUser, error)
	ChangePassword(ctx context.Context, sessionID, currentPassword, password string) error
	ListPasskeys(sessionID string) (services.PasskeyListResult, error)
	BeginPasskeyRegister(ctx context.Context, sessionID, name string, req *http.Request) (services.PasskeyBeginResult, error)
	FinishPasskeyRegister(ctx context.Context, sessionID, challengeID, name string, credentialJSON json.RawMessage, req *http.Request) (services.PasskeyItem, error)
	RenamePasskey(sessionID, id, name string) (services.PasskeyItem, error)
	DeletePasskey(sessionID, id string) error
	UpdatePreferences(ctx context.Context, sessionID string, input services.AuthUserPreferences) (services.AuthUser, error)
}

type AuthHandler struct {
	auth authService
}

type loginRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type login2FARequest struct {
	ChallengeID  string `json:"challenge_id"`
	Code         string `json:"code"`
	RecoveryCode string `json:"recovery_code"`
}

type totpEnableRequest struct {
	ChallengeID string `json:"challenge_id"`
	Code        string `json:"code"`
}

type totpDisableRequest struct {
	Password     string `json:"password"`
	RecoveryCode string `json:"recovery_code"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	Password        string `json:"password"`
}

type passkeyFinishRequest struct {
	ChallengeID string          `json:"challenge_id"`
	Name        string          `json:"name"`
	Credential  json.RawMessage `json:"credential"`
}

type passkey2FABeginRequest struct {
	ChallengeID string `json:"challenge_id"`
}

type passkey2FAFinishRequest struct {
	ChallengeID       string          `json:"challenge_id"`
	WebAuthnChallenge string          `json:"webauthn_challenge_id"`
	Credential        json.RawMessage `json:"credential"`
}

type passkeyRenameRequest struct {
	Name string `json:"name"`
}

type updateMeRequest struct {
	Language string `json:"language"`
}

func NewAuthHandler(auth authService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func (h *AuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/api/auth/bootstrap" && r.Method == http.MethodPost:
		h.bootstrap(w, r)
	case r.URL.Path == "/api/auth/login" && r.Method == http.MethodPost:
		h.login(w, r)
	case r.URL.Path == "/api/auth/login/2fa" && r.Method == http.MethodPost:
		h.login2FA(w, r)
	case r.URL.Path == "/api/auth/passkeys/login/begin" && r.Method == http.MethodPost:
		h.passkeyLoginBegin(w, r)
	case r.URL.Path == "/api/auth/passkeys/login/finish" && r.Method == http.MethodPost:
		h.passkeyLoginFinish(w, r)
	case r.URL.Path == "/api/auth/login/2fa/passkey/begin" && r.Method == http.MethodPost:
		h.login2FAPasskeyBegin(w, r)
	case r.URL.Path == "/api/auth/login/2fa/passkey/finish" && r.Method == http.MethodPost:
		h.login2FAPasskeyFinish(w, r)
	case r.URL.Path == "/api/auth/logout" && r.Method == http.MethodPost:
		h.logout(w, r)
	case r.URL.Path == "/api/auth/me" && r.Method == http.MethodGet:
		h.me(w, r)
	case r.URL.Path == "/api/auth/me" && r.Method == http.MethodPut:
		h.updateMe(w, r)
	case r.URL.Path == "/api/auth/2fa/status" && r.Method == http.MethodGet:
		h.twoFAStatus(w, r)
	case r.URL.Path == "/api/auth/2fa/setup" && r.Method == http.MethodPost:
		h.twoFASetup(w, r)
	case r.URL.Path == "/api/auth/2fa/enable" && r.Method == http.MethodPost:
		h.twoFAEnable(w, r)
	case r.URL.Path == "/api/auth/2fa/disable" && r.Method == http.MethodPost:
		h.twoFADisable(w, r)
	case r.URL.Path == "/api/auth/change-password" && r.Method == http.MethodPost:
		h.changePassword(w, r)
	case r.URL.Path == "/api/auth/passkeys" && r.Method == http.MethodGet:
		h.passkeysList(w, r)
	case r.URL.Path == "/api/auth/passkeys/register/begin" && r.Method == http.MethodPost:
		h.passkeysRegisterBegin(w, r)
	case r.URL.Path == "/api/auth/passkeys/register/finish" && r.Method == http.MethodPost:
		h.passkeysRegisterFinish(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/auth/passkeys/") && strings.HasSuffix(r.URL.Path, "/rename") && r.Method == http.MethodPut:
		h.passkeysRename(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/auth/passkeys/") && r.Method == http.MethodDelete:
		h.passkeysDelete(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (h *AuthHandler) bootstrap(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	session, err := h.auth.Bootstrap(withActorIP(r), req.Username, req.Email, req.Password)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	SetSessionCookieForRequest(w, r, session.ID)
	writeJSON(w, http.StatusOK, session)
}

func (h *AuthHandler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	username := strings.ToLower(strings.TrimSpace(req.Username))
	clientIP := normalizedClientIP(r)
	if blocked, retryAfter := loginRateLimiter.IsBlocked(username, clientIP, CurrentRuntimeSecuritySettings()); blocked {
		writeJSON(w, http.StatusTooManyRequests, map[string]any{
			"error":               "too many login attempts; please retry later",
			"retry_after_seconds": retryAfter,
		})
		return
	}
	result, err := h.auth.Login(withActorIP(r), req.Username, req.Password)
	if err != nil {
		loginRateLimiter.RegisterAttempt(username, clientIP, false, CurrentRuntimeSecuritySettings())
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": err.Error()})
		return
	}
	loginRateLimiter.RegisterAttempt(username, clientIP, true, CurrentRuntimeSecuritySettings())
	if !result.RequiresTwoFactor && result.Session.ID != "" {
		SetSessionCookieForRequest(w, r, result.Session.ID)
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *AuthHandler) login2FA(w http.ResponseWriter, r *http.Request) {
	var req login2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	session, err := h.auth.Login2FA(withActorIP(r), req.ChallengeID, req.Code, req.RecoveryCode)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": err.Error()})
		return
	}
	SetSessionCookieForRequest(w, r, session.ID)
	writeJSON(w, http.StatusOK, session)
}

func (h *AuthHandler) passkeyLoginBegin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	result, err := h.auth.BeginPasskeyLogin(withActorIP(r), req.Username, r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *AuthHandler) passkeyLoginFinish(w http.ResponseWriter, r *http.Request) {
	var req passkeyFinishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	result, err := h.auth.FinishPasskeyLogin(withActorIP(r), req.ChallengeID, req.Credential, r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": err.Error()})
		return
	}
	if !result.RequiresTwoFactor && result.Session.ID != "" {
		SetSessionCookieForRequest(w, r, result.Session.ID)
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *AuthHandler) login2FAPasskeyBegin(w http.ResponseWriter, r *http.Request) {
	var req passkey2FABeginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	result, err := h.auth.BeginPasskey2FA(withActorIP(r), req.ChallengeID, r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *AuthHandler) login2FAPasskeyFinish(w http.ResponseWriter, r *http.Request) {
	var req passkey2FAFinishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	session, err := h.auth.FinishPasskey2FA(withActorIP(r), req.ChallengeID, req.WebAuthnChallenge, req.Credential, r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": err.Error()})
		return
	}
	SetSessionCookieForRequest(w, r, session.ID)
	writeJSON(w, http.StatusOK, session)
}

func (h *AuthHandler) logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(SessionCookieName)
	if err == nil {
		_ = h.auth.Logout(withActorIP(r), cookie.Value)
	}
	clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) me(w http.ResponseWriter, r *http.Request) {
	sessionID, ok := readSessionID(r)
	if !ok {
		clearSessionCookie(w)
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "session required"})
		return
	}
	user, err := h.auth.Me(sessionID)
	if err != nil {
		clearSessionCookie(w)
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": err.Error()})
		return
	}
	SetSessionCookieForRequest(w, r, sessionID)
	writeJSON(w, http.StatusOK, user)
}

func (h *AuthHandler) updateMe(w http.ResponseWriter, r *http.Request) {
	sessionID, ok := readSessionID(r)
	if !ok {
		clearSessionCookie(w)
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "session required"})
		return
	}
	var req updateMeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	user, err := h.auth.UpdatePreferences(withActorIP(r), sessionID, services.AuthUserPreferences{
		Language: req.Language,
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	SetSessionCookieForRequest(w, r, sessionID)
	writeJSON(w, http.StatusOK, user)
}

func (h *AuthHandler) twoFAStatus(w http.ResponseWriter, r *http.Request) {
	sessionID, ok := readSessionID(r)
	if !ok {
		clearSessionCookie(w)
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "session required"})
		return
	}
	user, err := h.auth.Me(sessionID)
	if err != nil {
		clearSessionCookie(w)
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": err.Error()})
		return
	}
	SetSessionCookieForRequest(w, r, sessionID)
	writeJSON(w, http.StatusOK, map[string]any{
		"enabled":                  user.TOTPEnabled,
		"methods":                  []string{"totp", "recovery", "passkey"},
		"recovery_codes_remaining": user.RecoveryCodesRemaining,
	})
}

func (h *AuthHandler) twoFASetup(w http.ResponseWriter, r *http.Request) {
	sessionID, ok := readSessionID(r)
	if !ok {
		clearSessionCookie(w)
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "session required"})
		return
	}
	result, err := h.auth.SetupTOTP(withActorIP(r), sessionID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	SetSessionCookieForRequest(w, r, sessionID)
	writeJSON(w, http.StatusOK, result)
}

func (h *AuthHandler) twoFAEnable(w http.ResponseWriter, r *http.Request) {
	sessionID, ok := readSessionID(r)
	if !ok {
		clearSessionCookie(w)
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "session required"})
		return
	}
	var req totpEnableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	result, err := h.auth.EnableTOTP(withActorIP(r), sessionID, req.ChallengeID, req.Code)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	SetSessionCookieForRequest(w, r, sessionID)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":             result.OK,
		"user":           result.User,
		"recovery_codes": result.RecoveryCodes,
	})
}

func (h *AuthHandler) twoFADisable(w http.ResponseWriter, r *http.Request) {
	sessionID, ok := readSessionID(r)
	if !ok {
		clearSessionCookie(w)
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "session required"})
		return
	}
	var req totpDisableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	user, err := h.auth.DisableTOTP(withActorIP(r), sessionID, req.Password, req.RecoveryCode)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	SetSessionCookieForRequest(w, r, sessionID)
	writeJSON(w, http.StatusOK, user)
}

func (h *AuthHandler) changePassword(w http.ResponseWriter, r *http.Request) {
	sessionID, ok := readSessionID(r)
	if !ok {
		clearSessionCookie(w)
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "session required"})
		return
	}
	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	if err := h.auth.ChangePassword(withActorIP(r), sessionID, req.CurrentPassword, req.Password); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	SetSessionCookieForRequest(w, r, sessionID)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *AuthHandler) passkeysList(w http.ResponseWriter, r *http.Request) {
	sessionID, ok := readSessionID(r)
	if !ok {
		clearSessionCookie(w)
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "session required"})
		return
	}
	result, err := h.auth.ListPasskeys(sessionID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	SetSessionCookieForRequest(w, r, sessionID)
	writeJSON(w, http.StatusOK, result)
}

func (h *AuthHandler) passkeysRegisterBegin(w http.ResponseWriter, r *http.Request) {
	sessionID, ok := readSessionID(r)
	if !ok {
		clearSessionCookie(w)
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "session required"})
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	result, err := h.auth.BeginPasskeyRegister(withActorIP(r), sessionID, req.Name, r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	SetSessionCookieForRequest(w, r, sessionID)
	writeJSON(w, http.StatusOK, result)
}

func (h *AuthHandler) passkeysRegisterFinish(w http.ResponseWriter, r *http.Request) {
	sessionID, ok := readSessionID(r)
	if !ok {
		clearSessionCookie(w)
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "session required"})
		return
	}
	var req passkeyFinishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	item, err := h.auth.FinishPasskeyRegister(withActorIP(r), sessionID, req.ChallengeID, req.Name, req.Credential, r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	SetSessionCookieForRequest(w, r, sessionID)
	writeJSON(w, http.StatusOK, map[string]any{"item": item})
}

func (h *AuthHandler) passkeysRename(w http.ResponseWriter, r *http.Request) {
	sessionID, ok := readSessionID(r)
	if !ok {
		clearSessionCookie(w)
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "session required"})
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/auth/passkeys/")
	id = strings.TrimSuffix(id, "/rename")
	id = strings.Trim(id, "/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "passkey id is required"})
		return
	}
	var req passkeyRenameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}
	item, err := h.auth.RenamePasskey(sessionID, id, req.Name)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	SetSessionCookieForRequest(w, r, sessionID)
	writeJSON(w, http.StatusOK, map[string]any{"item": item, "ok": true})
}

func (h *AuthHandler) passkeysDelete(w http.ResponseWriter, r *http.Request) {
	sessionID, ok := readSessionID(r)
	if !ok {
		clearSessionCookie(w)
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "session required"})
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/auth/passkeys/")
	id = strings.Trim(id, "/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "passkey id is required"})
		return
	}
	if err := h.auth.DeletePasskey(sessionID, id); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	SetSessionCookieForRequest(w, r, sessionID)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func SetSessionCookie(w http.ResponseWriter, sessionID string) {
	SetSessionCookieWithOptions(w, sessionID, false)
}

func SetSessionCookieForRequest(w http.ResponseWriter, r *http.Request, sessionID string) {
	secure := requestIsSecure(r)
	SetSessionCookieWithOptions(w, sessionID, secure)
}

func SetSessionCookieWithOptions(w http.ResponseWriter, sessionID string, secure bool) {
	sessionBootTokenMu.RLock()
	bootToken := sessionBootToken
	sessionBootTokenMu.RUnlock()

	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   sessionCookieMaxAgeSeconds,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     SessionBootCookieName,
		Value:    bootToken,
		Path:     "/",
		MaxAge:   sessionCookieMaxAgeSeconds,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	ClearSessionCookie(w)
}

func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     SessionBootCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     SessionBootCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   true,
	})
}

func readSessionID(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return "", false
	}
	value := strings.TrimSpace(cookie.Value)
	return value, value != ""
}

func SessionBootToken() string {
	sessionBootTokenMu.RLock()
	defer sessionBootTokenMu.RUnlock()
	return sessionBootToken
}

func SetSessionBootToken(token string) {
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return
	}
	sessionBootTokenMu.Lock()
	defer sessionBootTokenMu.Unlock()
	sessionBootToken = trimmed
}

func generateSessionBootToken() string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "boot-token-fallback"
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}

func requestIsSecure(r *http.Request) bool {
	if r == nil {
		return false
	}
	if r.TLS != nil {
		return true
	}
	for _, header := range []string{"X-Forwarded-Proto", "X-Forwarded-Scheme", "X-Scheme", "Front-End-Https", "X-Forwarded-Ssl"} {
		value := strings.TrimSpace(strings.ToLower(r.Header.Get(header)))
		switch value {
		case "https", "on":
			return true
		}
	}
	return false
}

func normalizedClientIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		return strings.TrimSpace(r.RemoteAddr)
	}
	return strings.TrimSpace(host)
}

func (l *authLoginRateLimiter) key(username, ip string) string {
	return strings.TrimSpace(strings.ToLower(username)) + "|" + strings.TrimSpace(ip)
}

func (l *authLoginRateLimiter) IsBlocked(username, ip string, settings RuntimeSecuritySettings) (bool, int) {
	settings = normalizeRuntimeSecuritySettings(settings)
	if !settings.LoginRateLimitEnabled || l == nil {
		return false, 0
	}
	now := time.Now().UTC()
	windowStart := now.Add(-time.Duration(settings.LoginRateLimitWindowSecond) * time.Second)
	key := l.key(username, ip)

	l.mu.Lock()
	defer l.mu.Unlock()
	entry, ok := l.state[key]
	if !ok {
		return false, 0
	}
	entry = pruneLoginAttempts(entry, windowStart)
	if entry.BlockedUntil.After(now) {
		retry := int(entry.BlockedUntil.Sub(now).Seconds())
		if retry < 1 {
			retry = 1
		}
		l.state[key] = entry
		return true, retry
	}
	entry.BlockedUntil = time.Time{}
	l.state[key] = entry
	return false, 0
}

func (l *authLoginRateLimiter) RegisterAttempt(username, ip string, success bool, settings RuntimeSecuritySettings) {
	settings = normalizeRuntimeSecuritySettings(settings)
	if !settings.LoginRateLimitEnabled || l == nil {
		return
	}
	now := time.Now().UTC()
	windowStart := now.Add(-time.Duration(settings.LoginRateLimitWindowSecond) * time.Second)
	key := l.key(username, ip)

	l.mu.Lock()
	defer l.mu.Unlock()
	entry := pruneLoginAttempts(l.state[key], windowStart)
	if success {
		delete(l.state, key)
		return
	}
	entry.FailTimestamps = append(entry.FailTimestamps, now)
	if len(entry.FailTimestamps) >= settings.LoginRateLimitMaxAttempts {
		entry.BlockedUntil = now.Add(time.Duration(settings.LoginRateLimitBlockSecond) * time.Second)
		entry.FailTimestamps = nil
	}
	l.state[key] = entry
}

func pruneLoginAttempts(entry authLoginAttemptState, windowStart time.Time) authLoginAttemptState {
	if len(entry.FailTimestamps) == 0 {
		return entry
	}
	kept := entry.FailTimestamps[:0]
	for _, ts := range entry.FailTimestamps {
		if ts.After(windowStart) {
			kept = append(kept, ts)
		}
	}
	entry.FailTimestamps = kept
	return entry
}
