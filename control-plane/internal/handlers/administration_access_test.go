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
