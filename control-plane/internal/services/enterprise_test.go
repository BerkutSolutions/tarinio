package services

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"path/filepath"
	"testing"
	"time"

	"waf/control-plane/internal/audits"
	"waf/control-plane/internal/enterprise"
	"waf/control-plane/internal/events"
	"waf/control-plane/internal/jobs"
	"waf/control-plane/internal/revisions"
)

func TestEnterpriseServiceBuildSupportBundle(t *testing.T) {
	root := t.TempDir()

	enterpriseStore, err := enterprise.NewStore(filepath.Join(root, "enterprise"))
	if err != nil {
		t.Fatalf("enterprise.NewStore() error = %v", err)
	}
	auditStore, err := audits.NewStore(filepath.Join(root, "audits"))
	if err != nil {
		t.Fatalf("audits.NewStore() error = %v", err)
	}
	eventStore, err := events.NewStore(filepath.Join(root, "events"))
	if err != nil {
		t.Fatalf("events.NewStore() error = %v", err)
	}
	jobStore, err := jobs.NewStore(filepath.Join(root, "jobs"))
	if err != nil {
		t.Fatalf("jobs.NewStore() error = %v", err)
	}
	revisionStore, err := revisions.NewStore(filepath.Join(root, "revisions"))
	if err != nil {
		t.Fatalf("revisions.NewStore() error = %v", err)
	}

	now := time.Now().UTC()
	if err := auditStore.Append(audits.AuditEvent{
		ID:           "audit-1",
		Action:       "revision.compile_request",
		ResourceType: "revision",
		ResourceID:   "rev-000001",
		Status:       audits.StatusSucceeded,
		OccurredAt:   now.Format(time.RFC3339Nano),
		Summary:      "revision compile request",
	}); err != nil {
		t.Fatalf("Append audit-1: %v", err)
	}
	if err := auditStore.Append(audits.AuditEvent{
		ID:                "audit-2",
		Action:            "revision.apply_trigger",
		ResourceType:      "revision",
		ResourceID:        "rev-000001",
		RelatedRevisionID: "rev-000001",
		RelatedJobID:      "apply-rev-000001",
		Status:            audits.StatusSucceeded,
		OccurredAt:        now.Add(time.Second).Format(time.RFC3339Nano),
		Summary:           "revision apply trigger",
	}); err != nil {
		t.Fatalf("Append audit-2: %v", err)
	}
	if _, err := eventStore.Create(events.Event{
		ID:                "event-1",
		Type:              events.TypeApplySucceeded,
		Severity:          events.SeverityInfo,
		SourceComponent:   "apply-runner",
		OccurredAt:        now.Add(2 * time.Second).Format(time.RFC3339Nano),
		Summary:           "apply succeeded",
		RelatedRevisionID: "rev-000001",
		RelatedJobID:      "apply-rev-000001",
	}); err != nil {
		t.Fatalf("Create event: %v", err)
	}
	createdJob, err := jobStore.Create(jobs.Job{
		ID:               "apply-rev-000001",
		Type:             jobs.TypeApply,
		TargetRevisionID: "rev-000001",
	})
	if err != nil {
		t.Fatalf("Create job: %v", err)
	}
	if _, err := jobStore.MarkSucceeded(createdJob.ID, "revision applied"); err != nil {
		t.Fatalf("MarkSucceeded job: %v", err)
	}
	if err := revisionStore.SavePending(revisions.Revision{
		ID:                "rev-000001",
		Version:           1,
		CreatedAt:         now.Format(time.RFC3339Nano),
		Checksum:          "abc123",
		BundlePath:        "snapshots/rev-000001",
		Status:            revisions.StatusPending,
		ApprovalStatus:    revisions.ApprovalApproved,
		RequiredApprovals: 1,
		Approvals: []revisions.ApprovalRecord{{
			UserID:     "admin",
			Username:   "admin",
			ApprovedAt: now.Add(500 * time.Millisecond).Format(time.RFC3339Nano),
		}},
		ApprovedAt:     now.Add(500 * time.Millisecond).Format(time.RFC3339Nano),
		SignatureKeyID: "evidence-test",
		Signature:      "deadbeef",
	}); err != nil {
		t.Fatalf("SavePending revision: %v", err)
	}

	service := NewEnterpriseService(enterpriseStore, nil, nil, nil, revisionStore, auditStore, eventStore, jobStore, nil)
	archive, filename, err := service.BuildSupportBundle()
	if err != nil {
		t.Fatalf("BuildSupportBundle() error = %v", err)
	}
	if filename == "" {
		t.Fatalf("BuildSupportBundle() returned empty filename")
	}

	files := untarGzipJSON(t, archive)
	manifestRaw := files["manifest.json"]
	if manifestRaw == nil {
		t.Fatalf("manifest.json missing from bundle")
	}
	manifest, ok := manifestRaw.(map[string]any)
	if !ok {
		t.Fatalf("manifest.json has unexpected payload type: %T", manifestRaw)
	}
	auditChain, ok := manifest["audit_chain"].(map[string]any)
	if !ok {
		t.Fatalf("manifest audit_chain missing: %#v", manifest["audit_chain"])
	}
	if valid, _ := auditChain["valid"].(bool); !valid {
		t.Fatalf("expected valid audit chain, got %#v", auditChain)
	}
	if got := int(manifest["signed_revision_items"].(float64)); got != 1 {
		t.Fatalf("signed_revision_items = %d, want 1", got)
	}
	fileDigests, ok := manifest["file_digests"].(map[string]any)
	if !ok || fileDigests["audits.json"] == nil {
		t.Fatalf("manifest file_digests missing audits.json: %#v", manifest["file_digests"])
	}
	if files["signature.json"] == nil {
		t.Fatalf("signature.json missing from bundle")
	}
}

func untarGzipJSON(t *testing.T, archive []byte) map[string]any {
	t.Helper()
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		t.Fatalf("gzip.NewReader(): %v", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	out := map[string]any{}
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return out
		}
		if err != nil {
			t.Fatalf("tar.Next(): %v", err)
		}
		if filepath.Ext(header.Name) != ".json" {
			continue
		}
		raw, err := io.ReadAll(tr)
		if err != nil {
			t.Fatalf("ReadAll(%s): %v", header.Name, err)
		}
		var decoded any
		if err := json.Unmarshal(raw, &decoded); err != nil {
			t.Fatalf("Unmarshal(%s): %v", header.Name, err)
		}
		out[header.Name] = decoded
	}
}
