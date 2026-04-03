package sessions

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStore_TouchSessionExtendsExpiry(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	created, err := store.CreateSession("admin", "admin", []string{"admin"}, 2*time.Minute)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	createdExpiresAt, err := time.Parse(time.RFC3339, created.ExpiresAt)
	if err != nil {
		t.Fatalf("parse created expires_at: %v", err)
	}

	touched, ok, err := store.TouchSession(created.ID, 4*time.Minute)
	if err != nil {
		t.Fatalf("touch session: %v", err)
	}
	if !ok {
		t.Fatal("expected session to be touched")
	}
	touchedExpiresAt, err := time.Parse(time.RFC3339, touched.ExpiresAt)
	if err != nil {
		t.Fatalf("parse touched expires_at: %v", err)
	}
	if !touchedExpiresAt.After(createdExpiresAt) {
		t.Fatalf("expected touched expiry to be extended: before=%s after=%s", created.ExpiresAt, touched.ExpiresAt)
	}
}

func TestStore_TouchSessionDoesNotReviveExpiredSession(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	created, err := store.CreateSession("admin", "admin", []string{"admin"}, time.Millisecond)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	time.Sleep(15 * time.Millisecond)

	_, ok, err := store.TouchSession(created.ID, 5*time.Minute)
	if err != nil {
		t.Fatalf("touch session: %v", err)
	}
	if ok {
		t.Fatal("expected expired session to remain unavailable")
	}
}
