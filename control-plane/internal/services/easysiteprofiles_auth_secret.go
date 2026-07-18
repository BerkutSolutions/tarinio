package services

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"waf/control-plane/internal/audits"
)

// RevealAuthBasicPassword returns one saved Basic Auth password only after the
// caller has been authorized by the HTTP layer for a write-level operation.
func (s *EasySiteProfileService) RevealAuthBasicPassword(ctx context.Context, siteID, username string) (password string, err error) {
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			Action:       "easysiteprofile.auth_password.reveal",
			ResourceType: "easysiteprofile",
			ResourceID:   siteID,
			SiteID:       siteID,
			Status:       auditStatus(err),
			Summary:      "basic auth password revealed",
		})
	}()

	site, err := s.ensureSiteExists(siteID)
	if err != nil {
		return "", err
	}
	profile, ok, err := s.store.Get(site.ID)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("basic auth user %s not found", username)
	}
	needle := strings.ToLower(strings.TrimSpace(username))
	for _, user := range profile.SecurityAuthBasic.Users {
		if strings.ToLower(strings.TrimSpace(user.Username)) == needle {
			return user.Password, nil
		}
	}
	return "", fmt.Errorf("basic auth user %s not found", username)
}

func authPasswordLength(value string) int {
	return utf8.RuneCountInString(value)
}
