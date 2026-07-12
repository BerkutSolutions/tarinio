package compiler

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func buildEasyModSecurityRules(
	siteID,
	securityMode string,
	useCRSPlugins bool,
	crsVersion string,
	plugins []string,
	exclusionRules []ModSecurityExclusionRuleInput,
	useCustomConfiguration bool,
	customPath,
	customContent string,
	useAPIPositiveSecurity bool,
	openAPISchemaRef,
	apiEnforcementMode,
	apiDefaultAction string,
	apiEndpointPolicies []APIPositiveEndpointPolicyInput,
	virtualPatches []VirtualPatchInput,
) string {
	engineDirective := "SecRuleEngine On"
	switch strings.ToLower(strings.TrimSpace(securityMode)) {
	case "monitor":
		engineDirective = "SecRuleEngine DetectionOnly"
	case "transparent":
		engineDirective = "SecRuleEngine Off"
	default:
		engineDirective = "SecRuleEngine On"
	}
	lines := []string{
		"# Easy ModSecurity directives",
		"# site: " + siteID,
		"# security_mode: " + strings.TrimSpace(securityMode),
		"# crs_version: " + strings.TrimSpace(crsVersion),
		engineDirective,
	}
	if engineDirective != "SecRuleEngine Off" {
		lines = append(lines,
			"Include /etc/waf/modsecurity/crs-setup.conf",
			"Include /etc/waf/modsecurity/coreruleset/rules/*.conf",
		)
	}
	if useCRSPlugins && len(plugins) > 0 {
		lines = append(lines, "# crs_plugins: "+strings.Join(plugins, " "))
		for _, plugin := range plugins {
			lines = append(lines, fmt.Sprintf("Include /etc/waf/modsecurity/crs-overrides/%s.conf", strings.TrimSpace(plugin)))
		}
	}
	if len(exclusionRules) > 0 {
		lines = append(lines, buildModSecurityExclusionRules(exclusionRules)...)
	}
	if useCustomConfiguration && strings.TrimSpace(customPath) != "" {
		lines = append(lines, "# custom_path: "+strings.TrimSpace(customPath))
	}
	if useCustomConfiguration && strings.TrimSpace(customContent) != "" {
		lines = append(lines, customContent)
	}
	if useAPIPositiveSecurity {
		lines = append(lines, buildAPIPositiveModSecurityRules(siteID, openAPISchemaRef, apiEnforcementMode, apiDefaultAction, apiEndpointPolicies)...)
	}
	if len(virtualPatches) > 0 {
		lines = append(lines, buildVirtualPatchModSecurityRules(virtualPatches)...)
	}
	return strings.Join(lines, "\n") + "\n"
}

func buildModSecurityExclusionRules(rules []ModSecurityExclusionRuleInput) []string {
	lines := []string{"# structured_exclusion_rules"}
	ruleID := 191000
	for _, rule := range rules {
		matcher, matcherOperator := buildExclusionPathMatcher(rule)
		if matcher == "" {
			continue
		}
		methodsPattern := strings.Join(rule.Methods, "|")
		actions := []string{"t:none"}
		if len(rule.Targets) > 0 {
			for _, target := range rule.Targets {
				for _, excludedRuleID := range rule.RuleIDs {
					actions = append(actions, fmt.Sprintf("ctl:ruleRemoveTargetById=%d;%s", excludedRuleID, target))
				}
			}
		} else {
			for _, excludedRuleID := range rule.RuleIDs {
				actions = append(actions, fmt.Sprintf("ctl:ruleRemoveById=%d", excludedRuleID))
			}
		}
		lines = append(lines,
			fmt.Sprintf("# exclusion_comment: %s", strings.TrimSpace(rule.Comment)),
			fmt.Sprintf(`SecRule REQUEST_METHOD "@rx ^(?:%s)$" "id:%d,phase:1,t:none,chain,pass,nolog"`, methodsPattern, ruleID),
			fmt.Sprintf(`SecRule REQUEST_URI "%s %s" "%s"`, matcherOperator, matcher, strings.Join(actions, ",")),
		)
		ruleID += 1
	}
	return lines
}

func buildExclusionPathMatcher(rule ModSecurityExclusionRuleInput) (string, string) {
	mode := strings.ToLower(strings.TrimSpace(rule.Mode))
	switch mode {
	case "regex":
		if strings.TrimSpace(rule.PathPattern) == "" {
			return "", ""
		}
		return strings.TrimSpace(rule.PathPattern), "@rx"
	case "prefix":
		path := strings.TrimSpace(rule.Path)
		if path == "" {
			return "", ""
		}
		return "^" + regexp.QuoteMeta(path) + "(?:$|/)", "@rx"
	default:
		path := strings.TrimSpace(rule.Path)
		if path == "" {
			return "", ""
		}
		return "^" + regexp.QuoteMeta(path) + "$", "@rx"
	}
}

func buildAPIPositiveModSecurityRules(siteID, schemaRef, enforcementMode, defaultAction string, policies []APIPositiveEndpointPolicyInput) []string {
	lines := []string{
		"# API Positive Security directives",
		"# api_positive_site: " + siteID,
		"# api_positive_schema_ref: " + strings.TrimSpace(schemaRef),
		"# api_positive_enforcement_mode: " + strings.TrimSpace(enforcementMode),
		"# api_positive_default_action: " + strings.TrimSpace(defaultAction),
	}

	blocking := strings.ToLower(strings.TrimSpace(enforcementMode)) == "block"
	defaultDeny := strings.ToLower(strings.TrimSpace(defaultAction)) == "deny"
	ruleID := 190000
	knownPaths := make([]string, 0, len(policies))

	for _, policy := range policies {
		path := strings.TrimSpace(policy.Path)
		if path == "" {
			continue
		}
		knownPaths = append(knownPaths, regexp.QuoteMeta(path))

		mode := strings.ToLower(strings.TrimSpace(policy.Mode))
		policyBlocking := blocking
		if mode == "block" {
			policyBlocking = true
		}
		if mode == "monitor" {
			policyBlocking = false
		}
		action := "pass,log"
		if policyBlocking {
			action = "deny,status:403,log"
		}

		if len(policy.Methods) > 0 {
			lines = append(lines,
				fmt.Sprintf(`SecRule REQUEST_URI "@rx ^%s(?:$|/)" "id:%d,phase:1,t:none,chain,pass,nolog"`, regexp.QuoteMeta(path), ruleID),
				fmt.Sprintf(`SecRule REQUEST_METHOD "!@within %s" "t:none,%s,msg:'API positive security: method mismatch for %s'"`, strings.Join(policy.Methods, " "), action, path),
			)
			ruleID += 2
		}
		if len(policy.TokenIDs) > 0 {
			tokenPattern := "(?:" + strings.Join(quoteMetaList(policy.TokenIDs), "|") + ")"
			lines = append(lines,
				fmt.Sprintf(`SecRule REQUEST_URI "@rx ^%s(?:$|/)" "id:%d,phase:1,t:none,chain,pass,nolog"`, regexp.QuoteMeta(path), ruleID),
				fmt.Sprintf(`SecRule REQUEST_HEADERS:X-WAF-API-TOKEN-ID "!@rx ^%s$" "t:none,%s,msg:'API positive security: token mismatch for %s'"`, tokenPattern, action, path),
			)
			ruleID += 2
		}
		if len(policy.ContentTypes) > 0 {
			contentTypePattern := "(?:" + strings.Join(quoteMetaList(policy.ContentTypes), "|") + ")"
			lines = append(lines,
				fmt.Sprintf(`SecRule REQUEST_URI "@rx ^%s(?:$|/)" "id:%d,phase:1,t:none,chain,pass,nolog"`, regexp.QuoteMeta(path), ruleID),
				fmt.Sprintf(`SecRule REQUEST_HEADERS:Content-Type "!@rx ^%s(?:;|$)" "t:none,%s,msg:'API positive security: content-type mismatch for %s'"`, contentTypePattern, action, path),
			)
			ruleID += 2
		}
	}

	if defaultDeny && len(knownPaths) > 0 {
		action := "pass,log"
		if blocking {
			action = "deny,status:403,log"
		}
		lines = append(lines,
			fmt.Sprintf(`SecRule REQUEST_URI "!@rx ^(?:%s)(?:$|/)" "id:%d,phase:1,t:none,%s,msg:'API positive security: unknown endpoint'"`, strings.Join(knownPaths, "|"), ruleID, action),
		)
	}

	return lines
}

func quoteMetaList(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		v := strings.TrimSpace(value)
		if v == "" {
			continue
		}
		out = append(out, regexp.QuoteMeta(v))
	}
	return out
}

func parseRatePerSecond(value string) int {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "")
	value = strings.TrimSuffix(value, "r/s")
	v, err := strconv.Atoi(value)
	if err != nil || v <= 0 {
		return 0
	}
	return v
}

func buildHSTSHeaderValue(profile EasyProfileInput) string {
	if !profile.HSTSEnabled {
		return ""
	}
	maxAge := profile.HSTSMaxAgeSeconds
	if maxAge <= 0 {
		maxAge = 15552000
	}
	parts := []string{fmt.Sprintf("max-age=%d", maxAge)}
	if profile.HSTSIncludeSubdomains {
		parts = append(parts, "includeSubDomains")
	}
	if profile.HSTSPreload {
		parts = append(parts, "preload")
	}
	return strings.Join(parts, "; ")
}
