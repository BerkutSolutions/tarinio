package antiddossuggestions

import (
	"path/filepath"
	"testing"
)

func TestStore_UpsertListSetStatus(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "antiddos-suggestions"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	created, err := store.Upsert(Suggestion{
		PathPrefix: "/.env",
		Hits:       20,
		UniqueIPs:  5,
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if created.Status != StatusSuggested {
		t.Fatalf("expected default status suggested, got %q", created.Status)
	}

	updated, err := store.SetStatus(created.ID, StatusShadow)
	if err != nil {
		t.Fatalf("set status: %v", err)
	}
	if updated.Status != StatusShadow {
		t.Fatalf("expected shadow status, got %q", updated.Status)
	}

	items, err := store.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one suggestion, got %d", len(items))
	}
	if items[0].ID != created.ID {
		t.Fatalf("unexpected suggestion id: %q", items[0].ID)
	}
}
