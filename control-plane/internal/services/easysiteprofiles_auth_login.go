package services

import (
	"fmt"
	"strings"
	"time"
)

// MarkBasicAuthLogin records a successful Nginx Basic Auth verification. It
// deliberately changes metadata only: no compiler run or revision is needed.
func (s *EasySiteProfileService) MarkBasicAuthLogin(siteID, username string, when time.Time) error {
	siteID, username = strings.TrimSpace(siteID), strings.TrimSpace(username)
	if siteID == "" || username == "" {
		return fmt.Errorf("site id and username are required")
	}
	profile, found, err := s.store.Get(siteID)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("easy site profile for site %s not found", siteID)
	}
	for index := range profile.SecurityAuthBasic.Users {
		user := &profile.SecurityAuthBasic.Users[index]
		if user.Enabled && user.Username == username {
			user.LastLoginAt = when.UTC().Format(time.RFC3339)
			_, err = s.store.Update(profile)
			return err
		}
	}
	return fmt.Errorf("enabled Basic Auth user %s not found for site %s", username, siteID)
}
