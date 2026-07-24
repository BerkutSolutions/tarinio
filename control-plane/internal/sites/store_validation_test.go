package sites

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestStoreRejectsDuplicatePrimaryHost(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "sites"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Create(Site{ID: "one", PrimaryHost: "App.Example.Test", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Create(Site{ID: "two", PrimaryHost: "app.example.test", Enabled: true}); err == nil || !strings.Contains(err.Error(), "primary_host") {
		t.Fatalf("duplicate create must fail with primary_host error, got %v", err)
	}
	if _, err := store.Create(Site{ID: "two", PrimaryHost: "two.example.test", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update(Site{ID: "two", PrimaryHost: "APP.EXAMPLE.TEST", Enabled: true}); err == nil || !strings.Contains(err.Error(), "primary_host") {
		t.Fatalf("duplicate update must fail with primary_host error, got %v", err)
	}
	items, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[1].PrimaryHost != "two.example.test" {
		t.Fatalf("failed validation changed persisted sites: %+v", items)
	}
}
