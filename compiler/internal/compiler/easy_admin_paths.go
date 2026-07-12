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

func easyManagementProtectedPaths() []string {
	paths := make([]string, 0, len(easyAdminExactPaths)+len(easyAdminPrefixPaths)+len(easyAdminSegmentPrefixes)+2)
	paths = append(paths, easyAdminExactPaths...)
	paths = append(paths, easyAdminPrefixPaths...)
	paths = append(paths, easyAdminSegmentPrefixes...)
	paths = append(paths, "/login", "/login/2fa")
	return paths
}

func easyAdminAntibotExclusionRulesForSite(site SiteInput) []AntibotExclusionRuleInput {
	return easyAdminMethodExclusionRulesForSite(site, []string{"GET", "HEAD"})
}

func easyAdminAuthExclusionRulesForSite(site SiteInput) []AuthExclusionRuleInput {
	raw := easyAdminMethodExclusionRulesForSite(site, []string{"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE"})
	if len(raw) == 0 {
		return nil
	}
	rules := make([]AuthExclusionRuleInput, 0, len(raw))
	for _, rule := range raw {
		rules = append(rules, AuthExclusionRuleInput{Path: rule.Path, Methods: append([]string(nil), rule.Methods...)})
	}
	return rules
}

func easyAdminMethodExclusionRulesForSite(site SiteInput, methods []string) []AntibotExclusionRuleInput {
	if !isManagementSite(site) {
		return nil
	}
	rules := make([]AntibotExclusionRuleInput, 0, len(easyAdminExactPaths)+len(easyAdminPrefixPaths)+len(easyAdminSegmentPrefixes))
	for _, exact := range easyAdminExactPaths {
		rules = append(rules, AntibotExclusionRuleInput{Path: exact, Methods: methods})
	}
	for _, prefix := range easyAdminPrefixPaths {
		rules = append(rules, AntibotExclusionRuleInput{Path: prefix, Methods: methods})
	}
	for _, prefix := range easyAdminSegmentPrefixes {
		rules = append(rules, AntibotExclusionRuleInput{Path: prefix, Methods: methods})
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
	parts := make([]string, 0, len(easyManagementProtectedPaths()))
	for _, path := range easyManagementProtectedPaths() {
		switch {
		case path == "/":
			parts = append(parts, "")
		case strings.HasSuffix(path, "/"):
			trimmed := strings.TrimSuffix(strings.TrimPrefix(path, "/"), "/")
			parts = append(parts, regexp.QuoteMeta(trimmed)+"/.*")
		default:
			trimmed := strings.TrimPrefix(path, "/")
			parts = append(parts, regexp.QuoteMeta(trimmed)+"(?:/.*)?")
		}
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

func easyModSecurityBypassPathPatternForSite(site SiteInput) string {
	if !isManagementSite(site) {
		return ""
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
