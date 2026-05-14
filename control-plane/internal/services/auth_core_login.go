package services

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"

	"waf/control-plane/internal/audits"
	"waf/control-plane/internal/auth"
	"waf/control-plane/internal/users"
)

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
	user, verified, err := s.verifyAndUpgradePassword(user, password)
	if err != nil {
		return LoginResult{}, err
	}
	if !verified {
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
