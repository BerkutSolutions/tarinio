package services

import (
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
)

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
