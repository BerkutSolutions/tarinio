package tlsconfigs

import "testing"

func TestStore_CreateListUpdateDelete(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}

	created, err := store.Create(TLSConfig{
		SiteID:        "site-a",
		CertificateID: "cert-a",
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
	if len(items) != 1 || items[0].SiteID != "site-a" {
		t.Fatalf("unexpected items: %+v", items)
	}

	updated, err := store.Update(TLSConfig{
		SiteID:        "site-a",
		CertificateID: "cert-b",
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if updated.CertificateID != "cert-b" {
		t.Fatalf("unexpected updated item: %+v", updated)
	}

	if err := store.Delete("site-a"); err != nil {
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

func TestStore_RejectsDuplicateSite(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}
	if _, err := store.Create(TLSConfig{SiteID: "site-a", CertificateID: "cert-a"}); err != nil {
		t.Fatalf("first create failed: %v", err)
	}
	if _, err := store.Create(TLSConfig{SiteID: "site-a", CertificateID: "cert-b"}); err == nil {
		t.Fatal("expected duplicate site tls config error")
	}
}

func TestStore_NormalizesIDs(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}

	created, err := store.Create(TLSConfig{
		SiteID:        " Site-A ",
		CertificateID: " Cert-A ",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if created.SiteID != "site-a" || created.CertificateID != "cert-a" {
		t.Fatalf("expected normalized ids, got %+v", created)
	}
}
