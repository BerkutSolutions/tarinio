package accesspolicies

import "testing"

func TestStore_CreateListUpdateDelete(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}

	created, err := store.Create(AccessPolicy{
		ID:        "access-a",
		SiteID:    "site-a",
		Enabled:   true,
		AllowList: []string{"10.0.0.1", "10.0.0.0/24"},
		DenyList:  []string{"192.168.1.10"},
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if created.CreatedAt == "" || created.UpdatedAt == "" {
		t.Fatal("expected timestamps")
	}

	items, err := store.List()
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(items) != 1 || items[0].ID != "access-a" {
		t.Fatalf("unexpected items: %+v", items)
	}

	updated, err := store.Update(AccessPolicy{
		ID:        "access-a",
		SiteID:    "site-a",
		Enabled:   false,
		AllowList: []string{"10.0.0.0/24"},
		DenyList:  []string{"192.168.1.0/24"},
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if updated.Enabled || len(updated.DenyList) != 1 || updated.DenyList[0] != "192.168.1.0/24" {
		t.Fatalf("unexpected updated item: %+v", updated)
	}

	if err := store.Delete("access-a"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	items, err = store.List()
	if err != nil {
		t.Fatalf("list after delete failed: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty list, got %+v", items)
	}
}

func TestStore_RejectsInvalidCIDR(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}

	if _, err := store.Create(AccessPolicy{
		ID:        "access-a",
		SiteID:    "site-a",
		AllowList: []string{"bad-value"},
	}); err == nil {
		t.Fatal("expected invalid allowlist error")
	}
}

func TestStore_RejectsSecondPolicyForSameSite(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}
	if _, err := store.Create(AccessPolicy{ID: "access-a", SiteID: "site-a"}); err != nil {
		t.Fatalf("first create failed: %v", err)
	}
	if _, err := store.Create(AccessPolicy{ID: "access-b", SiteID: "site-a"}); err == nil {
		t.Fatal("expected duplicate site access policy error")
	}
}

func TestStore_NormalizesAndSortsLists(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}

	created, err := store.Create(AccessPolicy{
		ID:        " Access-A ",
		SiteID:    " Site-A ",
		AllowList: []string{" 10.0.0.2 ", "10.0.0.1", "10.0.0.1"},
		DenyList:  []string{" 192.168.1.2 ", "192.168.1.1"},
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if created.ID != "access-a" || created.SiteID != "site-a" {
		t.Fatalf("expected normalized ids, got %+v", created)
	}
	if created.AllowList[0] != "10.0.0.1" || created.AllowList[1] != "10.0.0.2" {
		t.Fatalf("expected sorted allowlist, got %+v", created.AllowList)
	}
}
