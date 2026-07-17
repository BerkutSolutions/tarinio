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

func TestAuthServiceTOTPStepUpRequiresFreshCodeAndBindsItToSession(t *testing.T) {
	root := t.TempDir()
	roleStore, err := roles.NewStore(filepath.Join(root, "roles"))
	if err != nil {
		t.Fatal(err)
	}
	userStore, err := users.NewStore(filepath.Join(root, "users"), users.BootstrapUser{
		Enabled: true, ID: "admin", Username: "admin", Email: "admin@example.test", Password: "admin", RoleIDs: []string{"admin"},
	})
	if err != nil {
		t.Fatal(err)
	}
	sessionStore, err := sessions.NewStore(filepath.Join(root, "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	passkeyStore, err := passkeys.NewStore(filepath.Join(root, "passkeys"))
	if err != nil {
		t.Fatal(err)
	}
	service := NewAuthService(userStore, roleStore, sessionStore, passkeyStore, "WAF", AuthSecurityConfig{Pepper: "test-pepper"}, nil)

	first, err := service.Login(context.Background(), "admin", "admin")
	if err != nil {
		t.Fatal(err)
	}
	setup, err := service.SetupTOTP(context.Background(), first.Session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.EnableTOTP(context.Background(), first.Session.ID, setup.ChallengeID, auth.GenerateCodeForTest(setup.Secret, time.Now().UTC())); err != nil {
		t.Fatal(err)
	}
	challenge, err := service.Login(context.Background(), "admin", "admin")
	if err != nil || !challenge.RequiresTwoFactor {
		t.Fatalf("login 2fa challenge=%+v err=%v", challenge, err)
	}
	verified, err := service.Login2FA(context.Background(), challenge.ChallengeID, auth.GenerateCodeForTest(setup.Secret, time.Now().UTC()), "")
	if err != nil {
		t.Fatal(err)
	}
	if err := service.RequireStepUp(verified.ID); err == nil {
		t.Fatal("expected certificate-export boundary to require fresh step-up")
	}
	if _, err := service.StepUpTOTP(context.Background(), verified.ID, "000000"); err == nil {
		t.Fatal("expected an invalid TOTP code to fail")
	}
	result, err := service.StepUpTOTP(context.Background(), verified.ID, auth.GenerateCodeForTest(setup.Secret, time.Now().UTC()))
	if err != nil || !result.OK {
		t.Fatalf("step-up result=%+v err=%v", result, err)
	}
	if err := service.RequireStepUp(verified.ID); err != nil {
		t.Fatalf("expected fresh step-up to satisfy export boundary: %v", err)
	}
	other, err := service.Login2FA(context.Background(), mustNewLoginChallenge(t, service).ChallengeID, auth.GenerateCodeForTest(setup.Secret, time.Now().UTC()), "")
	if err != nil {
		t.Fatal(err)
	}
	if err := service.RequireStepUp(other.ID); err == nil {
		t.Fatal("step-up assertion must not transfer to another session")
	}
}

func mustNewLoginChallenge(t *testing.T, service *AuthService) LoginResult {
	t.Helper()
	result, err := service.Login(context.Background(), "admin", "admin")
	if err != nil {
		t.Fatal(err)
	}
	return result
}
