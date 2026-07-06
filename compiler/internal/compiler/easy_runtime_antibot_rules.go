package compiler

import (
	"regexp"
	"sort"
	"strings"
)

func normalizeCompilerAntibotRules(values []AntibotChallengeRuleInput) []AntibotChallengeRuleInput {
	if len(values) == 0 {
		return nil
	}
	items := make([]AntibotChallengeRuleInput, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		path := strings.TrimSpace(value.Path)
		challenge := strings.ToLower(strings.TrimSpace(value.Challenge))
		if path == "" || challenge == "" {
			continue
		}
		key := strings.ToLower(path) + "\x00" + challenge
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		items = append(items, AntibotChallengeRuleInput{
			Path:      path,
			Challenge: challenge,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		leftPriority := customLimitPriority(items[i].Path)
		rightPriority := customLimitPriority(items[j].Path)
		if leftPriority != rightPriority {
			return leftPriority > rightPriority
		}
		if len(items[i].Path) != len(items[j].Path) {
			return len(items[i].Path) > len(items[j].Path)
		}
		if items[i].Path == items[j].Path {
			return items[i].Challenge < items[j].Challenge
		}
		return items[i].Path < items[j].Path
	})
	return items
}

func normalizeCompilerAntibotExclusionRules(values []AntibotExclusionRuleInput) []AntibotExclusionRuleInput {
	if len(values) == 0 {
		return nil
	}
	items := make([]AntibotExclusionRuleInput, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		path := strings.TrimSpace(value.Path)
		if path == "" {
			continue
		}
		methods := sortedUniqueUpper(value.Methods)
		if len(methods) == 0 || hasWildcardMethod(methods) {
			methods = []string{"*"}
		}
		key := strings.ToLower(path) + "\x00" + strings.Join(methods, ",")
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		items = append(items, AntibotExclusionRuleInput{
			Path:    path,
			Methods: methods,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		leftPriority := customLimitPriority(items[i].Path)
		rightPriority := customLimitPriority(items[j].Path)
		if leftPriority != rightPriority {
			return leftPriority > rightPriority
		}
		if len(items[i].Path) != len(items[j].Path) {
			return len(items[i].Path) > len(items[j].Path)
		}
		if items[i].Path == items[j].Path {
			return strings.Join(items[i].Methods, ",") < strings.Join(items[j].Methods, ",")
		}
		return items[i].Path < items[j].Path
	})
	return items
}

func buildEasyAntibotRuleData(rules []AntibotChallengeRuleInput, challengeURI string) []easyAntibotRuleData {
	if len(rules) == 0 {
		return nil
	}
	out := make([]easyAntibotRuleData, 0, len(rules))
	for _, rule := range rules {
		pattern := antibotRuleGuardPattern(rule.Path)
		if pattern == "" {
			continue
		}
		out = append(out, easyAntibotRuleData{
			GuardPattern: pattern,
			Challenge:    rule.Challenge,
			RedirectURI:  antibotRuleRedirectURI(rule.Challenge, challengeURI),
		})
	}
	return out
}

func antibotRuleRedirectURI(mode string, challengeURI string) string {
	if strings.EqualFold(strings.TrimSpace(mode), "no") {
		return ""
	}
	return antibotRedirectURI(mode, challengeURI)
}

func buildEasyAntibotExclusionRuleData(rules []AntibotExclusionRuleInput) []easyAntibotExclusionRuleData {
	if len(rules) == 0 {
		return nil
	}
	out := make([]easyAntibotExclusionRuleData, 0, len(rules))
	for _, rule := range rules {
		pattern := antibotExclusionMatchPattern(rule.Path, rule.Methods)
		if pattern == "" {
			continue
		}
		out = append(out, easyAntibotExclusionRuleData{MatchPattern: pattern})
	}
	return out
}

func antibotRuleGuardPattern(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	if hasComplexWildcard(trimmed) {
		return wildcardPathToRegex(trimmed)
	}
	if strings.HasSuffix(trimmed, "*") {
		base := strings.TrimSuffix(trimmed, "*")
		if base == "" {
			return "^/"
		}
		return "^" + regexp.QuoteMeta(base)
	}
	if strings.HasSuffix(trimmed, "/") {
		return "^" + regexp.QuoteMeta(trimmed)
	}
	return "^" + regexp.QuoteMeta(trimmed) + "$"
}

func antibotExclusionMatchPattern(path string, methods []string) string {
	pathPattern := antibotRuleGuardPattern(path)
	if pathPattern == "" {
		return ""
	}
	methodPattern := antibotExclusionMethodPattern(methods)
	if methodPattern == "" {
		return ""
	}
	return "^" + methodPattern + ":" + strings.TrimPrefix(pathPattern, "^")
}

func antibotExclusionMethodPattern(methods []string) string {
	if len(methods) == 0 || hasWildcardMethod(methods) {
		return "[A-Z]+"
	}
	escaped := make([]string, 0, len(methods))
	for _, method := range methods {
		trimmed := strings.ToUpper(strings.TrimSpace(method))
		if trimmed == "" {
			continue
		}
		escaped = append(escaped, regexp.QuoteMeta(trimmed))
	}
	if len(escaped) == 0 {
		return ""
	}
	return "(?:" + strings.Join(escaped, "|") + ")"
}

func hasWildcardMethod(methods []string) bool {
	for _, method := range methods {
		if strings.TrimSpace(method) == "*" {
			return true
		}
	}
	return false
}

func normalizeBlacklistURIPatterns(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		pattern := strings.TrimSpace(value)
		if pattern == "" {
			continue
		}
		pattern = toSafeBlacklistURIRegex(pattern)
		if pattern == "" {
			continue
		}
		if _, ok := seen[pattern]; ok {
			continue
		}
		seen[pattern] = struct{}{}
		out = append(out, pattern)
	}
	sort.Strings(out)
	return out
}

func toSafeBlacklistURIRegex(pattern string) string {
	// Keep existing regex behavior when the input is already a valid expression.
	if _, err := regexp.Compile(pattern); err == nil {
		return pattern
	}
	// Fallback: interpret as wildcard expression (`*`, `?`) and escape everything else.
	quoted := regexp.QuoteMeta(pattern)
	quoted = strings.ReplaceAll(quoted, `\*`, `.*`)
	quoted = strings.ReplaceAll(quoted, `\?`, `.`)
	return quoted
}

func antibotScannerPattern() string {
	patterns := []string{
		`/(?:\.env|\.git|phpmyadmin|wp-admin|vendor/phpunit|cgi-bin|boaform)`,
		`acunetix`,
		`dirbuster`,
		`dirsearch`,
		`feroxbuster`,
		`ffuf`,
		`gobuster`,
		`gospider`,
		`hakrawler`,
		`masscan`,
		`nessus`,
		`nikto`,
		`nmap`,
		`sqlmap`,
		`wfuzz`,
		`wpscan`,
		`zgrab`,
		`securityscanner`,
	}
	return strings.Join(patterns, "|")
}
