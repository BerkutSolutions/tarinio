package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"waf/control-plane/internal/audits"
	"waf/control-plane/internal/auth"
	"waf/control-plane/internal/sessions"
)

const stepUpAssertionTTL = 10 * time.Minute

func (s *AuthService) StepUpTOTP(ctx context.Context, sessionID, code string) (result StepUpTOTPResult, err error) {
	session, err := s.Authenticate(sessionID)
	if err != nil {
		return StepUpTOTPResult{}, err
	}
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			ActorUserID: session.UserID, Action: "auth.step_up_totp", ResourceType: "auth",
			ResourceID: session.UserID, Status: auditStatus(err), Summary: "TOTP step-up verification",
		})
	}()
	if s.stepUps == nil {
		return StepUpTOTPResult{}, errors.New("step-up verification is unavailable")
	}
	now := time.Now().UTC()
	status, err := s.stepUps.StepUpStatus(session.SessionID, session.UserID, now)
	if err != nil {
		return StepUpTOTPResult{}, err
	}
	if status.Locked {
		return StepUpTOTPResult{RetryAfterSeconds: status.RetryAfterSeconds}, stepUpLockedError(status)
	}
	user, ok, err := s.users.Get(session.UserID)
	if err != nil {
		return StepUpTOTPResult{}, err
	}
	if !ok || !user.IsActive || !user.TOTPEnabled {
		return StepUpTOTPResult{}, errors.New("TOTP must be enabled for step-up verification")
	}
	updatedUser, secret, migrated, err := s.resolveUserTOTPSecret(user)
	if err != nil || strings.TrimSpace(secret) == "" || !auth.VerifyTOTP(secret, code, now) {
		failed, failureErr := s.stepUps.RecordStepUpFailure(session.UserID, now)
		if failureErr != nil {
			return StepUpTOTPResult{}, failureErr
		}
		if failed.Locked {
			return StepUpTOTPResult{RetryAfterSeconds: failed.RetryAfterSeconds}, stepUpLockedError(failed)
		}
		return StepUpTOTPResult{}, errors.New("invalid TOTP code")
	}
	if migrated {
		if _, err := s.users.Update(updatedUser); err != nil {
			return StepUpTOTPResult{}, err
		}
	}
	granted, err := s.stepUps.GrantStepUp(session.SessionID, session.UserID, stepUpAssertionTTL, now)
	if errors.Is(err, sessions.ErrStepUpLocked) {
		return StepUpTOTPResult{RetryAfterSeconds: granted.RetryAfterSeconds}, stepUpLockedError(granted)
	}
	if err != nil {
		return StepUpTOTPResult{}, err
	}
	return StepUpTOTPResult{OK: true}, nil
}

func (s *AuthService) RequireStepUp(sessionID string) error {
	session, err := s.Authenticate(sessionID)
	if err != nil {
		return err
	}
	user, ok, err := s.users.Get(session.UserID)
	if err != nil {
		return err
	}
	if !ok || !user.IsActive {
		return errors.New("user not found")
	}
	// Existing users without 2FA retain their pre-upgrade export path. Users
	// who enrolled TOTP must prove a fresh factor before secret export.
	if !user.TOTPEnabled {
		return nil
	}
	if s.stepUps == nil {
		return errors.New("step-up verification is unavailable")
	}
	status, err := s.stepUps.StepUpStatus(session.SessionID, session.UserID, time.Now().UTC())
	if err != nil {
		return err
	}
	if status.Locked {
		return stepUpLockedError(status)
	}
	if !status.Verified {
		return errors.New("fresh TOTP step-up verification is required")
	}
	return nil
}

func stepUpLockedError(status sessions.StepUpStatus) error {
	return fmt.Errorf("%w: retry after %d seconds", sessions.ErrStepUpLocked, status.RetryAfterSeconds)
}
