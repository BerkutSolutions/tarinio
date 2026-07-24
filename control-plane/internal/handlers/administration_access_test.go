package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"waf/control-plane/internal/roles"
	"waf/control-plane/internal/users"
)

type recordingAdministrationSessions struct {
	deletedUserID string
}

func (s *recordingAdministrationSessions) DeleteSessionsByUser(userID string) error {
	s.deletedUserID = userID
	return nil
}

func TestZeroTrustHealthHandler_ReturnsOKForSeededStores(t *testing.T) {
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

	handler := NewZeroTrustHealthHandler(userStore, roleStore)
	req := httptest.NewRequest(http.MethodGet, "/api/administration/zero-trust/health", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rr.Code, rr.Body.String())
	}
	if body := rr.Body.String(); body == "" {
		t.Fatal("expected response body")
	}
}

func TestAdministrationUsersHandler_CreatePersistsProfileFields(t *testing.T) {
	root := t.TempDir()
	roleStore, err := roles.NewStore(filepath.Join(root, "roles"))
	if err != nil {
		t.Fatalf("roles store: %v", err)
	}
	userStore, err := users.NewStore(filepath.Join(root, "users"), users.BootstrapUser{})
	if err != nil {
		t.Fatalf("users store: %v", err)
	}

	handler := NewAdministrationUsersHandler(userStore, roleStore)
	body := bytes.NewBufferString(`{
		"username":"analyst",
		"email":"analyst@example.test",
		"password":"password-123",
		"department":"SOC",
		"position":"Tier 1 Analyst",
		"role_ids":["soc"]
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/administration/users", body)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("unexpected status: %d body=%s", rr.Code, rr.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got := payload["department"]; got != "SOC" {
		t.Fatalf("expected department to persist, got %#v", got)
	}
	if got := payload["position"]; got != "Tier 1 Analyst" {
		t.Fatalf("expected position to persist, got %#v", got)
	}
}

func TestAdministrationHandlers_DeleteUserAndRole(t *testing.T) {
	root := t.TempDir()
	roleStore, err := roles.NewStore(filepath.Join(root, "roles"))
	if err != nil {
		t.Fatalf("roles store: %v", err)
	}
	userStore, err := users.NewStore(filepath.Join(root, "users"), users.BootstrapUser{Enabled: true, ID: "admin", Username: "admin", Password: "admin-password", RoleIDs: []string{"admin"}})
	if err != nil {
		t.Fatalf("users store: %v", err)
	}
	if _, err := roleStore.Create(roles.Role{ID: "temporary", Name: "Temporary", Permissions: nil}); err != nil {
		t.Fatalf("create role: %v", err)
	}
	passwordHash, err := users.HashPassword("password-123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if _, err := userStore.Create(users.User{ID: "temporary", Username: "temporary", PasswordHash: passwordHash, IsActive: true, RoleIDs: []string{"temporary"}}); err != nil {
		t.Fatalf("create user: %v", err)
	}

	sessions := &recordingAdministrationSessions{}
	usersHandler := NewAdministrationUsersHandlerWithSessions(userStore, roleStore, sessions)
	rolesHandler := NewAdministrationRolesHandler(roleStore, userStore)
	assignedRole := httptest.NewRecorder()
	rolesHandler.ServeHTTP(assignedRole, httptest.NewRequest(http.MethodDelete, "/api/administration/roles/temporary", nil))
	if assignedRole.Code != http.StatusConflict {
		t.Fatalf("assigned role delete status=%d body=%s", assignedRole.Code, assignedRole.Body.String())
	}

	deletedUser := httptest.NewRecorder()
	usersHandler.ServeHTTP(deletedUser, httptest.NewRequest(http.MethodDelete, "/api/administration/users/temporary", nil))
	if deletedUser.Code != http.StatusOK {
		t.Fatalf("user delete status=%d body=%s", deletedUser.Code, deletedUser.Body.String())
	}
	if _, ok, _ := userStore.Get("temporary"); ok {
		t.Fatal("deleted user remains in store")
	}
	if sessions.deletedUserID != "temporary" {
		t.Fatalf("deleted user sessions were not revoked, got %q", sessions.deletedUserID)
	}

	deletedRole := httptest.NewRecorder()
	rolesHandler.ServeHTTP(deletedRole, httptest.NewRequest(http.MethodDelete, "/api/administration/roles/temporary", nil))
	if deletedRole.Code != http.StatusOK {
		t.Fatalf("role delete status=%d body=%s", deletedRole.Code, deletedRole.Body.String())
	}
	if _, ok, _ := roleStore.Get("temporary"); ok {
		t.Fatal("deleted role remains in store")
	}

	builtin := httptest.NewRecorder()
	usersHandler.ServeHTTP(builtin, httptest.NewRequest(http.MethodDelete, "/api/administration/users/admin", nil))
	if builtin.Code != http.StatusConflict {
		t.Fatalf("built-in delete status=%d body=%s", builtin.Code, builtin.Body.String())
	}
	builtinRole := httptest.NewRecorder()
	rolesHandler.ServeHTTP(builtinRole, httptest.NewRequest(http.MethodDelete, "/api/administration/roles/SOC", nil))
	if builtinRole.Code != http.StatusConflict {
		t.Fatalf("normalized built-in role delete status=%d body=%s", builtinRole.Code, builtinRole.Body.String())
	}
}
