package roles

import (
	"testing"

	"waf/control-plane/internal/rbac"
)

func TestNewStore_MigratesBuiltInRolesToCurrentPermissions(t *testing.T) {
	root := t.TempDir()

	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	admin, ok, err := store.Get("admin")
	if err != nil {
		t.Fatalf("get admin: %v", err)
	}
	if !ok {
		t.Fatal("expected seeded admin role")
	}
	admin.Name = "legacy admin"
	admin.Permissions = []rbac.Permission{rbac.PermissionDashboardRead}
	if _, err := store.Update(admin); err != nil {
		t.Fatalf("downgrade admin role: %v", err)
	}

	migratedStore, err := NewStore(root)
	if err != nil {
		t.Fatalf("re-open store: %v", err)
	}

	migratedAdmin, ok, err := migratedStore.Get("admin")
	if err != nil {
		t.Fatalf("get migrated admin: %v", err)
	}
	if !ok {
		t.Fatal("expected migrated admin role")
	}
	if migratedAdmin.Name != "Administrator" {
		t.Fatalf("expected admin name to be refreshed, got %q", migratedAdmin.Name)
	}
	got := rbac.SortedPermissions(migratedAdmin.Permissions)
	want := rbac.SortedPermissions(rbac.AllPermissions())
	if len(got) != len(want) {
		t.Fatalf("expected %d permissions, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected permission at %d: got %s want %s", i, got[i], want[i])
		}
	}
}
