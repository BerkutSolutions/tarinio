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
	bundle := &pipeline.RevisionBundle{Files: []pipeline.BundleFile{{Path: "nginx/easy/panel.conf", Content: []byte("modsecurity on; modsecurity_rules_file /etc/waf/modsecurity/easy/panel.conf;")}}}
	if err := validateManagementSafeguard(bundle, snapshot); err == nil {
		t.Fatal("expected missing safeguard artifact to fail preflight")
	}
	bundle.Files = []pipeline.BundleFile{{Path: "nginx/easy/panel.conf", Content: []byte("modsecurity on; modsecurity_rules_file /etc/waf/modsecurity/easy/panel.conf; modsecurity_rules 'SecRule REQUEST_URI \"@rx ^/api/\" \"id:100001,phase:1,pass,nolog,ctl:ruleEngine=Off\"';")}}
	if err := validateManagementSafeguard(bundle, snapshot); err != nil {
		t.Fatalf("expected safeguard artifact to pass preflight: %v", err)
	}
}

func TestHasManagementModSecurityBypassRejectsIncompleteEasyRule(t *testing.T) {
	if hasManagementModSecurityBypass("modsecurity on; modsecurity_rules_file /etc/waf/modsecurity/easy/panel.conf;") {
		t.Fatal("ModSecurity without the scoped bypass rule must not pass safeguard validation")
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
