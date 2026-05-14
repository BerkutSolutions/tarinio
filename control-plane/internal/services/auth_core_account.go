package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"waf/control-plane/internal/audits"
	"waf/control-plane/internal/auth"
	"waf/control-plane/internal/rbac"
	"waf/control-plane/internal/users"
)

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
	user, verified, err := s.verifyAndUpgradePassword(user, currentPassword)
	if err != nil {
		return err
	}
	if !verified {
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
	if _, err = s.users.Update(user); err != nil {
		return err
	}
	return s.sessions.DeleteSessionsByUserExcept(user.ID, sessionID)
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
	user, ok, err := s.users.Get(session.UserID)
	if err != nil {
		return auth.SessionView{}, err
	}
	if !ok || !user.IsActive {
		_ = s.sessions.DeleteSession(session.ID)
		return auth.SessionView{}, errors.New("session not found")
	}
	return auth.SessionView{
		SessionID:   session.ID,
		UserID:      user.ID,
		Username:    user.Username,
		RoleIDs:     append([]string(nil), user.RoleIDs...),
		Permissions: s.policy.Permissions(user.RoleIDs),
	}, nil
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
		FullName:               user.FullName,
		Department:             user.Department,
		Position:               user.Position,
		Language:               normalizeUserLanguage(user.Language),
		RoleIDs:                append([]string(nil), user.RoleIDs...),
		Permissions:            s.policy.Permissions(user.RoleIDs),
		TOTPEnabled:            user.TOTPEnabled,
		RecoveryCodesRemaining: countUnusedRecoveryCodes(user),
	}, nil
}

func (s *AuthService) verifyAndUpgradePassword(user users.User, password string) (users.User, bool, error) {
	if !users.VerifyPassword(password, user.PasswordHash) {
		return user, false, nil
	}
	if !users.NeedsPasswordRehash(user.PasswordHash) {
		return user, true, nil
	}
	passwordHash, err := users.HashPassword(password)
	if err != nil {
		return user, true, err
	}
	user.PasswordHash = passwordHash
	updated, err := s.users.Update(user)
	if err != nil {
		return user, true, err
	}
	return updated, true, nil
}
