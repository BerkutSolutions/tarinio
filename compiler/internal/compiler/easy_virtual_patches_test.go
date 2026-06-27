package compiler

import (
	"strings"
	"testing"
)

func TestVirtualPatch_BlockRuleInModSecurity(t *testing.T) {
	patches := []VirtualPatchInput{
		{ID: "vp-1", Pattern: "/test-vuln-path", Target: "uri", Action: "block"},
	}
	rules := buildEasyModSecurityRules(
		"site-1", "block", false, "", nil, false, "", "", false, "", "", "", nil, patches,
	)
	if !strings.Contains(rules, `SecRule REQUEST_URI "@rx /test-vuln-path"`) {
		t.Errorf("expected REQUEST_URI rule, got:\n%s", rules)
	}
	if !strings.Contains(rules, "deny,status:403") {
		t.Errorf("expected deny,status:403 action, got:\n%s", rules)
	}
	if !strings.Contains(rules, "Virtual patch vp-1") {
		t.Errorf("expected patch id in msg, got:\n%s", rules)
	}
}

func TestVirtualPatch_MonitorRuleInModSecurity(t *testing.T) {
	patches := []VirtualPatchInput{
		{ID: "vp-2", Pattern: `DROP\s+TABLE`, Target: "body", Action: "monitor"},
	}
	rules := buildEasyModSecurityRules(
		"site-1", "block", false, "", nil, false, "", "", false, "", "", "", nil, patches,
	)
	if !strings.Contains(rules, "REQUEST_BODY") {
		t.Errorf("expected REQUEST_BODY target, got:\n%s", rules)
	}
	if strings.Contains(rules, "deny") {
		t.Errorf("monitor rule should not contain deny, got:\n%s", rules)
	}
	if !strings.Contains(rules, "pass") {
		t.Errorf("expected pass action for monitor, got:\n%s", rules)
	}
}

func TestVirtualPatch_NoPatchesNoSection(t *testing.T) {
	rules := buildEasyModSecurityRules(
		"site-1", "block", false, "", nil, false, "", "", false, "", "", "", nil, nil,
	)
	if strings.Contains(rules, "Virtual Patch rules") {
		t.Errorf("expected no virtual patch section when patches is nil, got:\n%s", rules)
	}
}

func TestVirtualPatch_ExpiredPatchSkipped(t *testing.T) {
	// Empty patch (no pattern) should be skipped gracefully.
	patches := []VirtualPatchInput{
		{ID: "vp-empty", Pattern: "", Target: "uri", Action: "block"},
	}
	rules := buildEasyModSecurityRules(
		"site-1", "block", false, "", nil, false, "", "", false, "", "", "", nil, patches,
	)
	if strings.Contains(rules, "vp-empty") {
		t.Errorf("empty pattern patch should be skipped, got:\n%s", rules)
	}
}

func TestVirtualPatch_HeaderTarget(t *testing.T) {
	patches := []VirtualPatchInput{
		{ID: "vp-h", Pattern: "X-Evil.*", Target: "header", Action: "block"},
	}
	rules := buildEasyModSecurityRules(
		"site-1", "block", false, "", nil, false, "", "", false, "", "", "", nil, patches,
	)
	if !strings.Contains(rules, "REQUEST_HEADERS") {
		t.Errorf("expected REQUEST_HEADERS target, got:\n%s", rules)
	}
}
