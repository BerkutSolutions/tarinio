package services

import (
	"os"
	"strings"
)

func isManagementSiteID(siteID string) bool {
	normalized := strings.ToLower(strings.TrimSpace(siteID))
	switch normalized {
	case "control-plane-access", "control-plane", "ui", "localhost":
		return true
	default:
		configuredID := strings.ToLower(strings.TrimSpace(os.Getenv("CONTROL_PLANE_DEV_FAST_START_MANAGEMENT_SITE_ID")))
		return configuredID != "" && normalized == configuredID
	}
}
