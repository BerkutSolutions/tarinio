package revisions

import (
	"path/filepath"
	"testing"
)

func TestStore_SavePendingPersistsRevisionMetadata(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}

	revision := Revision{
		ID:         "rev-001",
		Version:    1,
		CreatedAt:  "2026-03-31T12:00:00Z",
		Checksum:   "abc123",
		BundlePath: filepath.ToSlash(filepath.Join("bundles", "rev-001")),
	}
	if err := store.SavePending(revision); err != nil {
		t.Fatalf("save pending failed: %v", err)
	}

	reloaded, ok, err := store.Get("rev-001")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if !ok {
		t.Fatal("expected revision to exist")
	}
	if reloaded.Status != StatusPending {
		t.Fatalf("expected pending status, got %s", reloaded.Status)
	}
	if reloaded.Checksum != "abc123" || reloaded.BundlePath == "" {
		t.Fatal("expected metadata to be persisted")
	}
}

func TestStore_MarkActivePersistsCurrentActiveRevision(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}

	revision := Revision{
		ID:         "rev-001",
		Version:    1,
		CreatedAt:  "2026-03-31T12:00:00Z",
		Checksum:   "abc123",
		BundlePath: "bundles/rev-001",
	}
	if err := store.SavePending(revision); err != nil {
		t.Fatalf("save pending failed: %v", err)
	}
	if err := store.MarkActive("rev-001"); err != nil {
		t.Fatalf("mark active failed: %v", err)
	}

	active, ok, err := store.CurrentActive()
	if err != nil {
		t.Fatalf("current active failed: %v", err)
	}
	if !ok {
		t.Fatal("expected current active revision")
	}
	if active.ID != "rev-001" || active.Status != StatusActive {
		t.Fatalf("unexpected active revision: %+v", active)
	}
}

func TestStore_MarkActiveResetsPreviousActiveRevision(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}

	revisions := []Revision{
		{ID: "rev-001", Version: 1, CreatedAt: "2026-03-31T12:00:00Z", Checksum: "abc123", BundlePath: "bundles/rev-001"},
		{ID: "rev-002", Version: 2, CreatedAt: "2026-03-31T12:01:00Z", Checksum: "def456", BundlePath: "bundles/rev-002"},
	}
	for _, revision := range revisions {
		if err := store.SavePending(revision); err != nil {
			t.Fatalf("save pending failed: %v", err)
		}
	}
	if err := store.MarkActive("rev-001"); err != nil {
		t.Fatalf("mark first active failed: %v", err)
	}
	if err := store.MarkActive("rev-002"); err != nil {
		t.Fatalf("mark second active failed: %v", err)
	}

	first, ok, err := store.Get("rev-001")
	if err != nil {
		t.Fatalf("get first failed: %v", err)
	}
	if !ok {
		t.Fatal("expected first revision to exist")
	}
	if first.Status == StatusActive {
		t.Fatal("expected previous active revision to be reset")
	}

	second, ok, err := store.Get("rev-002")
	if err != nil {
		t.Fatalf("get second failed: %v", err)
	}
	if !ok {
		t.Fatal("expected second revision to exist")
	}
	if second.Status != StatusActive {
		t.Fatalf("expected second revision to be active, got %s", second.Status)
	}
}

func TestStore_MarkFailedUpdatesLifecycleState(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}

	revision := Revision{
		ID:         "rev-001",
		Version:    1,
		CreatedAt:  "2026-03-31T12:00:00Z",
		Checksum:   "abc123",
		BundlePath: "bundles/rev-001",
	}
	if err := store.SavePending(revision); err != nil {
		t.Fatalf("save pending failed: %v", err)
	}
	if err := store.MarkFailed("rev-001"); err != nil {
		t.Fatalf("mark failed failed: %v", err)
	}

	reloaded, ok, err := store.Get("rev-001")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if !ok {
		t.Fatal("expected revision to exist")
	}
	if reloaded.Status != StatusFailed {
		t.Fatalf("expected failed status, got %s", reloaded.Status)
	}
}

func TestStore_ListOrdersRevisionsByVersion(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}

	inputs := []Revision{
		{ID: "rev-002", Version: 2, CreatedAt: "2026-03-31T12:02:00Z", Checksum: "b", BundlePath: "bundles/rev-002"},
		{ID: "rev-001", Version: 1, CreatedAt: "2026-03-31T12:01:00Z", Checksum: "a", BundlePath: "bundles/rev-001"},
	}
	for _, revision := range inputs {
		if err := store.SavePending(revision); err != nil {
			t.Fatalf("save pending failed: %v", err)
		}
	}

	revisions, err := store.List()
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(revisions) != 2 {
		t.Fatalf("expected 2 revisions, got %d", len(revisions))
	}
	if revisions[0].ID != "rev-001" || revisions[1].ID != "rev-002" {
		t.Fatalf("unexpected order: %+v", revisions)
	}
}
