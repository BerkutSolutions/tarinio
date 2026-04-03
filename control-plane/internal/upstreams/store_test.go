package upstreams

import "testing"

func TestStore_CreateListUpdateDelete(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}

	created, err := store.Create(Upstream{
		ID:     "up-a",
		SiteID: "site-a",
		Host:   "app.internal",
		Port:   8080,
		Scheme: "http",
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
	if len(items) != 1 || items[0].ID != "up-a" {
		t.Fatalf("unexpected items: %+v", items)
	}

	updated, err := store.Update(Upstream{
		ID:     "up-a",
		SiteID: "site-a",
		Host:   "app2.internal",
		Port:   8443,
		Scheme: "https",
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if updated.Host != "app2.internal" || updated.Port != 8443 || updated.Scheme != "https" {
		t.Fatalf("unexpected updated item: %+v", updated)
	}

	if err := store.Delete("up-a"); err != nil {
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

func TestStore_RejectsInvalidFields(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}

	if _, err := store.Create(Upstream{ID: "up-a", SiteID: "site-a", Host: "", Port: 8080, Scheme: "http"}); err == nil {
		t.Fatal("expected host validation error")
	}
	if _, err := store.Create(Upstream{ID: "up-a", SiteID: "site-a", Host: "app", Port: 0, Scheme: "http"}); err == nil {
		t.Fatal("expected port validation error")
	}
	if _, err := store.Create(Upstream{ID: "up-a", SiteID: "site-a", Host: "app", Port: 8080, Scheme: "tcp"}); err == nil {
		t.Fatal("expected scheme validation error")
	}
}

func TestStore_NormalizesIDsAndScheme(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}

	created, err := store.Create(Upstream{
		ID:     " Up-A ",
		SiteID: " Site-A ",
		Host:   " app.internal ",
		Port:   8080,
		Scheme: " HTTPS ",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if created.ID != "up-a" || created.SiteID != "site-a" || created.Host != "app.internal" || created.Scheme != "https" {
		t.Fatalf("expected normalized upstream, got %+v", created)
	}
}
