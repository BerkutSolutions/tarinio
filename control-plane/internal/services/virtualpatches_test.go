package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"waf/control-plane/internal/jobs"
	"waf/control-plane/internal/revisions"
	"waf/control-plane/internal/virtualpatches"
)

// mockVirtualPatchStore implements VirtualPatchStore for testing.
type mockVirtualPatchStore struct {
	patches []virtualpatches.VirtualPatch
}

func (m *mockVirtualPatchStore) Create(patch virtualpatches.VirtualPatch) (virtualpatches.VirtualPatch, error) {
	for _, existing := range m.patches {
		if existing.ID == patch.ID {
			return virtualpatches.VirtualPatch{}, fmt.Errorf("virtual patch %s already exists", patch.ID)
		}
	}
	m.patches = append(m.patches, patch)
	return patch, nil
}

func (m *mockVirtualPatchStore) List(siteID string) ([]virtualpatches.VirtualPatch, error) {
	out := make([]virtualpatches.VirtualPatch, 0)
	for _, p := range m.patches {
		if siteID == "" || p.SiteID == siteID {
			out = append(out, p)
		}
	}
	return out, nil
}

func (m *mockVirtualPatchStore) ListActive(siteID string) ([]virtualpatches.VirtualPatch, error) {
	all, _ := m.List(siteID)
	out := make([]virtualpatches.VirtualPatch, 0)
	for _, p := range all {
		if !p.IsExpired() {
			out = append(out, p)
		}
	}
	return out, nil
}

func (m *mockVirtualPatchStore) Delete(id string) error {
	for i, p := range m.patches {
		if p.ID == id {
			m.patches = append(m.patches[:i], m.patches[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("virtual patch %s not found", id)
}

type virtualPatchAutoCompile struct{ calls int }

func (f *virtualPatchAutoCompile) Create(context.Context) (CompileRequestResult, error) {
	f.calls++
	return CompileRequestResult{Revision: revisions.Revision{ID: "rev-virtual-patch"}}, nil
}

type virtualPatchAutoApply struct{ calls int }

func (f *virtualPatchAutoApply) Apply(context.Context, string) (jobs.Job, error) {
	f.calls++
	return jobs.Job{Status: jobs.StatusSucceeded}, nil
}

func TestVirtualPatchService_CreateTriggersCompileAndApply(t *testing.T) {
	compile := &virtualPatchAutoCompile{}
	apply := &virtualPatchAutoApply{}
	ConfigureAutoApply(compile, apply, NewNoopDistributedCoordinator())
	t.Cleanup(func() { ConfigureAutoApply(nil, nil, nil) })

	service := NewVirtualPatchService(&mockVirtualPatchStore{})
	_, err := service.Create(context.Background(), virtualpatches.VirtualPatch{SiteID: "site-auto", Pattern: "/protected", Target: "uri", Action: "block"})
	if err != nil {
		t.Fatalf("create virtual patch: %v", err)
	}
	if compile.calls != 1 || apply.calls != 1 {
		t.Fatalf("expected compile/apply once, got compile=%d apply=%d", compile.calls, apply.calls)
	}
}

func TestVirtualPatchService_CreateAndListActive(t *testing.T) {
	store := &mockVirtualPatchStore{}
	svc := NewVirtualPatchService(store)
	ctx := context.Background()

	patch := virtualpatches.VirtualPatch{
		SiteID:  "site-1",
		Pattern: "/test-vuln-path",
		Target:  "uri",
		Action:  "block",
	}
	created, err := svc.Create(ctx, patch)
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}
	if created.ID == "" {
		t.Fatal("Create: expected non-empty ID")
	}
	if created.SiteID != "site-1" {
		t.Errorf("Create: expected site_id site-1, got %s", created.SiteID)
	}

	active, err := svc.ListActive(ctx, "site-1")
	if err != nil {
		t.Fatalf("ListActive: unexpected error: %v", err)
	}
	if len(active) != 1 {
		t.Fatalf("ListActive: expected 1 patch, got %d", len(active))
	}
	if active[0].ID != created.ID {
		t.Errorf("ListActive: expected id %s, got %s", created.ID, active[0].ID)
	}
}

func TestVirtualPatchService_ExpiredNotInListActive(t *testing.T) {
	store := &mockVirtualPatchStore{}
	svc := NewVirtualPatchService(store)
	ctx := context.Background()

	// Patch that is already expired.
	expiredAt := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
	patch := virtualpatches.VirtualPatch{
		SiteID:    "site-2",
		Pattern:   "/old-vuln",
		Target:    "uri",
		Action:    "block",
		ExpiresAt: expiredAt,
	}
	_, err := svc.Create(ctx, patch)
	if err != nil {
		t.Fatalf("Create expired patch: unexpected error: %v", err)
	}

	active, err := svc.ListActive(ctx, "site-2")
	if err != nil {
		t.Fatalf("ListActive: unexpected error: %v", err)
	}
	if len(active) != 0 {
		t.Errorf("ListActive: expected 0 active patches after expiry, got %d", len(active))
	}

	// List (all) must still contain it.
	all, err := svc.List(ctx, "site-2")
	if err != nil {
		t.Fatalf("List: unexpected error: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("List: expected 1 patch (expired), got %d", len(all))
	}
}

func TestVirtualPatchService_Delete(t *testing.T) {
	store := &mockVirtualPatchStore{}
	svc := NewVirtualPatchService(store)
	ctx := context.Background()

	patch := virtualpatches.VirtualPatch{
		SiteID:  "site-3",
		Pattern: "/to-delete",
		Target:  "uri",
		Action:  "monitor",
	}
	created, err := svc.Create(ctx, patch)
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}

	if err := svc.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete: unexpected error: %v", err)
	}

	all, err := svc.List(ctx, "site-3")
	if err != nil {
		t.Fatalf("List after delete: unexpected error: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("List after delete: expected 0 patches, got %d", len(all))
	}
}

func TestVirtualPatchService_InvalidPattern(t *testing.T) {
	store := &mockVirtualPatchStore{}
	svc := NewVirtualPatchService(store)
	ctx := context.Background()

	patch := virtualpatches.VirtualPatch{
		SiteID:  "site-4",
		Pattern: "[invalid regex",
		Target:  "uri",
		Action:  "block",
	}
	_, err := svc.Create(ctx, patch)
	if err == nil {
		t.Fatal("Create with invalid regex: expected error, got nil")
	}
}

func TestVirtualPatchService_InvalidTarget(t *testing.T) {
	store := &mockVirtualPatchStore{}
	svc := NewVirtualPatchService(store)
	ctx := context.Background()

	patch := virtualpatches.VirtualPatch{
		SiteID:  "site-5",
		Pattern: "/test",
		Target:  "unknown",
		Action:  "block",
	}
	_, err := svc.Create(ctx, patch)
	if err == nil {
		t.Fatal("Create with invalid target: expected error, got nil")
	}
}
