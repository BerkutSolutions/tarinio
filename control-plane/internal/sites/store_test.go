package sites

import "testing"

func TestStore_CreateListUpdateDelete(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}

	created, err := store.Create(Site{
		ID:          "site-a",
		PrimaryHost: "a.example.com",
		Enabled:     true,
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if created.CreatedAt == "" || created.UpdatedAt == "" {
		t.Fatal("expected timestamps to be populated")
	}

	items, err := store.List()
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(items) != 1 || items[0].ID != "site-a" {
		t.Fatalf("unexpected sites: %+v", items)
	}

	updated, err := store.Update(Site{
		ID:          "site-a",
		PrimaryHost: "b.example.com",
		Enabled:     false,
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if updated.PrimaryHost != "b.example.com" || updated.Enabled {
		t.Fatalf("unexpected updated site: %+v", updated)
	}

	if err := store.Delete("site-a"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	items, err = store.List()
	if err != nil {
		t.Fatalf("list after delete failed: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty list after delete, got %+v", items)
	}
}

func TestStore_RejectsMissingPrimaryHost(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}
	if _, err := store.Create(Site{ID: "site-a"}); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestStore_NormalizesSiteID(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}

	created, err := store.Create(Site{
		ID:          " Site-A ",
		PrimaryHost: "a.example.com",
		Enabled:     true,
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if created.ID != "site-a" {
		t.Fatalf("expected normalized id, got %s", created.ID)
	}
}
