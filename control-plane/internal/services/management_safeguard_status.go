package services

import (
	"os"
	"path/filepath"
	"strings"

	"waf/compiler/pipeline"
	"waf/control-plane/internal/revisions"
	"waf/control-plane/internal/revisionsnapshots"
)

type ManagementSafeguardStatus struct {
	ActiveRevisionID string   `json:"active_revision_id,omitempty"`
	ExpectedHosts    []string `json:"expected_hosts"`
	SafeguardPresent bool     `json:"safeguard_present"`
	Drift            bool     `json:"drift"`
}

type ManagementSafeguardStatusService struct {
	runtimeRoot string
	revisions   interface {
		CurrentActive() (revisions.Revision, bool, error)
	}
	snapshots interface {
		Load(string) (revisionsnapshots.Snapshot, error)
	}
}

func NewManagementSafeguardStatusService(root string, revisions interface {
	CurrentActive() (revisions.Revision, bool, error)
}, snapshots interface {
	Load(string) (revisionsnapshots.Snapshot, error)
}) *ManagementSafeguardStatusService {
	return &ManagementSafeguardStatusService{runtimeRoot: root, revisions: revisions, snapshots: snapshots}
}

func (s *ManagementSafeguardStatusService) Status() (ManagementSafeguardStatus, error) {
	active, ok, err := s.revisions.CurrentActive()
	if err != nil || !ok {
		return ManagementSafeguardStatus{}, err
	}
	snapshot, err := s.snapshots.Load(active.BundlePath)
	if err != nil {
		return ManagementSafeguardStatus{}, err
	}
	result := ManagementSafeguardStatus{ActiveRevisionID: active.ID, ExpectedHosts: append([]string(nil), snapshot.ManagementHosts...)}
	if !snapshot.ManagementHostsConfigured || len(snapshot.ManagementHosts) == 0 {
		return result, nil
	}
	pointer, err := pipeline.LoadActivePointer(s.runtimeRoot)
	if err != nil {
		result.Drift = true
		return result, nil
	}
	if pointer.RevisionID != active.ID {
		result.Drift = true
		return result, nil
	}
	for _, site := range snapshot.Sites {
		if !site.Enabled || !containsManagementHost(snapshot.ManagementHosts, site.PrimaryHost) {
			continue
		}
		content, readErr := os.ReadFile(filepath.Join(pointer.CandidatePath, "nginx", "sites", site.ID+".conf"))
		if readErr != nil || !hasManagementModSecurityBypass(string(content)) {
			result.Drift = true
			return result, nil
		}
	}
	result.SafeguardPresent = true
	return result, nil
}

func containsManagementHost(hosts []string, host string) bool {
	for _, item := range hosts {
		if strings.EqualFold(strings.TrimSpace(item), strings.TrimSpace(host)) {
			return true
		}
	}
	return false
}
