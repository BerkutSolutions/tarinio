package compiler

import (
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

var easyReservedLimitExactPaths = []string{
	"/",
	"/api/",
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
