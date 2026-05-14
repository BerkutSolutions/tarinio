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
