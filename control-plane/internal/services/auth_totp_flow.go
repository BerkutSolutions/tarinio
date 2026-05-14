package services

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"waf/control-plane/internal/audits"
	"waf/control-plane/internal/auth"
	"waf/control-plane/internal/users"

	"github.com/skip2/go-qrcode"
)

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
	user, verified, err := s.verifyAndUpgradePassword(user, password)
	if err != nil {
		return AuthUser{}, err
	}
	if !verified {
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
