package users

import (
	"path/filepath"
	"testing"
)

func TestStoreDeletePersistsRemoval(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "users"), BootstrapUser{})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	hash, err := HashPassword("password-123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if _, err := store.Create(User{ID: "delete-me", Username: "delete-me", PasswordHash: hash, IsActive: true, RoleIDs: []string{"soc"}}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := store.Delete("delete-me"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, ok, err := store.Get("delete-me"); err != nil || ok {
		t.Fatalf("get after delete: ok=%v err=%v", ok, err)
	}
}
