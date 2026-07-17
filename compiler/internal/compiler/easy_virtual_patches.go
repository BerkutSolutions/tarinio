package compiler

import (
	"fmt"
	"regexp"
	"strings"
)

// buildVirtualPatchModSecurityRules generates ModSecurity SecRule directives
// for each active virtual patch. Rules start at ID 200000.
func buildVirtualPatchModSecurityRules(patches []VirtualPatchInput) []string {
	if len(patches) == 0 {
		return nil
	}
	lines := []string{"# Virtual Patch rules"}
	ruleID := 200000
	for _, p := range patches {
		pattern := strings.TrimSpace(p.Pattern)
		if pattern == "" {
			continue
		}
		if _, err := regexp.Compile(pattern); err != nil {
			continue
		}
		if strings.IndexFunc(pattern, func(r rune) bool { return r < 0x20 || r == 0x7f }) >= 0 {
			continue
		}
		target := targetVariable(p.Target)
		action := secRuleAction(p.Action, ruleID, p.ID)
		lines = append(lines,
			fmt.Sprintf(`SecRule %s "@rx %s" "%s"`, target, escapeModSecurityQuoted(pattern), action),
		)
		ruleID++
	}
	return lines
}

// targetVariable maps a VirtualPatch target to a ModSecurity REQUEST_* variable.
func targetVariable(target string) string {
	switch strings.ToLower(strings.TrimSpace(target)) {
	case "body":
		return "REQUEST_BODY"
	case "header":
		return "REQUEST_HEADERS"
	default:
		return "REQUEST_URI"
	}
}

// secRuleAction builds the ModSecurity action string for a virtual patch rule.
func secRuleAction(action string, ruleID int, patchID string) string {
	msg := fmt.Sprintf("Virtual patch %s", strings.ReplaceAll(patchID, "'", "\\'"))
	base := fmt.Sprintf("id:%d,phase:2,t:none,log,msg:'%s'", ruleID, msg)
	if strings.ToLower(strings.TrimSpace(action)) == "block" {
		return base + ",deny,status:403"
	}
	return base + ",pass"
}

func escapeModSecurityQuoted(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	return strings.ReplaceAll(value, `"`, `\"`)
}
