package services

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"waf/control-plane/internal/passkeys"
	"waf/control-plane/internal/users"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

type webAuthnUser struct {
	user        users.User
	credentials []webauthn.Credential
}

func newWebAuthnUser(user users.User, items []passkeys.Passkey) webauthn.User {
	creds := make([]webauthn.Credential, 0, len(items))
	for _, item := range items {
		idRaw, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(item.CredentialID))
		if err != nil || len(idRaw) == 0 {
			continue
		}
		transports := make([]protocol.AuthenticatorTransport, 0, len(item.Transports))
		for _, transport := range item.Transports {
			transport = strings.TrimSpace(transport)
			if transport == "" {
				continue
			}
			transports = append(transports, protocol.AuthenticatorTransport(transport))
		}
		creds = append(creds, webauthn.Credential{
			ID:              idRaw,
			PublicKey:       append([]byte(nil), item.PublicKey...),
			AttestationType: item.AttestationType,
			Transport:       transports,
			Authenticator: webauthn.Authenticator{
				AAGUID:    append([]byte(nil), item.AAGUID...),
				SignCount: item.SignCount,
			},
		})
	}
	return &webAuthnUser{user: user, credentials: creds}
}

func (u *webAuthnUser) WebAuthnID() []byte {
	if u == nil {
		return []byte("u:")
	}
	return []byte("u:" + u.user.ID)
}

func (u *webAuthnUser) WebAuthnName() string {
	if u == nil {
		return ""
	}
	return u.user.Username
}

func (u *webAuthnUser) WebAuthnDisplayName() string {
	if u == nil {
		return ""
	}
	if strings.TrimSpace(u.user.Username) != "" {
		return u.user.Username
	}
	return u.user.ID
}

func (u *webAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	if u == nil {
		return nil
	}
	return u.credentials
}

func toJSONMap(value any) (map[string]any, error) {
	buf, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(buf, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func parseUserIDFromWebAuthnHandle(handle []byte) (string, error) {
	raw := strings.TrimSpace(string(handle))
	if !strings.HasPrefix(raw, "u:") {
		return "", fmt.Errorf("invalid user handle")
	}
	id := strings.TrimSpace(strings.TrimPrefix(raw, "u:"))
	if id == "" {
		return "", fmt.Errorf("invalid user handle")
	}
	return id, nil
}
