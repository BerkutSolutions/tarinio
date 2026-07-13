package compiler

import (
	"strings"
	"testing"
)

func TestExplicitManagementConfiguredDisablesLegacyHeuristics(t *testing.T) {
	if isManagementSite(SiteInput{ID: "control-plane", ManagementConfigured: true}) {
		t.Fatal("legacy ID must not receive bypass after persisted settings migration")
	}
	if !isManagementSite(SiteInput{ID: "ordinary", Management: true, ManagementConfigured: true}) {
		t.Fatal("explicit management host must receive bypass")
	}
}

func TestExplicitManagementHostDoesNotGrantBypassToSameUpstream(t *testing.T) {
	management := SiteInput{ID: "panel", PrimaryHost: "panel.example", Management: true, ManagementConfigured: true}
	ordinary := SiteInput{ID: "app", PrimaryHost: "app.example", ManagementConfigured: true}
	if !isManagementSite(management) || isManagementSite(ordinary) {
		t.Fatal("management ownership must be explicit and independent of upstream")
	}
	if !strings.Contains(easyModSecurityBypassPathPatternForSite(management), "api") {
		t.Fatal("management host must receive management API safeguard")
	}
	if easyModSecurityBypassPathPatternForSite(ordinary) != "" {
		t.Fatal("ordinary site must not receive management safeguard")
	}
}

func TestExplicitManagementDNSAndIPHostsRenderOnlyOwnSafeguards(t *testing.T) {
	sites := []SiteInput{
		{ID: "panel-dns", Enabled: true, PrimaryHost: "panel.example", ListenHTTP: true, Management: true, ManagementConfigured: true},
		{ID: "panel-ip", Enabled: true, PrimaryHost: "192.0.2.44", ListenHTTP: true, Management: true, ManagementConfigured: true},
		{ID: "ordinary", Enabled: true, PrimaryHost: "app.example", ListenHTTP: true, ManagementConfigured: true},
	}
	profiles := make([]EasyProfileInput, 0, len(sites))
	for _, site := range sites {
		profiles = append(profiles, defaultEasyProfileForSite(site.ID))
	}
	artifacts, err := RenderEasyArtifacts(sites, profiles)
	if err != nil {
		t.Fatalf("render easy artifacts: %v", err)
	}
	byPath := artifactsByPath(artifacts)
	for _, id := range []string{"panel-dns", "panel-ip"} {
		content := string(byPath["nginx/easy/"+id+".conf"].Content)
		if strings.Contains(content, "modsecurity on;") || strings.Contains(content, "modsecurity_rules_file") {
			t.Fatalf("%s must not enable ModSecurity in the easy location", id)
		}
	}
	ordinary := string(byPath["nginx/easy/ordinary.conf"].Content)
	if !strings.Contains(ordinary, "modsecurity on;") || !strings.Contains(ordinary, "modsecurity_rules_file") {
		t.Fatal("ordinary site must retain its ModSecurity configuration")
	}
}

func TestSnapshotManagementHostsIgnoreLegacyEnvironment(t *testing.T) {
	t.Setenv("CONTROL_PLANE_DEV_FAST_START_MANAGEMENT_SITE_ID", "ordinary")
	management := SiteInput{ID: "panel", PrimaryHost: "panel.example", Management: true, ManagementConfigured: true}
	ordinary := SiteInput{ID: "ordinary", PrimaryHost: "ordinary.example", ManagementConfigured: true}
	if !isManagementSite(management) || isManagementSite(ordinary) {
		t.Fatal("configured snapshot must ignore legacy environment identifiers")
	}
}
