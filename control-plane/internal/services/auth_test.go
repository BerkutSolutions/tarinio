package services

import (
	"context"
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
