package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"waf/control-plane/internal/audits"
)

func (s *AuthService) Me(sessionID string) (AuthUser, error) {
	session, err := s.Authenticate(sessionID)
	if err != nil {
		return AuthUser{}, err
	}
	user, err := s.userByID(session.UserID)
	if err != nil {
		return AuthUser{}, err
	}
	if current, ok, getErr := s.sessions.GetSession(session.SessionID); getErr == nil && ok {
		user.SessionCreatedAt = strings.TrimSpace(current.CreatedAt)
		user.SessionExpiresAt = strings.TrimSpace(current.ExpiresAt)
	}
	lastLoginIP, frequentLoginIP, passwordChangedAt := s.userSecuritySummary(session.UserID)
	user.LastLoginIP = lastLoginIP
	user.FrequentLoginIP = frequentLoginIP
	user.PasswordChangedAt = passwordChangedAt
	return user, nil
}

func (s *AuthService) UpdatePreferences(ctx context.Context, sessionID string, input AuthUserPreferences) (result AuthUser, err error) {
	session, err := s.Authenticate(sessionID)
	if err != nil {
		return AuthUser{}, err
	}
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			ActorUserID:  session.UserID,
			Action:       "auth.preferences_update",
			ResourceType: "user",
			ResourceID:   session.UserID,
			Status:       auditStatus(err),
			Summary:      "update personal preferences",
			Details: map[string]any{
				"language": normalizeUserLanguage(input.Language),
			},
		})
	}()
	user, ok, err := s.users.Get(session.UserID)
	if err != nil {
		return AuthUser{}, err
	}
	if !ok {
		return AuthUser{}, errors.New("user not found")
	}
	user.Language = normalizeUserLanguage(input.Language)
	if _, err := s.users.Update(user); err != nil {
		return AuthUser{}, err
	}
	return s.userByID(user.ID)
}

func normalizeAuditResource(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "unknown"
	}
	return value
}

func auditBootstrapDetails(email string) map[string]any {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return nil
	}
	return map[string]any{
		"email": email,
	}
}

func parseAuditTimestamp(value string) (time.Time, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, false
	}
	if parsed, err := time.Parse(time.RFC3339Nano, trimmed); err == nil {
		return parsed, true
	}
	if parsed, err := time.Parse(time.RFC3339, trimmed); err == nil {
		return parsed, true
	}
	return time.Time{}, false
}

func (s *AuthService) userSecuritySummary(userID string) (string, string, string) {
	if s == nil || s.audits == nil {
		return "", "", ""
	}
	result, err := s.audits.List(audits.Query{
		ActorUserID: strings.TrimSpace(userID),
		Limit:       1000,
		Offset:      0,
	})
	if err != nil {
		return "", "", ""
	}
	loginActions := map[string]struct{}{
		"auth.login": {},
		"auth.2fa":   {},
	}
	ipHits := map[string]int{}
	var (
		lastLoginIP       string
		lastLoginAt       time.Time
		passwordChanged   string
		passwordChangedAt time.Time
	)
	for _, event := range result.Items {
		action := strings.TrimSpace(event.Action)
		status := strings.TrimSpace(string(event.Status))
		actorIP := strings.TrimSpace(event.ActorIP)
		occurredAt, ok := parseAuditTimestamp(event.OccurredAt)
		if !ok {
			continue
		}
		if action == "auth.change_password" && status == string(audits.StatusSucceeded) {
			if passwordChanged == "" || occurredAt.After(passwordChangedAt) {
				passwordChangedAt = occurredAt
				passwordChanged = strings.TrimSpace(event.OccurredAt)
			}
		}
		if _, isLoginAction := loginActions[action]; !isLoginAction || status != string(audits.StatusSucceeded) || actorIP == "" {
			continue
		}
		ipHits[actorIP]++
		if lastLoginIP == "" || occurredAt.After(lastLoginAt) {
			lastLoginAt = occurredAt
			lastLoginIP = actorIP
		}
	}
	frequentLoginIP := ""
	frequentHits := 0
	for ip, hits := range ipHits {
		if hits > frequentHits || (hits == frequentHits && (frequentLoginIP == "" || strings.Compare(ip, frequentLoginIP) < 0)) {
			frequentHits = hits
			frequentLoginIP = ip
		}
	}
	return lastLoginIP, frequentLoginIP, passwordChanged
}
