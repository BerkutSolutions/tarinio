package services

import (
	"path/filepath"
	"testing"

	"waf/control-plane/internal/passkeys"
	"waf/control-plane/internal/rbac"
	"waf/control-plane/internal/roles"
	"waf/control-plane/internal/sessions"
	"waf/control-plane/internal/users"
)

func TestZeroTrustDefaultRolesPermissions(t *testing.T) {
	root := t.TempDir()
	roleStore, err := roles.NewStore(filepath.Join(root, "roles"))
	if err != nil {
		t.Fatalf("roles store: %v", err)
	}
	userStore, err := users.NewStore(filepath.Join(root, "users"), users.BootstrapUser{
		Enabled:  true,
		ID:       "admin",
		Username: "admin",
		Email:    "admin@example.test",
		Password: "admin",
		RoleIDs:  []string{"admin"},
	})
	if err != nil {
		t.Fatalf("users store: %v", err)
	}
	sessionStore, err := sessions.NewStore(filepath.Join(root, "sessions"))
	if err != nil {
		t.Fatalf("sessions store: %v", err)
	}
	passkeyStore, err := passkeys.NewStore(filepath.Join(root, "passkeys"))
	if err != nil {
		t.Fatalf("passkeys store: %v", err)
	}
	service := NewAuthService(userStore, roleStore, sessionStore, passkeyStore, "WAF", AuthSecurityConfig{Pepper: "test-pepper"}, nil)

	createTestUser(t, userStore, "auditor", []string{"auditor"})
	createTestUser(t, userStore, "manager", []string{"manager"})
	createTestUser(t, userStore, "soc", []string{"soc"})

	adminUser, err := service.userByID("admin")
	if err != nil {
		t.Fatalf("admin me: %v", err)
	}
	assertHasPermissions(t, adminUser.Permissions, []string{
		string(rbac.PermissionAdministrationUsersWrite),
		string(rbac.PermissionAdministrationRolesWrite),
		string(rbac.PermissionSettingsGeneralWrite),
		string(rbac.PermissionSettingsStorageWrite),
	})

	auditorUser, err := service.userByID("auditor")
	if err != nil {
		t.Fatalf("auditor me: %v", err)
	}
	assertHasPermissions(t, auditorUser.Permissions, []string{
		string(rbac.PermissionProfileRead),
		string(rbac.PermissionSettingsAboutRead),
		string(rbac.PermissionRequestsRead),
		string(rbac.PermissionEventsRead),
	})
	assertLacksPermissions(t, auditorUser.Permissions, []string{
		string(rbac.PermissionAdministrationUsersWrite),
		string(rbac.PermissionSettingsGeneralRead),
		string(rbac.PermissionSettingsStorageRead),
	})

	managerUser, err := service.userByID("manager")
	if err != nil {
		t.Fatalf("manager me: %v", err)
	}
	assertHasPermissions(t, managerUser.Permissions, []string{
		string(rbac.PermissionAdministrationUsersWrite),
		string(rbac.PermissionAdministrationRolesRead),
		string(rbac.PermissionSitesWrite),
	})
	assertLacksPermissions(t, managerUser.Permissions, []string{
		string(rbac.PermissionSettingsGeneralRead),
		string(rbac.PermissionSettingsStorageRead),
		string(rbac.PermissionAdministrationRolesWrite),
	})

	socUser, err := service.userByID("soc")
	if err != nil {
		t.Fatalf("soc me: %v", err)
	}
	assertHasPermissions(t, socUser.Permissions, []string{
		string(rbac.PermissionAntiDDoSWrite),
		string(rbac.PermissionOWASPCRSWrite),
		string(rbac.PermissionRequestsRead),
		string(rbac.PermissionEventsRead),
	})
	assertLacksPermissions(t, socUser.Permissions, []string{
		string(rbac.PermissionAdministrationUsersRead),
		string(rbac.PermissionSettingsGeneralRead),
	})
}

func createTestUser(t *testing.T, store *users.Store, username string, roleIDs []string) {
	t.Helper()
	hash, err := users.HashPassword("secret")
	if err != nil {
		t.Fatalf("hash password for %s: %v", username, err)
	}
	if _, err := store.Create(users.User{
		ID:           username,
		Username:     username,
		Email:        username + "@example.test",
		PasswordHash: hash,
		IsActive:     true,
		RoleIDs:      roleIDs,
	}); err != nil {
		t.Fatalf("create user %s: %v", username, err)
	}
}

func assertHasPermissions(t *testing.T, items []string, expected []string) {
	t.Helper()
	set := map[string]struct{}{}
	for _, item := range items {
		set[item] = struct{}{}
	}
	for _, item := range expected {
		if _, ok := set[item]; !ok {
			t.Fatalf("expected permission %s in %v", item, items)
		}
	}
}

func assertLacksPermissions(t *testing.T, items []string, expected []string) {
	t.Helper()
	set := map[string]struct{}{}
	for _, item := range items {
		set[item] = struct{}{}
	}
	for _, item := range expected {
		if _, ok := set[item]; ok {
			t.Fatalf("unexpected permission %s in %v", item, items)
		}
	}
}
