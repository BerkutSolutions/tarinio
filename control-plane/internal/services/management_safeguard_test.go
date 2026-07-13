package services

import (
	"testing"

	"waf/compiler/pipeline"
	"waf/control-plane/internal/revisionsnapshots"
	"waf/control-plane/internal/sites"
)

func TestValidateManagementSafeguardRejectsMissingArtifact(t *testing.T) {
	snapshot := revisionsnapshots.Snapshot{
		ManagementHostsConfigured: true,
		ManagementHosts:           []string{"panel.example"},
		Sites:                     []sites.Site{{ID: "panel", PrimaryHost: "panel.example", Enabled: true}},
	}
	bundle := &pipeline.RevisionBundle{Files: []pipeline.BundleFile{{Path: "modsecurity/easy/panel.conf", Content: []byte("SecRuleEngine On")}}}
	if err := validateManagementSafeguard(bundle, snapshot); err == nil {
		t.Fatal("expected missing safeguard artifact to fail preflight")
	}
	bundle.Files = []pipeline.BundleFile{{Path: "nginx/sites/panel.conf", Content: []byte("location ^~ /api/ { modsecurity off; proxy_pass http://control-plane:8080; }")}}
	if err := validateManagementSafeguard(bundle, snapshot); err != nil {
		t.Fatalf("expected safeguard artifact to pass preflight: %v", err)
	}
}

func TestHasManagementModSecurityBypassRejectsIncompleteAPILocation(t *testing.T) {
	if hasManagementModSecurityBypass("location ^~ /api/ { proxy_pass http://control-plane:8080; }") {
		t.Fatal("API location without modsecurity off must not pass safeguard validation")
	}
}

func TestValidateManagementSafeguardAllowsManagementProfileWithModSecurityDisabled(t *testing.T) {
	snapshot := revisionsnapshots.Snapshot{
		ManagementHostsConfigured: true,
		ManagementHosts:           []string{"panel.example"},
		Sites:                     []sites.Site{{ID: "panel", PrimaryHost: "panel.example", Enabled: true}},
	}
	if err := validateManagementSafeguard(&pipeline.RevisionBundle{}, snapshot); err != nil {
		t.Fatalf("transparent management profile must be applicable: %v", err)
	}
}
