package certificates

import "testing"

func TestStore_CreateListUpdateDelete(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}

	created, err := store.Create(Certificate{
		ID:         "cert-a",
		CommonName: "example.com",
		SANList:    []string{"www.example.com", "api.example.com"},
		NotBefore:  "2026-04-01T00:00:00Z",
		NotAfter:   "2026-10-01T00:00:00Z",
		Status:     "active",
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
	if len(items) != 1 || items[0].ID != "cert-a" {
		t.Fatalf("unexpected items: %+v", items)
	}

	updated, err := store.Update(Certificate{
		ID:         "cert-a",
		CommonName: "example.org",
		SANList:    []string{"api.example.org"},
		NotBefore:  "2026-04-01T00:00:00Z",
		NotAfter:   "2026-10-01T00:00:00Z",
		Status:     "expired",
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if updated.CommonName != "example.org" || updated.Status != "expired" {
		t.Fatalf("unexpected updated item: %+v", updated)
	}

	if err := store.Delete("cert-a"); err != nil {
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

	if _, err := store.Create(Certificate{ID: "cert-a", Status: "active"}); err == nil {
		t.Fatal("expected common name validation error")
	}
	if _, err := store.Create(Certificate{ID: "cert-a", CommonName: "example.com", Status: "pending"}); err == nil {
		t.Fatal("expected status validation error")
	}
	if _, err := store.Create(Certificate{ID: "cert-a", CommonName: "example.com", Status: "active", NotBefore: "bad"}); err == nil {
		t.Fatal("expected not_before validation error")
	}
}

func TestStore_NormalizesCertificate(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}

	created, err := store.Create(Certificate{
		ID:         " Cert-A ",
		CommonName: " example.com ",
		SANList:    []string{" www.example.com ", "api.example.com", "api.example.com"},
		Status:     " ACTIVE ",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if created.ID != "cert-a" || created.CommonName != "example.com" || created.Status != "active" {
		t.Fatalf("expected normalized certificate, got %+v", created)
	}
	if len(created.SANList) != 2 || created.SANList[0] != "api.example.com" || created.SANList[1] != "www.example.com" {
		t.Fatalf("expected normalized SANs, got %+v", created.SANList)
	}
}
