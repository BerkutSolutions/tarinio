package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"waf/control-plane/internal/services"
)

type fakeAuthService struct{}

func (f *fakeAuthService) Bootstrap(ctx context.Context, username, email, password string) (services.SessionResult, error) {
	return services.SessionResult{
		ID: "session-bootstrap",
		User: services.AuthUser{
			ID:       "admin",
			Username: username,
			Email:    email,
		},
	}, nil
}

func (f *fakeAuthService) Login(ctx context.Context, username, password string) (services.LoginResult, error) {
	return services.LoginResult{
		Session: services.SessionResult{
			ID: "session-1",
			User: services.AuthUser{
				ID:       "admin",
				Username: "admin",
			},
		},
	}, nil
}

func (f *fakeAuthService) Login2FA(ctx context.Context, challengeID, code, recoveryCode string) (services.SessionResult, error) {
	return services.SessionResult{ID: "session-2"}, nil
}

func (f *fakeAuthService) BeginPasskeyLogin(ctx context.Context, username string, req *http.Request) (services.PasskeyBeginResult, error) {
	return services.PasskeyBeginResult{ChallengeID: "pk-login", Options: map[string]any{"challenge": "abc"}}, nil
}
func (f *fakeAuthService) FinishPasskeyLogin(ctx context.Context, challengeID string, credentialJSON json.RawMessage, req *http.Request) (services.LoginResult, error) {
	return services.LoginResult{Session: services.SessionResult{ID: "session-passkey"}}, nil
}
func (f *fakeAuthService) BeginPasskey2FA(ctx context.Context, loginChallengeID string, req *http.Request) (services.Passkey2FABeginResult, error) {
	return services.Passkey2FABeginResult{WebAuthnChallengeID: "wch", Options: map[string]any{"challenge": "abc"}}, nil
}
func (f *fakeAuthService) FinishPasskey2FA(ctx context.Context, loginChallengeID, webAuthnChallengeID string, credentialJSON json.RawMessage, req *http.Request) (services.SessionResult, error) {
	return services.SessionResult{ID: "session-passkey-2fa"}, nil
}

func (f *fakeAuthService) Logout(ctx context.Context, sessionID string) error { return nil }
func (f *fakeAuthService) Me(sessionID string) (services.AuthUser, error) {
	return services.AuthUser{ID: "admin", Username: "admin", TOTPEnabled: true}, nil
}
func (f *fakeAuthService) SetupTOTP(ctx context.Context, sessionID string) (services.TOTPSetupResult, error) {
	return services.TOTPSetupResult{ChallengeID: "challenge-1", Secret: "SECRET"}, nil
}
func (f *fakeAuthService) EnableTOTP(ctx context.Context, sessionID, challengeID, code string) (services.TOTPEnableResult, error) {
	return services.TOTPEnableResult{OK: true, User: services.AuthUser{ID: "admin", Username: "admin", TOTPEnabled: true}}, nil
}
func (f *fakeAuthService) DisableTOTP(ctx context.Context, sessionID, password, recoveryCode string) (services.AuthUser, error) {
	return services.AuthUser{ID: "admin", Username: "admin", TOTPEnabled: false}, nil
}
func (f *fakeAuthService) ChangePassword(ctx context.Context, sessionID, currentPassword, password string) error {
	return nil
}
func (f *fakeAuthService) ListPasskeys(sessionID string) (services.PasskeyListResult, error) {
	return services.PasskeyListResult{Items: []services.PasskeyItem{{ID: "pk-1", Name: "device"}}}, nil
}
func (f *fakeAuthService) BeginPasskeyRegister(ctx context.Context, sessionID, name string, req *http.Request) (services.PasskeyBeginResult, error) {
	return services.PasskeyBeginResult{ChallengeID: "pk-register", Options: map[string]any{"challenge": "abc"}}, nil
}
func (f *fakeAuthService) FinishPasskeyRegister(ctx context.Context, sessionID, challengeID, name string, credentialJSON json.RawMessage, req *http.Request) (services.PasskeyItem, error) {
	return services.PasskeyItem{ID: "pk-1", Name: "device"}, nil
}
func (f *fakeAuthService) RenamePasskey(sessionID, id, name string) (services.PasskeyItem, error) {
	return services.PasskeyItem{ID: id, Name: name}, nil
}
func (f *fakeAuthService) DeletePasskey(sessionID, id string) error { return nil }

func TestAuthHandler_LoginSetsCookie(t *testing.T) {
	handler := NewAuthHandler(&fakeAuthService{})

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"username":"admin","password":"admin"}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	if len(resp.Result().Cookies()) == 0 {
		t.Fatal("expected session cookie")
	}
}

func TestAuthHandler_BootstrapSetsCookie(t *testing.T) {
	handler := NewAuthHandler(&fakeAuthService{})

	req := httptest.NewRequest(http.MethodPost, "/api/auth/bootstrap", bytes.NewBufferString(`{"username":"admin","email":"admin@example.test","password":"admin"}`))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	if len(resp.Result().Cookies()) == 0 {
		t.Fatal("expected bootstrap to return session cookie")
	}
}
