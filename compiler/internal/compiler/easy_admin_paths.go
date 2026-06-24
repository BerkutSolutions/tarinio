package compiler

import (
	"os"
	"regexp"
	"strings"
)

var easyAdminSegmentPrefixes = []string{
	"/dashboard",
	"/sites",
	"/services",
	"/anti-ddos",
	"/tls",
	"/requests",
	"/revisions",
	"/events",
	"/bans",
	"/jobs",
	"/administration",
	"/activity",
	"/settings",
	"/about",
	"/profile",
	"/healthcheck",
	"/onboarding",
}

var easyAdminExactPaths = []string{
	"/",
	"/login",
	"/login/2fa",
	"/auth",
	"/auth/verify",
}

var easyAdminPrefixPaths = []string{
	"/api/",
	"/static/",
}

func isReservedAdminAppPath(path string) bool {
	canonical := strings.TrimSpace(path)
	if canonical == "" {
		return false
	}
	for _, exact := range easyAdminExactPaths {
		if canonical == exact {
			return true
		}
	}
	for _, prefix := range easyAdminPrefixPaths {
		if strings.HasPrefix(canonical, prefix) {
			return true
		}
	}
	for _, prefix := range easyAdminSegmentPrefixes {
		if canonical == prefix || strings.HasPrefix(canonical, prefix+"/") {
			return true
		}
	}
	return false
}

func easyAdminBypassPathPattern() string {
	parts := []string{
		"",
		"login",
		"login/2fa",
		"auth",
		"auth/verify",
	}
	for _, prefix := range easyAdminSegmentPrefixes {
		trimmed := strings.TrimPrefix(prefix, "/")
		parts = append(parts, regexp.QuoteMeta(trimmed)+"(?:/.*)?")
	}
	for _, prefix := range easyAdminPrefixPaths {
		trimmed := strings.TrimSuffix(strings.TrimPrefix(prefix, "/"), "/")
		parts = append(parts, regexp.QuoteMeta(trimmed)+"/.*")
	}
	return "^/(?:" + strings.Join(parts, "|") + ")$"
}

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

func easyAdminBypassPathPatternForSite(siteID string) string {
	if !isManagementSiteID(siteID) {
		return "^$"
	}
	return easyAdminBypassPathPattern()
}

var easyReservedLimitExactPaths = []string{
	"/",
	"/api/",
	"/auth",
	"/auth/verify",
}

var easyReservedLimitPrefixPaths = []string{
	"/static/",
}

var easyReservedLimitSegmentPrefixes = []string{
	"/dashboard",
	"/sites",
	"/services",
	"/anti-ddos",
	"/tls",
	"/requests",
	"/revisions",
	"/events",
	"/bans",
	"/jobs",
	"/administration",
	"/activity",
	"/settings",
	"/about",
	"/profile",
	"/healthcheck",
	"/onboarding",
}

func isReservedEasyLimitPath(path string) bool {
	canonical := strings.TrimSpace(path)
	if canonical == "" {
		return false
	}
	for _, exact := range easyReservedLimitExactPaths {
		if canonical == exact {
			return true
		}
	}
	for _, prefix := range easyReservedLimitPrefixPaths {
		if strings.HasPrefix(canonical, prefix) {
			return true
		}
	}
	for _, prefix := range easyReservedLimitSegmentPrefixes {
		if canonical == prefix || strings.HasPrefix(canonical, prefix+"/") {
			return true
		}
	}
	return false
}
