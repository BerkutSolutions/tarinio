package services

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"path/filepath"
	"testing"
	"time"

	"waf/control-plane/internal/auth"
	"waf/control-plane/internal/passkeys"
	"waf/control-plane/internal/roles"
	"waf/control-plane/internal/sessions"
	"waf/control-plane/internal/users"
)

func TestAuthService_LoginAndTOTPFlow(t *testing.T) {
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
	service := NewAuthService(userStore, roleStore, sessionStore, passkeyStore, "WAF", AuthSecurityConfig{
		Pepper: "test-pepper",
		WebAuthn: WebAuthnConfig{
			Enabled: true,
			RPName:  "TARINIO",
		},
	}, nil)

	login, err := service.Login(context.Background(), "admin", "admin")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if login.Session.ID == "" {
		t.Fatal("expected session")
	}

	setup, err := service.SetupTOTP(context.Background(), login.Session.ID)
	if err != nil {
		t.Fatalf("setup totp failed: %v", err)
	}
	code := auth.GenerateCodeForTest(setup.Secret, time.Now().UTC())
	enableResult, err := service.EnableTOTP(context.Background(), login.Session.ID, setup.ChallengeID, code)
	if err != nil {
		t.Fatalf("enable totp failed: %v", err)
	}
	if !enableResult.User.TOTPEnabled {
		t.Fatal("expected totp enabled")
	}

	login2, err := service.Login(context.Background(), "admin", "admin")
	if err != nil {
		t.Fatalf("login second phase failed: %v", err)
	}
	if !login2.RequiresTwoFactor || login2.ChallengeID == "" {
		t.Fatal("expected 2fa challenge")
	}
	session, err := service.Login2FA(context.Background(), login2.ChallengeID, auth.GenerateCodeForTest(setup.Secret, time.Now().UTC()), "")
	if err != nil {
		t.Fatalf("login 2fa failed: %v", err)
	}
	if session.ID == "" {
		t.Fatal("expected final session")
	}
}

func TestAuthService_LoginMigratesLegacyPasswordHash(t *testing.T) {
	root := t.TempDir()
	roleStore, err := roles.NewStore(filepath.Join(root, "roles"))
	if err != nil {
		t.Fatalf("roles store: %v", err)
	}
	userStore, err := users.NewStore(filepath.Join(root, "users"), users.BootstrapUser{})
	if err != nil {
		t.Fatalf("users store: %v", err)
	}
	legacyHash := legacyPasswordHashForTest(t, "secret")
	if _, err := userStore.Create(users.User{
		ID:           "legacy",
		Username:     "legacy",
		Email:        "legacy@example.test",
		PasswordHash: legacyHash,
		IsActive:     true,
		RoleIDs:      []string{"auditor"},
	}); err != nil {
		t.Fatalf("create legacy user: %v", err)
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

	if _, err := service.Login(context.Background(), "legacy", "secret"); err != nil {
		t.Fatalf("login failed: %v", err)
	}
	updated, ok, err := userStore.Get("legacy")
	if err != nil || !ok {
		t.Fatalf("get updated user: %v, ok=%v", err, ok)
	}
	if users.NeedsPasswordRehash(updated.PasswordHash) {
		t.Fatalf("expected legacy password hash to be migrated, got %q", updated.PasswordHash)
	}
}

func TestAuthService_AuthenticateUsesLiveUserRolesAndRejectsInactiveUsers(t *testing.T) {
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

	login, err := service.Login(context.Background(), "admin", "admin")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	user, ok, err := userStore.Get("admin")
	if err != nil || !ok {
		t.Fatalf("get user: %v, ok=%v", err, ok)
	}
	user.RoleIDs = []string{"auditor"}
	if _, err := userStore.Update(user); err != nil {
		t.Fatalf("update user roles: %v", err)
	}

	session, err := service.Authenticate(login.Session.ID)
	if err != nil {
		t.Fatalf("authenticate after role change: %v", err)
	}
	if len(session.RoleIDs) != 1 || session.RoleIDs[0] != "auditor" {
		t.Fatalf("expected live roles from user store, got %+v", session.RoleIDs)
	}

	user.IsActive = false
	if _, err := userStore.Update(user); err != nil {
		t.Fatalf("deactivate user: %v", err)
	}
	if _, err := service.Authenticate(login.Session.ID); err == nil {
		t.Fatal("expected inactive user session to be rejected")
	}
}

func TestAuthService_ChangePasswordRevokesOtherSessions(t *testing.T) {
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

	first, err := service.Login(context.Background(), "admin", "admin")
	if err != nil {
		t.Fatalf("first login: %v", err)
	}
	second, err := service.Login(context.Background(), "admin", "admin")
	if err != nil {
		t.Fatalf("second login: %v", err)
	}
	if err := service.ChangePassword(context.Background(), first.Session.ID, "admin", "new-secret"); err != nil {
		t.Fatalf("change password: %v", err)
	}
	if _, err := service.Authenticate(second.Session.ID); err == nil {
		t.Fatal("expected second session to be revoked")
	}
	if _, err := service.Authenticate(first.Session.ID); err != nil {
		t.Fatalf("expected current session to remain valid, got %v", err)
	}
}

func legacyPasswordHashForTest(t *testing.T, password string) string {
	t.Helper()
	salt := []byte("0123456789abcdef")
	sum := sha256.Sum256(append(append([]byte(nil), []byte(password)...), salt...))
	for i := 0; i < 120000; i++ {
		combined := append(append([]byte(nil), sum[:]...), salt...)
		sum = sha256.Sum256(combined)
	}
	return base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(sum[:])
}
