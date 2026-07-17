package services

import (
	"time"

	"waf/control-plane/internal/passkeys"
	"waf/control-plane/internal/rbac"
	"waf/control-plane/internal/roles"
	"waf/control-plane/internal/sessions"
	"waf/control-plane/internal/users"
)

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
	DeleteSessionsByUser(userID string) error
	DeleteSessionsByUserExcept(userID, exceptSessionID string) error
	CreateLoginChallenge(userID string, ttl time.Duration) (sessions.LoginChallenge, error)
	ConsumeLoginChallenge(id string) (sessions.LoginChallenge, bool, error)
	GetLoginChallenge(id string) (sessions.LoginChallenge, bool, error)
	CreateTOTPSetupChallenge(userID, secret string, ttl time.Duration) (sessions.TOTPSetupChallenge, error)
	ConsumeTOTPSetupChallenge(id string) (sessions.TOTPSetupChallenge, bool, error)
}

type AuthStepUpStore interface {
	StepUpStatus(sessionID, userID string, now time.Time) (sessions.StepUpStatus, error)
	RecordStepUpFailure(userID string, now time.Time) (sessions.StepUpStatus, error)
	GrantStepUp(sessionID, userID string, ttl time.Duration, now time.Time) (sessions.StepUpStatus, error)
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
	stepUps      AuthStepUpStore
	policy       *rbac.Policy
	issuer       string
	security     AuthSecurityConfig
	sessionTTL   time.Duration
	challengeTTL time.Duration
	audits       *AuditService
}
