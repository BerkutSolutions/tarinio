package compiler

import (
	"strings"
	"testing"
)

func TestRenderWAFArtifacts_Deterministic(t *testing.T) {
	sites := []SiteInput{
		{ID: "site-b", Enabled: true, PrimaryHost: "b.example.com", ListenHTTP: true, DefaultUpstreamID: "up-b"},
		{ID: "site-a", Enabled: true, PrimaryHost: "a.example.com", ListenHTTP: true, DefaultUpstreamID: "up-a"},
	}

	policies := []WAFPolicyInput{
		{ID: "waf-b", SiteID: "site-b", Enabled: true, Mode: WAFModeDetection, CRSEnabled: true, CustomRuleIncludes: []string{"/rules/b.conf"}},
		{ID: "waf-a", SiteID: "site-a", Enabled: true, Mode: WAFModePrevention, CRSEnabled: true, CustomRuleIncludes: []string{"/rules/a.conf", "/rules/a.conf"}},
	}

	first, err := RenderWAFArtifacts(sites, policies)
	if err != nil {
		t.Fatalf("first render failed: %v", err)
	}
	second, err := RenderWAFArtifacts(sites, policies)
	if err != nil {
		t.Fatalf("second render failed: %v", err)
	}

	if len(first) != len(second) {
		t.Fatalf("artifact counts differ: %d vs %d", len(first), len(second))
	}

	for i := range first {
		if first[i].Path != second[i].Path {
			t.Fatalf("artifact path differs at %d: %s vs %s", i, first[i].Path, second[i].Path)
		}
		if first[i].Checksum != second[i].Checksum {
			t.Fatalf("artifact checksum differs for %s", first[i].Path)
		}
		if strings.TrimSpace(string(first[i].Content)) == "" {
			t.Fatalf("empty content for %s", first[i].Path)
		}
	}
}

func TestRenderWAFArtifacts_ManifestCompatibleKinds(t *testing.T) {
	artifacts, err := RenderWAFArtifacts(
		[]SiteInput{
			{ID: "site-a", Enabled: true, PrimaryHost: "a.example.com", ListenHTTP: true, DefaultUpstreamID: "up-a"},
		},
		[]WAFPolicyInput{
			{ID: "waf-a", SiteID: "site-a", Enabled: true, Mode: WAFModePrevention, CRSEnabled: true},
		},
	)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	if len(artifacts) != 4 {
		t.Fatalf("expected 4 WAF artifacts, got %d", len(artifacts))
	}

	for _, artifact := range artifacts {
		switch artifact.Kind {
		case ArtifactKindModSecurity, ArtifactKindCRSConfig:
		default:
			t.Fatalf("unexpected artifact kind for %s: %s", artifact.Path, artifact.Kind)
		}
		if artifact.Checksum == "" {
			t.Fatalf("missing checksum for %s", artifact.Path)
		}
	}
}

func TestRenderWAFArtifacts_RejectsInvalidMode(t *testing.T) {
	_, err := RenderWAFArtifacts(
		[]SiteInput{
			{ID: "site-a", Enabled: true, PrimaryHost: "a.example.com", ListenHTTP: true, DefaultUpstreamID: "up-a"},
		},
		[]WAFPolicyInput{
			{ID: "waf-a", SiteID: "site-a", Enabled: true, Mode: "invalid"},
		},
	)
	if err == nil {
		t.Fatal("expected error for invalid WAF mode")
	}
}

func TestRenderWAFArtifacts_FileBasedSiteConfig(t *testing.T) {
	artifacts, err := RenderWAFArtifacts(
		[]SiteInput{
			{ID: "site-a", Enabled: true, PrimaryHost: "a.example.com", ListenHTTP: true, DefaultUpstreamID: "up-a"},
		},
		[]WAFPolicyInput{
			{ID: "waf-a", SiteID: "site-a", Enabled: true, Mode: WAFModePrevention, CRSEnabled: true, CustomRuleIncludes: []string{"/rules/custom.conf"}},
		},
	)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	var siteArtifact ArtifactOutput
	for _, artifact := range artifacts {
		if artifact.Path == "modsecurity/sites/site-a.conf" {
			siteArtifact = artifact
			break
		}
	}

	content := string(siteArtifact.Content)
	if !strings.Contains(content, "SecRuleEngine On") {
		t.Fatal("expected file-based site config to set SecRuleEngine")
	}
	if !strings.Contains(content, "Include /etc/waf/modsecurity/crs-overrides/site-a.conf") {
		t.Fatal("expected file-based site config to include per-site overrides")
	}
	if strings.Contains(content, "modsecurity_rules '") {
		t.Fatal("did not expect inline nginx modsecurity_rules in site config")
	}
}
