package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"waf/control-plane/internal/passkeys"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

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
