package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"waf/control-plane/internal/revisions"
	"waf/control-plane/internal/revisionsnapshots"
	"waf/control-plane/internal/sites"
)

type managementStatusRevisions struct{ item revisions.Revision }

func (s managementStatusRevisions) CurrentActive() (revisions.Revision, bool, error) {
	return s.item, true, nil
}

type managementStatusSnapshots struct{ item revisionsnapshots.Snapshot }

func (s managementStatusSnapshots) Load(string) (revisionsnapshots.Snapshot, error) {
	return s.item, nil
}

func TestManagementSafeguardStatusReportsArtifactDrift(t *testing.T) {
	root := t.TempDir()
	candidate := filepath.Join(root, "candidate")
	if err := os.MkdirAll(filepath.Join(candidate, "nginx", "sites"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "active"), 0o755); err != nil {
		t.Fatal(err)
	}
	pointer, err := json.Marshal(map[string]string{"revision_id": "rev-1", "candidate_path": candidate})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "active", "current.json"), pointer, 0o644); err != nil {
		t.Fatal(err)
	}
	snapshot := revisionsnapshots.Snapshot{ManagementHostsConfigured: true, ManagementHosts: []string{"panel.example"}, Sites: []sites.Site{{ID: "panel", PrimaryHost: "panel.example", Enabled: true}}}
	service := NewManagementSafeguardStatusService(root, managementStatusRevisions{revisions.Revision{ID: "rev-1", BundlePath: "snapshot"}}, managementStatusSnapshots{snapshot})
	status, err := service.Status()
	if err != nil {
		t.Fatal(err)
	}
	if !status.Drift {
		t.Fatal("expected missing artifact drift")
	}
	if err := os.WriteFile(filepath.Join(candidate, "nginx", "sites", "panel.conf"), []byte("location ^~ /api/ { modsecurity off; proxy_pass http://control-plane:8080; }"), 0o644); err != nil {
		t.Fatal(err)
	}
	status, err = service.Status()
	if err != nil {
		t.Fatal(err)
	}
	if status.Drift || !status.SafeguardPresent {
		t.Fatalf("unexpected status: %+v", status)
	}
}
