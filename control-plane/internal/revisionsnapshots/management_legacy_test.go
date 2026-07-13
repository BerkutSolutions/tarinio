package revisionsnapshots

import "testing"

func TestNormalizeLegacyManagementSnapshotNeverPromotesAlias(t *testing.T) {
	snapshot := Snapshot{ManagementHosts: []string{"panel.example"}, LegacyManagementSiteID: "control-plane-access"}
	normalizeLegacyManagementSnapshot(&snapshot)
	if len(snapshot.ManagementHosts) != 0 || snapshot.LegacyManagementSiteID != "" {
		t.Fatalf("legacy fields escaped guard: %+v", snapshot)
	}
	snapshot = Snapshot{ManagementHostsConfigured: true, ManagementHosts: []string{"panel.example"}, LegacyManagementSiteID: "old"}
	normalizeLegacyManagementSnapshot(&snapshot)
	if len(snapshot.ManagementHosts) != 1 || snapshot.LegacyManagementSiteID != "" {
		t.Fatalf("persisted hosts were changed: %+v", snapshot)
	}
}
