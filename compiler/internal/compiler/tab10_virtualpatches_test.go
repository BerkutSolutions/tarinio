package compiler

import (
	"strings"
	"testing"
)

// tab10_virtualpatches_test.go — тесты вкладки 10: Виртуальные патчи
// Покрывает: block/monitor action, target uri/body/header,
// pattern в SecRule, multiple patches, empty patches.

// --- VirtualPatch block на uri ---

func TestVirtualPatches_Block_URI_Rule(t *testing.T) {
	rules := buildEasyModSecurityRules(
		"vp-site", "block", false, "", nil, nil, false, "", "", false, "", "", "", nil,
		[]VirtualPatchInput{
			{ID: "vp-uri-1", Pattern: `/admin/secret`, Target: "uri", Action: "block"},
		},
	)
	if !strings.Contains(rules, `SecRule REQUEST_URI "@rx /admin/secret"`) {
		t.Fatalf("expected REQUEST_URI rule for uri target, got:\n%s", rules)
	}
	if !strings.Contains(rules, "deny,status:403") {
		t.Fatalf("expected deny,status:403 for block action, got:\n%s", rules)
	}
}

// --- VirtualPatch block на body ---

func TestVirtualPatches_Block_Body_Rule(t *testing.T) {
	rules := buildEasyModSecurityRules(
		"vp-site", "block", false, "", nil, nil, false, "", "", false, "", "", "", nil,
		[]VirtualPatchInput{
			{ID: "vp-body-1", Pattern: `DROP\s+TABLE`, Target: "body", Action: "block"},
		},
	)
	if !strings.Contains(rules, "REQUEST_BODY") {
		t.Fatalf("expected REQUEST_BODY for body target, got:\n%s", rules)
	}
	if !strings.Contains(rules, "deny,status:403") {
		t.Fatalf("expected deny,status:403 for block action, got:\n%s", rules)
	}
}

// --- VirtualPatch block на header ---

func TestVirtualPatches_Block_Header_Rule(t *testing.T) {
	rules := buildEasyModSecurityRules(
		"vp-site", "block", false, "", nil, nil, false, "", "", false, "", "", "", nil,
		[]VirtualPatchInput{
			{ID: "vp-hdr-1", Pattern: `evilbot`, Target: "header", Action: "block"},
		},
	)
	if !strings.Contains(rules, "REQUEST_HEADERS") {
		t.Fatalf("expected REQUEST_HEADERS for header target, got:\n%s", rules)
	}
	if !strings.Contains(rules, "deny,status:403") {
		t.Fatalf("expected deny,status:403 for block action, got:\n%s", rules)
	}
}

// --- VirtualPatch monitor → pass, нет deny ---

func TestVirtualPatches_Monitor_URI_Rule(t *testing.T) {
	rules := buildEasyModSecurityRules(
		"vp-site", "block", false, "", nil, nil, false, "", "", false, "", "", "", nil,
		[]VirtualPatchInput{
			{ID: "vp-mon-1", Pattern: `/suspicious`, Target: "uri", Action: "monitor"},
		},
	)
	if !strings.Contains(rules, "REQUEST_URI") {
		t.Fatalf("expected REQUEST_URI for uri target in monitor, got:\n%s", rules)
	}
	if strings.Contains(rules, "deny") {
		t.Fatalf("monitor rule should not contain deny, got:\n%s", rules)
	}
	if !strings.Contains(rules, "pass") {
		t.Fatalf("expected pass action for monitor rule, got:\n%s", rules)
	}
}

// --- VirtualPatch monitor на body ---

func TestVirtualPatches_Monitor_Body_Rule(t *testing.T) {
	rules := buildEasyModSecurityRules(
		"vp-site", "block", false, "", nil, nil, false, "", "", false, "", "", "", nil,
		[]VirtualPatchInput{
			{ID: "vp-mon-body", Pattern: `(?i)union.*select`, Target: "body", Action: "monitor"},
		},
	)
	if !strings.Contains(rules, "REQUEST_BODY") {
		t.Fatalf("expected REQUEST_BODY for body target in monitor, got:\n%s", rules)
	}
	if strings.Contains(rules, "deny") {
		t.Fatalf("monitor rule should not contain deny, got:\n%s", rules)
	}
}

// --- Patch ID в сообщении правила ---

func TestVirtualPatches_ID_InRuleMsg(t *testing.T) {
	rules := buildEasyModSecurityRules(
		"vp-site", "block", false, "", nil, nil, false, "", "", false, "", "", "", nil,
		[]VirtualPatchInput{
			{ID: "patch-42", Pattern: `/exploit`, Target: "uri", Action: "block"},
		},
	)
	if !strings.Contains(rules, "patch-42") {
		t.Fatalf("expected patch ID patch-42 in rule msg, got:\n%s", rules)
	}
}

// --- Несколько патчей ---

func TestVirtualPatches_Multiple_Rules(t *testing.T) {
	rules := buildEasyModSecurityRules(
		"vp-site", "block", false, "", nil, nil, false, "", "", false, "", "", "", nil,
		[]VirtualPatchInput{
			{ID: "vp-a", Pattern: `/path-a`, Target: "uri", Action: "block"},
			{ID: "vp-b", Pattern: `/path-b`, Target: "uri", Action: "monitor"},
			{ID: "vp-c", Pattern: `evil-body`, Target: "body", Action: "block"},
		},
	)
	for _, pat := range []string{"/path-a", "/path-b", "evil-body"} {
		if !strings.Contains(rules, pat) {
			t.Fatalf("expected pattern %q in rules, got:\n%s", pat, rules)
		}
	}
}

// --- Нет патчей → нет SecRule виртуальных патчей ---

func TestVirtualPatches_Empty_NoRules(t *testing.T) {
	rules := buildEasyModSecurityRules(
		"vp-site", "block", false, "", nil, nil, false, "", "", false, "", "", "", nil,
		nil,
	)
	if strings.Contains(rules, "Virtual patch") {
		t.Fatalf("did not expect Virtual patch rules when patches nil, got:\n%s", rules)
	}
}

// --- Интеграция: VirtualPatch в артефакте modsec ---

func TestVirtualPatches_Integration_InModsecArtifact(t *testing.T) {
	artifacts, err := RenderEasyArtifacts(
		[]SiteInput{{ID: "vp-int", Enabled: true, PrimaryHost: "vp-int.example.com", ListenHTTP: true}},
		[]EasyProfileInput{{
			SiteID:         "vp-int",
			SecurityMode:   "block",
			AllowedMethods: []string{"GET"},
			MaxClientSize:  "10m",
			UseModSecurity: true,
			VirtualPatches: []VirtualPatchInput{
				{ID: "vp-int-1", Pattern: `/evil-path`, Target: "uri", Action: "block"},
			},
		}},
	)
	if err != nil {
		t.Fatalf("RenderEasyArtifacts: %v", err)
	}
	byPath := artifactsByPath(artifacts)
	content := string(byPath["modsecurity/easy/vp-int.conf"].Content)
	if !strings.Contains(content, "/evil-path") {
		t.Fatalf("expected virtual patch pattern in modsec artifact, got:\n%s", content)
	}
	if !strings.Contains(content, "deny,status:403") {
		t.Fatalf("expected deny,status:403 in modsec artifact, got:\n%s", content)
	}
}

// --- VirtualPatch только в block+UseModSecurity — disabled не блокирует создание артефакта ---

func TestVirtualPatches_SecurityMode_Disabled_ArtifactStillCreated(t *testing.T) {
	// UseModSecurity=true создаёт артефакт независимо от SecurityMode
	artifacts, err := RenderEasyArtifacts(
		[]SiteInput{{ID: "vp-dis", Enabled: true, PrimaryHost: "vp-dis.example.com", ListenHTTP: true}},
		[]EasyProfileInput{{
			SiteID:         "vp-dis",
			SecurityMode:   "disabled",
			AllowedMethods: []string{"GET"},
			MaxClientSize:  "10m",
			UseModSecurity: true,
			VirtualPatches: []VirtualPatchInput{
				{ID: "vp-dis-1", Pattern: `/test`, Target: "uri", Action: "block"},
			},
		}},
	)
	if err != nil {
		t.Fatalf("RenderEasyArtifacts: %v", err)
	}
	byPath := artifactsByPath(artifacts)
	if _, ok := byPath["modsecurity/easy/vp-dis.conf"]; !ok {
		t.Fatalf("expected modsec artifact with VirtualPatches even when SecurityMode=disabled")
	}
}
