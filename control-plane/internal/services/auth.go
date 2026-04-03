package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"waf/control-plane/internal/audits"
	"waf/control-plane/internal/auth"
	"waf/control-plane/internal/passkeys"
	"waf/control-plane/internal/rbac"
	"waf/control-plane/internal/roles"
	"waf/control-plane/internal/sessions"
	"waf/control-plane/internal/users"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/skip2/go-qrcode"
)

type AuthUser struct {
	ID                     string   `json:"id"`
	Username               string   `json:"username"`
	Email                  string   `json:"email"`
	RoleIDs                []string `json:"role_ids"`
	Permissions            []string `json:"permissions"`
	TOTPEnabled            bool     `json:"totp_enabled"`
	RecoveryCodesRemaining int      `json:"recovery_codes_remaining"`
}

type SessionResult struct {
	ID   string   `json:"id"`
	User AuthUser `json:"user"`
}

type LoginResult struct {
	RequiresTwoFactor bool          `json:"requires_2fa"`
	ChallengeID       string        `json:"challenge_id,omitempty"`
	Methods           []string      `json:"methods,omitempty"`
	Session           SessionResult `json:"session,omitempty"`
}

type TOTPSetupResult struct {
	ChallengeID     string `json:"challenge_id"`
	Secret          string `json:"secret,omitempty"`
	ProvisioningURI string `json:"provisioning_uri,omitempty"`
	QRPNGBase64     string `json:"qr_png_base64,omitempty"`
	ManualSecret    string `json:"manual_secret,omitempty"`
}

type TOTPEnableResult struct {
	User          AuthUser  `json:"user"`
	RecoveryCodes []string  `json:"recovery_codes"`
	OK            bool      `json:"ok"`
	ExpiresAt     time.Time `json:"expires_at,omitempty"`
}

type PasskeyListResult struct {
	Items []PasskeyItem `json:"items"`
}

type PasskeyItem struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	CreatedAt  string `json:"created_at"`
	LastUsedAt string `json:"last_used_at,omitempty"`
}

type PasskeyBeginResult struct {
	ChallengeID string         `json:"challenge_id,omitempty"`
	Options     map[string]any `json:"options"`
}

type Passkey2FABeginResult struct {
	WebAuthnChallengeID string         `json:"webauthn_challenge_id"`
	Options             map[string]any `json:"options"`
}

type AuthSecurityConfig struct {
	Pepper   string
	WebAuthn WebAuthnConfig
}

type WebAuthnConfig struct {
	Enabled bool
	RPID    string
	RPName  string
	Origins []string
}

type AuthUserStore interface {
	Get(id string) (users.User, bool, error)
	FindByUsername(username string) (users.User, bool, error)
	Create(user users.User) (users.User, error)
	Count() (int, error)
	Update(user users.User) (users.User, error)
	MarkLogin(userID string, when time.Time) error
}

type AuthRoleStore interface {
	Get(id string) (roles.Role, bool, error)
	PermissionsForRoleIDs(roleIDs []string) []rbac.Permission
}

type AuthSessionStore interface {
	CreateSession(userID string, username string, roleIDs []string, ttl time.Duration) (sessions.Session, error)
	GetSession(id string) (sessions.Session, bool, error)
	TouchSession(id string, ttl time.Duration) (sessions.Session, bool, error)
	DeleteSession(id string) error
	CreateLoginChallenge(userID string, ttl time.Duration) (sessions.LoginChallenge, error)
	ConsumeLoginChallenge(id string) (sessions.LoginChallenge, bool, error)
	GetLoginChallenge(id string) (sessions.LoginChallenge, bool, error)
	CreateTOTPSetupChallenge(userID, secret string, ttl time.Duration) (sessions.TOTPSetupChallenge, error)
	ConsumeTOTPSetupChallenge(id string) (sessions.TOTPSetupChallenge, bool, error)
}

type AuthPasskeyStore interface {
	ListByUser(userID string) ([]passkeys.Passkey, error)
	FindByCredentialID(credentialID string) (passkeys.Passkey, bool, error)
	FindByID(id string) (passkeys.Passkey, bool, error)
	Create(item passkeys.Passkey) (passkeys.Passkey, error)
	Rename(id, name string) (passkeys.Passkey, error)
	Delete(id string) error
	MarkUsed(credentialID string, signCount uint32, when time.Time) error
	CreateChallenge(item passkeys.Challenge, ttl time.Duration) (passkeys.Challenge, error)
	GetChallenge(id string) (passkeys.Challenge, bool, error)
	DeleteChallenge(id string) error
}

type AuthService struct {
	users        AuthUserStore
	roles        AuthRoleStore
	sessions     AuthSessionStore
	passkeys     AuthPasskeyStore
	policy       *rbac.Policy
	issuer       string
	security     AuthSecurityConfig
	sessionTTL   time.Duration
	challengeTTL time.Duration
	audits       *AuditService
}

func NewAuthService(users AuthUserStore, roles AuthRoleStore, sessions AuthSessionStore, passkeys AuthPasskeyStore, issuer string, security AuthSecurityConfig, audits *AuditService) *AuthService {
	return &AuthService{
		users:        users,
		roles:        roles,
		sessions:     sessions,
		passkeys:     passkeys,
		policy:       rbac.NewPolicy(roles),
		issuer:       strings.TrimSpace(issuer),
		security:     normalizeAuthSecurityConfig(security),
		sessionTTL:   12 * time.Hour,
		challengeTTL: 5 * time.Minute,
		audits:       audits,
	}
}

func (s *AuthService) Login(ctx context.Context, username, password string) (result LoginResult, err error) {
	defer func() {
		actorUserID := ""
		if result.Session.User.ID != "" {
			actorUserID = result.Session.User.ID
		}
		recordAudit(ctx, s.audits, audits.AuditEvent{
			ActorUserID:  actorUserID,
			Action:       "auth.login",
			ResourceType: "auth",
			ResourceID:   normalizeAuditResource(username),
			Status:       auditStatus(err),
			Summary:      "login",
		})
	}()
	user, ok, err := s.users.FindByUsername(username)
	if err != nil {
		return LoginResult{}, err
	}
	if !ok || !user.IsActive {
		return LoginResult{}, errors.New("invalid credentials")
	}
	if !users.VerifyPassword(password, user.PasswordHash) {
		return LoginResult{}, errors.New("invalid credentials")
	}
	if user.TOTPEnabled {
		challenge, err := s.sessions.CreateLoginChallenge(user.ID, s.challengeTTL)
		if err != nil {
			return LoginResult{}, err
		}
		return LoginResult{
			RequiresTwoFactor: true,
			ChallengeID:       challenge.ID,
			Methods:           []string{"totp", "recovery", "passkey"},
		}, nil
	}
	session, err := s.createSession(user)
	if err != nil {
		return LoginResult{}, err
	}
	return LoginResult{Session: session}, nil
}

func (s *AuthService) Bootstrap(ctx context.Context, username, email, password string) (result SessionResult, err error) {
	defer func() {
		actorUserID := ""
		if result.User.ID != "" {
			actorUserID = result.User.ID
		}
		recordAudit(ctx, s.audits, audits.AuditEvent{
			ActorUserID:  actorUserID,
			Action:       "auth.bootstrap",
			ResourceType: "user",
			ResourceID:   normalizeAuditResource(username),
			Status:       auditStatus(err),
			Summary:      "initial admin bootstrap",
			Details:      auditBootstrapDetails(email),
		})
	}()

	username = strings.TrimSpace(username)
	email = strings.TrimSpace(strings.ToLower(email))
	password = strings.TrimSpace(password)
	if username == "" {
		return SessionResult{}, errors.New("username is required")
	}
	if email != "" {
		if _, err := mail.ParseAddress(email); err != nil {
			return SessionResult{}, errors.New("email is invalid")
		}
	}
	if password == "" {
		return SessionResult{}, errors.New("password is required")
	}

	count, err := s.users.Count()
	if err != nil {
		return SessionResult{}, err
	}
	if count > 0 {
		return SessionResult{}, errors.New("bootstrap is no longer available")
	}

	passwordHash, err := users.HashPassword(password)
	if err != nil {
		return SessionResult{}, err
	}

	user, err := s.users.Create(users.User{
		ID:           username,
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
		IsActive:     true,
		RoleIDs:      []string{"admin"},
	})
	if err != nil {
		return SessionResult{}, err
	}
	return s.createSession(user)
}

func (s *AuthService) Login2FA(ctx context.Context, challengeID, code, recoveryCode string) (result SessionResult, err error) {
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			ActorUserID:  result.User.ID,
			Action:       "auth.2fa",
			ResourceType: "auth",
			ResourceID:   challengeID,
			Status:       auditStatus(err),
			Summary:      "2fa verification",
		})
	}()
	challenge, ok, err := s.sessions.ConsumeLoginChallenge(challengeID)
	if err != nil {
		return SessionResult{}, err
	}
	if !ok {
		return SessionResult{}, errors.New("2fa challenge not found")
	}
	user, ok, err := s.users.Get(challenge.UserID)
	if err != nil {
		return SessionResult{}, err
	}
	if !ok || !user.IsActive || !user.TOTPEnabled {
		return SessionResult{}, errors.New("2fa challenge is invalid")
	}

	now := time.Now().UTC()
	if strings.TrimSpace(recoveryCode) != "" {
		if !consumeRecoveryCode(&user, recoveryCode, s.security.Pepper, now) {
			return SessionResult{}, errors.New("invalid recovery code")
		}
		if _, err := s.users.Update(user); err != nil {
			return SessionResult{}, err
		}
		return s.createSession(user)
	}

	updatedUser, secret, migrated, err := s.resolveUserTOTPSecret(user)
	if err != nil {
		return SessionResult{}, errors.New("invalid 2fa code")
	}
	if migrated {
		if _, err := s.users.Update(updatedUser); err != nil {
			return SessionResult{}, err
		}
	}
	if !auth.VerifyTOTP(secret, code, now) {
		return SessionResult{}, errors.New("invalid 2fa code")
	}
	return s.createSession(user)
}

func (s *AuthService) BeginPasskeyLogin(ctx context.Context, username string, req *http.Request) (PasskeyBeginResult, error) {
	web, err := s.webAuthnForRequest(req)
	if err != nil {
		return PasskeyBeginResult{}, err
	}

	username = strings.TrimSpace(strings.ToLower(username))
	now := time.Now().UTC()
	var (
		assertion *protocol.CredentialAssertion
		session   *webauthn.SessionData
		kind      = "discoverable"
		userID    string
	)

	if username != "" {
		user, ok, err := s.users.FindByUsername(username)
		if err != nil {
			return PasskeyBeginResult{}, err
		}
		if !ok || !user.IsActive {
			return PasskeyBeginResult{}, errors.New("invalid credentials")
		}
		items, err := s.passkeys.ListByUser(user.ID)
		if err != nil {
			return PasskeyBeginResult{}, err
		}
		waUser := newWebAuthnUser(user, items)
		assertion, session, err = web.BeginLogin(waUser, webauthn.WithUserVerification(protocol.VerificationPreferred))
		if err != nil {
			return PasskeyBeginResult{}, errors.New("auth.passkeys.failed")
		}
		kind = "login"
		userID = user.ID
	} else {
		assertion, session, err = web.BeginDiscoverableLogin(webauthn.WithUserVerification(protocol.VerificationPreferred))
		if err != nil {
			return PasskeyBeginResult{}, errors.New("auth.passkeys.failed")
		}
	}
	session.Expires = now.Add(5 * time.Minute)
	sessionDataJSON, err := json.Marshal(session)
	if err != nil {
		return PasskeyBeginResult{}, err
	}
	challenge, err := s.passkeys.CreateChallenge(passkeys.Challenge{
		Kind:            kind,
		UserID:          userID,
		SessionDataJSON: string(sessionDataJSON),
		ClientIP:        requestIP(req),
		UserAgent:       requestUserAgent(req),
	}, 5*time.Minute)
	if err != nil {
		return PasskeyBeginResult{}, err
	}
	options, err := toJSONMap(assertion)
	if err != nil {
		return PasskeyBeginResult{}, err
	}
	return PasskeyBeginResult{ChallengeID: challenge.ID, Options: options}, nil
}

func (s *AuthService) FinishPasskeyLogin(ctx context.Context, challengeID string, credentialJSON json.RawMessage, req *http.Request) (LoginResult, error) {
	ch, ok, err := s.passkeys.GetChallenge(challengeID)
	if err != nil {
		return LoginResult{}, err
	}
	if !ok {
		return LoginResult{}, errors.New("auth.passkeys.challengeInvalid")
	}
	defer func() { _ = s.passkeys.DeleteChallenge(ch.ID) }()
	if strings.TrimSpace(ch.Kind) != "login" && strings.TrimSpace(ch.Kind) != "discoverable" {
		return LoginResult{}, errors.New("auth.passkeys.challengeInvalid")
	}
	if !challengeNotExpired(ch.ExpiresAt) {
		return LoginResult{}, errors.New("auth.passkeys.challengeExpired")
	}
	if len(credentialJSON) == 0 {
		return LoginResult{}, errors.New("auth.passkeys.failed")
	}
	web, err := s.webAuthnForRequest(req)
	if err != nil {
		return LoginResult{}, err
	}
	var sessionData webauthn.SessionData
	if err := json.Unmarshal([]byte(ch.SessionDataJSON), &sessionData); err != nil {
		return LoginResult{}, errors.New("auth.passkeys.challengeInvalid")
	}
	parsed, err := protocol.ParseCredentialRequestResponseBytes(credentialJSON)
	if err != nil {
		return LoginResult{}, errors.New("auth.passkeys.failed")
	}

	var (
		user         users.User
		credentialID string
		signCount    uint32
	)

	if strings.TrimSpace(ch.UserID) != "" {
		u, ok, err := s.users.Get(ch.UserID)
		if err != nil {
			return LoginResult{}, err
		}
		if !ok || !u.IsActive {
			return LoginResult{}, errors.New("auth.passkeys.failed")
		}
		keys, err := s.passkeys.ListByUser(u.ID)
		if err != nil {
			return LoginResult{}, err
		}
		waUser := newWebAuthnUser(u, keys)
		credential, err := web.ValidateLogin(waUser, sessionData, parsed)
		if err != nil || credential == nil {
			return LoginResult{}, errors.New("auth.passkeys.failed")
		}
		user = u
		credentialID = base64.RawURLEncoding.EncodeToString(credential.ID)
		signCount = credential.Authenticator.SignCount
	} else {
		handler := func(rawID []byte, _ []byte) (webauthn.User, error) {
			credID := base64.RawURLEncoding.EncodeToString(rawID)
			record, ok, err := s.passkeys.FindByCredentialID(credID)
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, errors.New("credential not found")
			}
			u, ok, err := s.users.Get(record.UserID)
			if err != nil {
				return nil, err
			}
			if !ok || !u.IsActive {
				return nil, errors.New("user not found")
			}
			keys, err := s.passkeys.ListByUser(u.ID)
			if err != nil {
				return nil, err
			}
			return newWebAuthnUser(u, keys), nil
		}
		waUser, credential, err := web.ValidatePasskeyLogin(handler, sessionData, parsed)
		if err != nil || waUser == nil || credential == nil {
			return LoginResult{}, errors.New("auth.passkeys.failed")
		}
		uid, err := parseUserIDFromWebAuthnHandle(waUser.WebAuthnID())
		if err != nil {
			return LoginResult{}, errors.New("auth.passkeys.failed")
		}
		u, ok, err := s.users.Get(uid)
		if err != nil {
			return LoginResult{}, err
		}
		if !ok || !u.IsActive {
			return LoginResult{}, errors.New("auth.passkeys.failed")
		}
		user = u
		credentialID = base64.RawURLEncoding.EncodeToString(credential.ID)
		signCount = credential.Authenticator.SignCount
	}

	_ = s.passkeys.MarkUsed(credentialID, signCount, time.Now().UTC())
	if user.TOTPEnabled {
		loginChallenge, err := s.sessions.CreateLoginChallenge(user.ID, s.challengeTTL)
		if err != nil {
			return LoginResult{}, err
		}
		return LoginResult{
			RequiresTwoFactor: true,
			ChallengeID:       loginChallenge.ID,
			Methods:           []string{"totp", "recovery", "passkey"},
		}, nil
	}
	session, err := s.createSession(user)
	if err != nil {
		return LoginResult{}, err
	}
	return LoginResult{Session: session}, nil
}

func (s *AuthService) BeginPasskey2FA(ctx context.Context, loginChallengeID string, req *http.Request) (Passkey2FABeginResult, error) {
	web, err := s.webAuthnForRequest(req)
	if err != nil {
		return Passkey2FABeginResult{}, err
	}
	loginChallenge, ok, err := s.sessions.GetLoginChallenge(loginChallengeID)
	if err != nil {
		return Passkey2FABeginResult{}, err
	}
	if !ok {
		return Passkey2FABeginResult{}, errors.New("auth.2fa.challengeInvalid")
	}
	user, ok, err := s.users.Get(loginChallenge.UserID)
	if err != nil {
		return Passkey2FABeginResult{}, err
	}
	if !ok || !user.IsActive || !user.TOTPEnabled {
		return Passkey2FABeginResult{}, errors.New("auth.2fa.challengeInvalid")
	}
	items, err := s.passkeys.ListByUser(user.ID)
	if err != nil {
		return Passkey2FABeginResult{}, err
	}
	if len(items) == 0 {
		return Passkey2FABeginResult{}, errors.New("auth.passkeys.notAvailable")
	}
	waUser := newWebAuthnUser(user, items)
	assertion, session, err := web.BeginLogin(waUser, webauthn.WithUserVerification(protocol.VerificationPreferred))
	if err != nil || session == nil {
		return Passkey2FABeginResult{}, errors.New("auth.passkeys.failed")
	}
	session.Expires = time.Now().UTC().Add(5 * time.Minute)
	sessionDataJSON, err := json.Marshal(session)
	if err != nil {
		return Passkey2FABeginResult{}, err
	}
	created, err := s.passkeys.CreateChallenge(passkeys.Challenge{
		Kind:            "login2fa-passkey",
		UserID:          user.ID,
		RefID:           loginChallengeID,
		SessionDataJSON: string(sessionDataJSON),
		ClientIP:        requestIP(req),
		UserAgent:       requestUserAgent(req),
	}, 5*time.Minute)
	if err != nil {
		return Passkey2FABeginResult{}, err
	}
	options, err := toJSONMap(assertion)
	if err != nil {
		return Passkey2FABeginResult{}, err
	}
	return Passkey2FABeginResult{
		WebAuthnChallengeID: created.ID,
		Options:             options,
	}, nil
}

func (s *AuthService) FinishPasskey2FA(ctx context.Context, loginChallengeID, webAuthnChallengeID string, credentialJSON json.RawMessage, req *http.Request) (SessionResult, error) {
	ch, ok, err := s.passkeys.GetChallenge(webAuthnChallengeID)
	if err != nil {
		return SessionResult{}, err
	}
	if !ok {
		return SessionResult{}, errors.New("auth.passkeys.challengeInvalid")
	}
	defer func() { _ = s.passkeys.DeleteChallenge(ch.ID) }()
	if strings.TrimSpace(ch.Kind) != "login2fa-passkey" || strings.TrimSpace(ch.RefID) != strings.TrimSpace(loginChallengeID) {
		return SessionResult{}, errors.New("auth.passkeys.challengeInvalid")
	}
	if !challengeNotExpired(ch.ExpiresAt) {
		return SessionResult{}, errors.New("auth.passkeys.challengeExpired")
	}
	loginChallenge, ok, err := s.sessions.GetLoginChallenge(loginChallengeID)
	if err != nil {
		return SessionResult{}, err
	}
	if !ok || loginChallenge.UserID != ch.UserID {
		return SessionResult{}, errors.New("auth.2fa.challengeInvalid")
	}
	user, ok, err := s.users.Get(ch.UserID)
	if err != nil {
		return SessionResult{}, err
	}
	if !ok || !user.IsActive || !user.TOTPEnabled {
		return SessionResult{}, errors.New("auth.2fa.challengeInvalid")
	}
	web, err := s.webAuthnForRequest(req)
	if err != nil {
		return SessionResult{}, err
	}
	var sessionData webauthn.SessionData
	if err := json.Unmarshal([]byte(ch.SessionDataJSON), &sessionData); err != nil {
		return SessionResult{}, errors.New("auth.passkeys.challengeInvalid")
	}
	parsed, err := protocol.ParseCredentialRequestResponseBytes(credentialJSON)
	if err != nil {
		return SessionResult{}, errors.New("auth.passkeys.failed")
	}
	items, err := s.passkeys.ListByUser(user.ID)
	if err != nil {
		return SessionResult{}, err
	}
	waUser := newWebAuthnUser(user, items)
	credential, err := web.ValidateLogin(waUser, sessionData, parsed)
	if err != nil || credential == nil {
		return SessionResult{}, errors.New("auth.passkeys.failed")
	}
	if consumed, ok, err := s.sessions.ConsumeLoginChallenge(loginChallengeID); err != nil {
		return SessionResult{}, err
	} else if !ok || consumed.UserID != user.ID {
		return SessionResult{}, errors.New("auth.2fa.challengeInvalid")
	}
	credentialID := base64.RawURLEncoding.EncodeToString(credential.ID)
	_ = s.passkeys.MarkUsed(credentialID, credential.Authenticator.SignCount, time.Now().UTC())
	return s.createSession(user)
}

func (s *AuthService) ListPasskeys(sessionID string) (PasskeyListResult, error) {
	session, err := s.Authenticate(sessionID)
	if err != nil {
		return PasskeyListResult{}, err
	}
	items, err := s.passkeys.ListByUser(session.UserID)
	if err != nil {
		return PasskeyListResult{}, err
	}
	out := make([]PasskeyItem, 0, len(items))
	for _, item := range items {
		out = append(out, PasskeyItem{
			ID:         item.ID,
			Name:       item.Name,
			CreatedAt:  item.CreatedAt,
			LastUsedAt: item.LastUsedAt,
		})
	}
	return PasskeyListResult{Items: out}, nil
}

func (s *AuthService) BeginPasskeyRegister(ctx context.Context, sessionID, name string, req *http.Request) (PasskeyBeginResult, error) {
	web, err := s.webAuthnForRequest(req)
	if err != nil {
		return PasskeyBeginResult{}, err
	}
	session, err := s.Authenticate(sessionID)
	if err != nil {
		return PasskeyBeginResult{}, err
	}
	user, ok, err := s.users.Get(session.UserID)
	if err != nil {
		return PasskeyBeginResult{}, err
	}
	if !ok || !user.IsActive {
		return PasskeyBeginResult{}, errors.New("user not found")
	}

	name = strings.TrimSpace(name)
	if name == "" {
		name = "My Passkey"
	}
	items, err := s.passkeys.ListByUser(user.ID)
	if err != nil {
		return PasskeyBeginResult{}, err
	}
	waUser := newWebAuthnUser(user, items)
	exclude := webauthn.Credentials(waUser.WebAuthnCredentials()).CredentialDescriptors()
	creation, sessionData, err := web.BeginRegistration(
		waUser,
		webauthn.WithExclusions(exclude),
		webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			UserVerification: protocol.VerificationPreferred,
		}),
		webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementPreferred),
		webauthn.WithConveyancePreference(protocol.PreferNoAttestation),
	)
	if err != nil || sessionData == nil {
		return PasskeyBeginResult{}, errors.New("auth.passkeys.failed")
	}
	sessionData.Expires = time.Now().UTC().Add(8 * time.Minute)
	sessionDataJSON, err := json.Marshal(sessionData)
	if err != nil {
		return PasskeyBeginResult{}, err
	}
	created, err := s.passkeys.CreateChallenge(passkeys.Challenge{
		Kind:            "register-passkey",
		UserID:          user.ID,
		Name:            name,
		SessionDataJSON: string(sessionDataJSON),
		ClientIP:        requestIP(req),
		UserAgent:       requestUserAgent(req),
	}, 8*time.Minute)
	if err != nil {
		return PasskeyBeginResult{}, err
	}
	options, err := toJSONMap(creation)
	if err != nil {
		return PasskeyBeginResult{}, err
	}
	return PasskeyBeginResult{ChallengeID: created.ID, Options: options}, nil
}

func (s *AuthService) FinishPasskeyRegister(ctx context.Context, sessionID, challengeID, name string, credentialJSON json.RawMessage, req *http.Request) (PasskeyItem, error) {
	web, err := s.webAuthnForRequest(req)
	if err != nil {
		return PasskeyItem{}, err
	}
	session, err := s.Authenticate(sessionID)
	if err != nil {
		return PasskeyItem{}, err
	}
	ch, ok, err := s.passkeys.GetChallenge(challengeID)
	if err != nil {
		return PasskeyItem{}, err
	}
	if !ok {
		return PasskeyItem{}, errors.New("auth.passkeys.challengeInvalid")
	}
	defer func() { _ = s.passkeys.DeleteChallenge(ch.ID) }()
	if strings.TrimSpace(ch.Kind) != "register-passkey" || ch.UserID != session.UserID {
		return PasskeyItem{}, errors.New("auth.passkeys.challengeInvalid")
	}
	if !challengeNotExpired(ch.ExpiresAt) {
		return PasskeyItem{}, errors.New("auth.passkeys.challengeExpired")
	}
	if len(credentialJSON) == 0 {
		return PasskeyItem{}, errors.New("auth.passkeys.failed")
	}
	user, ok, err := s.users.Get(session.UserID)
	if err != nil {
		return PasskeyItem{}, err
	}
	if !ok || !user.IsActive {
		return PasskeyItem{}, errors.New("auth.passkeys.failed")
	}
	existing, err := s.passkeys.ListByUser(user.ID)
	if err != nil {
		return PasskeyItem{}, err
	}
	waUser := newWebAuthnUser(user, existing)
	var sessionData webauthn.SessionData
	if err := json.Unmarshal([]byte(ch.SessionDataJSON), &sessionData); err != nil {
		return PasskeyItem{}, errors.New("auth.passkeys.challengeInvalid")
	}
	parsed, err := protocol.ParseCredentialCreationResponseBytes(credentialJSON)
	if err != nil {
		return PasskeyItem{}, errors.New("auth.passkeys.failed")
	}
	cred, err := web.CreateCredential(waUser, sessionData, parsed)
	if err != nil || cred == nil {
		return PasskeyItem{}, errors.New("auth.passkeys.failed")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = strings.TrimSpace(ch.Name)
	}
	if name == "" {
		name = "My Passkey"
	}
	transports := make([]string, 0, len(cred.Transport))
	for _, item := range cred.Transport {
		value := strings.TrimSpace(string(item))
		if value == "" {
			continue
		}
		transports = append(transports, value)
	}
	item, err := s.passkeys.Create(passkeys.Passkey{
		UserID:          session.UserID,
		Name:            name,
		CredentialID:    base64.RawURLEncoding.EncodeToString(cred.ID),
		PublicKey:       append([]byte(nil), cred.PublicKey...),
		AttestationType: strings.TrimSpace(cred.AttestationType),
		Transports:      transports,
		AAGUID:          append([]byte(nil), cred.Authenticator.AAGUID...),
		SignCount:       cred.Authenticator.SignCount,
	})
	if err != nil {
		return PasskeyItem{}, err
	}
	return PasskeyItem{
		ID:         item.ID,
		Name:       item.Name,
		CreatedAt:  item.CreatedAt,
		LastUsedAt: item.LastUsedAt,
	}, nil
}

func (s *AuthService) RenamePasskey(sessionID, id, name string) (PasskeyItem, error) {
	session, err := s.Authenticate(sessionID)
	if err != nil {
		return PasskeyItem{}, err
	}
	item, ok, err := s.passkeys.FindByID(id)
	if err != nil {
		return PasskeyItem{}, err
	}
	if !ok || item.UserID != session.UserID {
		return PasskeyItem{}, errors.New("auth.passkeys.failed")
	}
	updated, err := s.passkeys.Rename(id, name)
	if err != nil {
		return PasskeyItem{}, err
	}
	return PasskeyItem{
		ID:         updated.ID,
		Name:       updated.Name,
		CreatedAt:  updated.CreatedAt,
		LastUsedAt: updated.LastUsedAt,
	}, nil
}

func (s *AuthService) DeletePasskey(sessionID, id string) error {
	session, err := s.Authenticate(sessionID)
	if err != nil {
		return err
	}
	item, ok, err := s.passkeys.FindByID(id)
	if err != nil {
		return err
	}
	if !ok || item.UserID != session.UserID {
		return errors.New("auth.passkeys.failed")
	}
	return s.passkeys.Delete(id)
}

func (s *AuthService) ChangePassword(ctx context.Context, sessionID, currentPassword, password string) (err error) {
	session, err := s.Authenticate(sessionID)
	if err != nil {
		return err
	}
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			ActorUserID:  session.UserID,
			Action:       "auth.change_password",
			ResourceType: "user",
			ResourceID:   session.UserID,
			Status:       auditStatus(err),
			Summary:      "change password",
		})
	}()
	user, ok, err := s.users.Get(session.UserID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("user not found")
	}
	if !users.VerifyPassword(currentPassword, user.PasswordHash) {
		return errors.New("current password is invalid")
	}
	password = strings.TrimSpace(password)
	if password == "" {
		return errors.New("password is required")
	}
	passwordHash, err := users.HashPassword(password)
	if err != nil {
		return err
	}
	user.PasswordHash = passwordHash
	_, err = s.users.Update(user)
	return err
}

func (s *AuthService) Logout(ctx context.Context, sessionID string) (err error) {
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			Action:       "auth.logout",
			ResourceType: "session",
			ResourceID:   sessionID,
			Status:       auditStatus(err),
			Summary:      "logout",
		})
	}()
	return s.sessions.DeleteSession(sessionID)
}

func (s *AuthService) Authenticate(sessionID string) (auth.SessionView, error) {
	session, ok, err := s.sessions.TouchSession(sessionID, s.sessionTTL)
	if err != nil {
		return auth.SessionView{}, err
	}
	if !ok {
		return auth.SessionView{}, errors.New("session not found")
	}
	return auth.SessionView{
		SessionID:   session.ID,
		UserID:      session.UserID,
		Username:    session.Username,
		RoleIDs:     append([]string(nil), session.RoleIDs...),
		Permissions: s.policy.Permissions(session.RoleIDs),
	}, nil
}

func (s *AuthService) Me(sessionID string) (AuthUser, error) {
	session, err := s.Authenticate(sessionID)
	if err != nil {
		return AuthUser{}, err
	}
	return s.userByID(session.UserID)
}

func (s *AuthService) SetupTOTP(ctx context.Context, sessionID string) (result TOTPSetupResult, err error) {
	session, err := s.Authenticate(sessionID)
	if err != nil {
		return TOTPSetupResult{}, err
	}
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			ActorUserID:  session.UserID,
			Action:       "auth.2fa_setup",
			ResourceType: "user",
			ResourceID:   session.UserID,
			Status:       auditStatus(err),
			Summary:      "2fa setup",
		})
	}()
	user, ok, err := s.users.Get(session.UserID)
	if err != nil {
		return TOTPSetupResult{}, err
	}
	if !ok {
		return TOTPSetupResult{}, errors.New("user not found")
	}
	secret, err := auth.NewTOTPSecret()
	if err != nil {
		return TOTPSetupResult{}, err
	}
	secretEnc, err := auth.EncryptTOTPSecret(secret, s.security.Pepper)
	if err != nil {
		return TOTPSetupResult{}, err
	}
	challenge, err := s.sessions.CreateTOTPSetupChallenge(user.ID, secretEnc, 10*time.Minute)
	if err != nil {
		return TOTPSetupResult{}, err
	}
	uri := auth.ProvisioningURI(s.issuer, user.Username, secret)
	png, err := qrcode.Encode(uri, qrcode.Medium, 256)
	if err != nil {
		return TOTPSetupResult{}, err
	}
	return TOTPSetupResult{
		ChallengeID:     challenge.ID,
		Secret:          secret,
		ManualSecret:    secret,
		ProvisioningURI: uri,
		QRPNGBase64:     "data:image/png;base64," + base64.StdEncoding.EncodeToString(png),
	}, nil
}

func (s *AuthService) EnableTOTP(ctx context.Context, sessionID, challengeID, code string) (result TOTPEnableResult, err error) {
	session, err := s.Authenticate(sessionID)
	if err != nil {
		return TOTPEnableResult{}, err
	}
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			ActorUserID:  session.UserID,
			Action:       "auth.2fa_enable",
			ResourceType: "user",
			ResourceID:   session.UserID,
			Status:       auditStatus(err),
			Summary:      "2fa enable",
		})
	}()
	challenge, ok, err := s.sessions.ConsumeTOTPSetupChallenge(challengeID)
	if err != nil {
		return TOTPEnableResult{}, err
	}
	if !ok || challenge.UserID != session.UserID {
		return TOTPEnableResult{}, errors.New("totp setup challenge not found")
	}
	secret, err := auth.DecryptTOTPSecret(challenge.Secret, s.security.Pepper)
	if err != nil || strings.TrimSpace(secret) == "" {
		return TOTPEnableResult{}, errors.New("invalid 2fa code")
	}
	if !auth.VerifyTOTP(secret, code, time.Now().UTC()) {
		return TOTPEnableResult{}, errors.New("invalid 2fa code")
	}
	user, ok, err := s.users.Get(session.UserID)
	if err != nil {
		return TOTPEnableResult{}, err
	}
	if !ok {
		return TOTPEnableResult{}, errors.New("user not found")
	}

	codes, err := auth.GenerateRecoveryCodes(10)
	if err != nil {
		return TOTPEnableResult{}, err
	}
	recoveryHashes := make([]users.TOTPRecoveryHash, 0, len(codes))
	for _, item := range codes {
		hashed, err := auth.HashRecoveryCode(item, s.security.Pepper)
		if err != nil {
			return TOTPEnableResult{}, err
		}
		recoveryHashes = append(recoveryHashes, users.TOTPRecoveryHash{Hash: hashed.Hash, Salt: hashed.Salt})
	}

	user.TOTPEnabled = true
	user.TOTPSecretEnc = challenge.Secret
	user.TOTPSecret = ""
	user.TOTPRecoveryHashes = recoveryHashes
	if _, err := s.users.Update(user); err != nil {
		return TOTPEnableResult{}, err
	}
	me, err := s.userByID(user.ID)
	if err != nil {
		return TOTPEnableResult{}, err
	}
	return TOTPEnableResult{
		OK:            true,
		User:          me,
		RecoveryCodes: codes,
	}, nil
}

func (s *AuthService) DisableTOTP(ctx context.Context, sessionID, password, recoveryCode string) (result AuthUser, err error) {
	session, err := s.Authenticate(sessionID)
	if err != nil {
		return AuthUser{}, err
	}
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			ActorUserID:  session.UserID,
			Action:       "auth.2fa_disable",
			ResourceType: "user",
			ResourceID:   session.UserID,
			Status:       auditStatus(err),
			Summary:      "2fa disable",
		})
	}()
	user, ok, err := s.users.Get(session.UserID)
	if err != nil {
		return AuthUser{}, err
	}
	if !ok {
		return AuthUser{}, errors.New("user not found")
	}
	if !user.TOTPEnabled {
		return s.userByID(user.ID)
	}
	if !users.VerifyPassword(password, user.PasswordHash) {
		return AuthUser{}, errors.New("invalid current password")
	}
	if !consumeRecoveryCode(&user, recoveryCode, s.security.Pepper, time.Now().UTC()) {
		return AuthUser{}, errors.New("invalid recovery code")
	}
	user.TOTPEnabled = false
	user.TOTPSecret = ""
	user.TOTPSecretEnc = ""
	user.TOTPRecoveryHashes = nil
	if _, err := s.users.Update(user); err != nil {
		return AuthUser{}, err
	}
	return s.userByID(user.ID)
}

func (s *AuthService) RequirePermission(session auth.SessionView, permission rbac.Permission) bool {
	return s.policy.Allowed(session.RoleIDs, permission)
}

func (s *AuthService) createSession(user users.User) (SessionResult, error) {
	session, err := s.sessions.CreateSession(user.ID, user.Username, user.RoleIDs, s.sessionTTL)
	if err != nil {
		return SessionResult{}, err
	}
	_ = s.users.MarkLogin(user.ID, time.Now().UTC())
	me, err := s.userByID(user.ID)
	if err != nil {
		return SessionResult{}, err
	}
	return SessionResult{ID: session.ID, User: me}, nil
}

func (s *AuthService) userByID(userID string) (AuthUser, error) {
	user, ok, err := s.users.Get(userID)
	if err != nil {
		return AuthUser{}, err
	}
	if !ok {
		return AuthUser{}, fmt.Errorf("user %s not found", userID)
	}
	return AuthUser{
		ID:                     user.ID,
		Username:               user.Username,
		Email:                  user.Email,
		RoleIDs:                append([]string(nil), user.RoleIDs...),
		Permissions:            s.policy.Permissions(user.RoleIDs),
		TOTPEnabled:            user.TOTPEnabled,
		RecoveryCodesRemaining: countUnusedRecoveryCodes(user),
	}, nil
}

func normalizeAuditResource(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "unknown"
	}
	return value
}

func auditBootstrapDetails(email string) map[string]any {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return nil
	}
	return map[string]any{
		"email": email,
	}
}

func consumeRecoveryCode(user *users.User, recoveryCode, pepper string, now time.Time) bool {
	if user == nil {
		return false
	}
	for i := range user.TOTPRecoveryHashes {
		item := user.TOTPRecoveryHashes[i]
		if strings.TrimSpace(item.UsedAt) != "" {
			continue
		}
		if auth.VerifyRecoveryCode(recoveryCode, pepper, item.Hash, item.Salt) {
			user.TOTPRecoveryHashes[i].UsedAt = now.UTC().Format(time.RFC3339Nano)
			return true
		}
	}
	return false
}

func countUnusedRecoveryCodes(user users.User) int {
	count := 0
	for _, item := range user.TOTPRecoveryHashes {
		if strings.TrimSpace(item.UsedAt) == "" {
			count++
		}
	}
	return count
}

func deriveRPID(req *http.Request) string {
	if req == nil {
		return "localhost"
	}
	host := strings.TrimSpace(req.Host)
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		return "localhost"
	}
	return host
}

func requestIP(req *http.Request) string {
	if req == nil {
		return ""
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(req.RemoteAddr))
	if err != nil {
		return strings.TrimSpace(req.RemoteAddr)
	}
	return strings.TrimSpace(host)
}

func requestUserAgent(req *http.Request) string {
	if req == nil {
		return ""
	}
	return strings.TrimSpace(req.UserAgent())
}

func challengeNotExpired(expiresAt string) bool {
	exp, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(expiresAt))
	if err != nil {
		return false
	}
	return exp.After(time.Now().UTC())
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func newB64URL(size int) string {
	if size <= 0 {
		size = 32
	}
	buf := make([]byte, size)
	_, _ = rand.Read(buf)
	return base64.RawURLEncoding.EncodeToString(buf)
}

func normalizeAuthSecurityConfig(input AuthSecurityConfig) AuthSecurityConfig {
	cfg := input
	cfg.Pepper = strings.TrimSpace(cfg.Pepper)
	if cfg.Pepper == "" {
		cfg.Pepper = "waf-dev-pepper-change-me"
	}
	cfg.WebAuthn.RPID = strings.ToLower(strings.TrimSpace(cfg.WebAuthn.RPID))
	cfg.WebAuthn.RPName = strings.TrimSpace(cfg.WebAuthn.RPName)
	if cfg.WebAuthn.RPName == "" {
		cfg.WebAuthn.RPName = "TARINIO"
	}
	if !cfg.WebAuthn.Enabled {
		cfg.WebAuthn.Origins = nil
		return cfg
	}
	out := make([]string, 0, len(cfg.WebAuthn.Origins))
	for _, item := range cfg.WebAuthn.Origins {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	cfg.WebAuthn.Origins = out
	return cfg
}

func (s *AuthService) webAuthnForRequest(req *http.Request) (*webauthn.WebAuthn, error) {
	if !s.security.WebAuthn.Enabled {
		return nil, errors.New("auth.passkeys.disabled")
	}
	rpID := strings.TrimSpace(s.security.WebAuthn.RPID)
	if rpID == "" {
		rpID = deriveRPID(req)
	}
	origins := append([]string(nil), s.security.WebAuthn.Origins...)
	if len(origins) == 0 {
		host := ""
		if req != nil {
			host = strings.TrimSpace(req.Host)
		}
		if host != "" {
			origins = []string{"https://" + host, "http://" + host}
		}
	}
	if rpID == "" || len(origins) == 0 {
		return nil, errors.New("auth.passkeys.misconfigured")
	}
	return webauthn.New(&webauthn.Config{
		RPID:          rpID,
		RPDisplayName: s.security.WebAuthn.RPName,
		RPOrigins:     origins,
	})
}

func (s *AuthService) resolveUserTOTPSecret(user users.User) (users.User, string, bool, error) {
	if strings.TrimSpace(user.TOTPSecretEnc) != "" {
		secret, err := auth.DecryptTOTPSecret(user.TOTPSecretEnc, s.security.Pepper)
		if err != nil {
			return user, "", false, err
		}
		return user, secret, false, nil
	}
	secret := strings.TrimSpace(user.TOTPSecret)
	if secret == "" {
		return user, "", false, nil
	}
	secretEnc, err := auth.EncryptTOTPSecret(secret, s.security.Pepper)
	if err != nil {
		return user, "", false, err
	}
	user.TOTPSecretEnc = secretEnc
	user.TOTPSecret = ""
	return user, secret, true, nil
}
