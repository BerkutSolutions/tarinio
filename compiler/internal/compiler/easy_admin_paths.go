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
	"/auth",
	"/auth/verify",
}

var easyAdminPrefixPaths = []string{
	"/api/",
	"/static/",
}

func easyAdminAntibotExclusionRulesForSite(site SiteInput) []AntibotExclusionRuleInput {
	if !isManagementSite(site) {
		return nil
	}
	rules := make([]AntibotExclusionRuleInput, 0, len(easyAdminExactPaths)+len(easyAdminPrefixPaths)+len(easyAdminSegmentPrefixes))
	for _, exact := range easyAdminExactPaths {
		rules = append(rules, AntibotExclusionRuleInput{Path: exact, Methods: []string{"GET", "HEAD"}})
	}
	for _, prefix := range easyAdminPrefixPaths {
		rules = append(rules, AntibotExclusionRuleInput{Path: prefix, Methods: []string{"GET", "HEAD"}})
	}
	for _, prefix := range easyAdminSegmentPrefixes {
		rules = append(rules, AntibotExclusionRuleInput{Path: prefix, Methods: []string{"GET", "HEAD"}})
	}
	return rules
}

func appendAntibotExclusionRules(base []AntibotExclusionRuleInput, extra []AntibotExclusionRuleInput) []AntibotExclusionRuleInput {
	if len(extra) == 0 {
		return base
	}
	merged := make([]AntibotExclusionRuleInput, 0, len(base)+len(extra))
	merged = append(merged, base...)
	merged = append(merged, extra...)
	return normalizeCompilerAntibotExclusionRules(merged)
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
	case "control-plane-access", "control-plane", "ui":
		return true
	default:
		configuredID := strings.ToLower(strings.TrimSpace(os.Getenv("CONTROL_PLANE_DEV_FAST_START_MANAGEMENT_SITE_ID")))
		if normalized == "localhost" {
			return configuredID == "localhost"
		}
		return configuredID != "" && normalized == configuredID
	}
}

func isManagementSite(site SiteInput) bool {
	if isManagementSiteID(site.ID) {
		return true
	}
	primaryHost := strings.ToLower(strings.TrimSpace(site.PrimaryHost))
	if primaryHost == "" {
		return false
	}
	if primaryHost == managementUIHost() {
		return true
	}
	for _, alias := range site.Aliases {
		if strings.ToLower(strings.TrimSpace(alias)) == managementUIHost() {
			return true
		}
	}
	return false
}

func managementUIHost() string {
	target := strings.TrimSpace(strings.TrimPrefix(uiProxyTarget(), "http://"))
	target = strings.TrimSpace(strings.TrimPrefix(target, "https://"))
	if idx := strings.IndexByte(target, ':'); idx >= 0 {
		target = target[:idx]
	}
	return strings.ToLower(strings.TrimSpace(target))
}

func easyAdminBypassPathPatternForSite(site SiteInput) string {
	if !isManagementSite(site) {
		return "^$"
	}
	return easyAdminBypassPathPattern()
}

var easyReservedLimitExactPaths = []string{
	"/",
	"/login",
	"/login/2fa",
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

