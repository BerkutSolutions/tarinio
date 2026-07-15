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
	"waf/control-plane/internal/users"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

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
		if len(items) == 0 {
			return PasskeyBeginResult{}, errors.New("auth.passkeys.notFound")
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
