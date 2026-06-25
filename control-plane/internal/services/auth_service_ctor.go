package services

import (
	"strings"
	"time"

	"waf/control-plane/internal/rbac"
)

func NewAuthService(users AuthUserStore, roles AuthRoleStore, sessions AuthSessionStore, passkeys AuthPasskeyStore, issuer string, security AuthSecurityConfig, audits *AuditService) *AuthService {
	normalizedSecurity := normalizeAuthSecurityConfig(security)
	return &AuthService{
		users:        users,
		roles:        roles,
		sessions:     sessions,
		passkeys:     passkeys,
		policy:       rbac.NewPolicy(roles),
		issuer:       strings.TrimSpace(issuer),
		security:     normalizedSecurity,
		sessionTTL:   normalizedSecurity.SessionTTL,
		challengeTTL: 5 * time.Minute,
		audits:       audits,
	}
}
