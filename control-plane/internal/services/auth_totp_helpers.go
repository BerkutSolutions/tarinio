package services

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
	"time"

	"waf/control-plane/internal/auth"
	"waf/control-plane/internal/users"
)

func consumeRecoveryCode(user *users.User, recoveryCode, pepper string, now time.Time) bool {
	if user == nil {
		return false
	}
	for i := range user.TOTPRecoveryHashes {
		item := user.TOTPRecoveryHashes[i]
		if strings.TrimSpace(item.UsedAt) != "" {
			continue
		}
		if auth.VerifyRecoveryCode(recoveryCode, pepper, item.Hash, item.Salt) {
			user.TOTPRecoveryHashes[i].UsedAt = now.UTC().Format(time.RFC3339Nano)
			return true
		}
	}
	return false
}

func countUnusedRecoveryCodes(user users.User) int {
	count := 0
	for _, item := range user.TOTPRecoveryHashes {
		if strings.TrimSpace(item.UsedAt) == "" {
			count++
		}
	}
	return count
}

func normalizeUserLanguage(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return ""
	case "ru", "en", "de", "sr", "zh":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func newB64URL(size int) string {
	if size <= 0 {
		size = 32
	}
	buf := make([]byte, size)
	_, _ = rand.Read(buf)
	return base64.RawURLEncoding.EncodeToString(buf)
}

func normalizeAuthSecurityConfig(input AuthSecurityConfig) AuthSecurityConfig {
	cfg := input
	cfg.Pepper = strings.TrimSpace(cfg.Pepper)
	if cfg.Pepper == "" {
		cfg.Pepper = "waf-dev-pepper-change-me"
	}
	cfg.WebAuthn.RPID = strings.ToLower(strings.TrimSpace(cfg.WebAuthn.RPID))
	cfg.WebAuthn.RPName = strings.TrimSpace(cfg.WebAuthn.RPName)
	if cfg.WebAuthn.RPName == "" {
		cfg.WebAuthn.RPName = "TARINIO"
	}
	if !cfg.WebAuthn.Enabled {
		cfg.WebAuthn.Origins = nil
		return cfg
	}
	out := make([]string, 0, len(cfg.WebAuthn.Origins))
	for _, item := range cfg.WebAuthn.Origins {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	cfg.WebAuthn.Origins = out
	return cfg
}

func (s *AuthService) resolveUserTOTPSecret(user users.User) (users.User, string, bool, error) {
	if strings.TrimSpace(user.TOTPSecretEnc) != "" {
		secret, err := auth.DecryptTOTPSecret(user.TOTPSecretEnc, s.security.Pepper)
		if err != nil {
			return user, "", false, err
		}
		return user, secret, false, nil
	}
	secret := strings.TrimSpace(user.TOTPSecret)
	if secret == "" {
		return user, "", false, nil
	}
	secretEnc, err := auth.EncryptTOTPSecret(secret, s.security.Pepper)
	if err != nil {
		return user, "", false, err
	}
	user.TOTPSecretEnc = secretEnc
	user.TOTPSecret = ""
	return user, secret, true, nil
}
