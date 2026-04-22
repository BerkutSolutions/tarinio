package app

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"waf/control-plane/internal/config"
)

func TestNew_WiresHTTPServerAndRevisionStore(t *testing.T) {
	cfg := config.Config{
		HTTPAddr:         "127.0.0.1:8080",
		RuntimeRoot:      "/tmp/runtime",
		RevisionStoreDir: t.TempDir(),
		AuthIssuer:       "WAF",
		BootstrapAdmin: config.BootstrapAdminConfig{
			Enabled:  true,
			ID:       "admin",
			Username: "admin",
			Email:    "admin@example.test",
			Password: "admin",
		},
	}

	application, err := New(cfg)
	if err != nil {
		t.Fatalf("app bootstrap failed: %v", err)
	}
	if application.HTTPServer == nil || application.RevisionStore == nil || application.RevisionSnapshotStore == nil || application.RevisionService == nil || application.RevisionCompileService == nil || application.ApplyService == nil || application.EventStore == nil || application.EventService == nil || application.ReportService == nil || application.JobStore == nil || application.JobService == nil || application.RoleStore == nil || application.SessionStore == nil || application.UserStore == nil || application.AuthService == nil || application.SiteStore == nil || application.SiteService == nil || application.ManualBanService == nil || application.UpstreamStore == nil || application.UpstreamService == nil || application.CertificateStore == nil || application.CertificateService == nil || application.CertificateMaterialStore == nil || application.CertificateUploadService == nil || application.LetsEncryptService == nil || application.TLSConfigStore == nil || application.TLSConfigService == nil || application.WAFPolicyStore == nil || application.WAFPolicyService == nil || application.AccessPolicyStore == nil || application.AccessPolicyService == nil || application.RateLimitPolicyStore == nil || application.RateLimitPolicyService == nil {
		t.Fatal("expected app dependencies to be wired")
	}
	if application.RedisBackend != nil {
		t.Fatal("expected standalone app bootstrap without redis backend")
	}

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	resp := httptest.NewRecorder()
	application.HTTPServer.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200 from health endpoint, got %d", resp.Code)
	}
}

func TestNew_ProtectsControlPlaneEndpointsWithAuth(t *testing.T) {
	cfg := config.Config{
		HTTPAddr:         "127.0.0.1:8080",
		RuntimeRoot:      "/tmp/runtime",
		RevisionStoreDir: t.TempDir(),
		AuthIssuer:       "WAF",
		BootstrapAdmin: config.BootstrapAdminConfig{
			Enabled:  true,
			ID:       "admin",
			Username: "admin",
			Email:    "admin@example.test",
			Password: "admin",
		},
	}

	application, err := New(cfg)
	if err != nil {
		t.Fatalf("app bootstrap failed: %v", err)
	}

	protectedReq := httptest.NewRequest(http.MethodGet, "/api/sites", nil)
	protectedResp := httptest.NewRecorder()
	application.HTTPServer.Handler().ServeHTTP(protectedResp, protectedReq)
	if protectedResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without session, got %d", protectedResp.Code)
	}

	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"username":"admin","password":"admin"}`))
	loginResp := httptest.NewRecorder()
	application.HTTPServer.Handler().ServeHTTP(loginResp, loginReq)
	if loginResp.Code != http.StatusOK {
		t.Fatalf("expected 200 from login, got %d", loginResp.Code)
	}
	cookies := loginResp.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected auth cookie after login")
	}
	var sessionCookie, bootCookie *http.Cookie
	for i := range cookies {
		switch cookies[i].Name {
		case "waf_session":
			sessionCookie = cookies[i]
		case "waf_session_boot":
			bootCookie = cookies[i]
		}
	}
	if sessionCookie == nil {
		t.Fatal("expected waf_session cookie after login")
	}
	if bootCookie == nil {
		t.Fatal("expected waf_session_boot cookie after login")
	}

	authedReq := httptest.NewRequest(http.MethodGet, "/api/sites", nil)
	authedReq.AddCookie(sessionCookie)
	authedReq.AddCookie(bootCookie)
	authedResp := httptest.NewRecorder()
	application.HTTPServer.Handler().ServeHTTP(authedResp, authedReq)
	if authedResp.Code != http.StatusOK {
		t.Fatalf("expected 200 with session, got %d", authedResp.Code)
	}
}

func TestNew_RejectsSessionWithoutBootCookie(t *testing.T) {
	cfg := config.Config{
		HTTPAddr:         "127.0.0.1:8080",
		RuntimeRoot:      "/tmp/runtime",
		RevisionStoreDir: t.TempDir(),
		AuthIssuer:       "WAF",
		BootstrapAdmin: config.BootstrapAdminConfig{
			Enabled:  true,
			ID:       "admin",
			Username: "admin",
			Email:    "admin@example.test",
			Password: "admin",
		},
	}

	application, err := New(cfg)
	if err != nil {
		t.Fatalf("app bootstrap failed: %v", err)
	}

	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"username":"admin","password":"admin"}`))
	loginResp := httptest.NewRecorder()
	application.HTTPServer.Handler().ServeHTTP(loginResp, loginReq)
	if loginResp.Code != http.StatusOK {
		t.Fatalf("expected 200 from login, got %d", loginResp.Code)
	}

	var sessionCookie *http.Cookie
	for _, cookie := range loginResp.Result().Cookies() {
		if cookie.Name == "waf_session" {
			sessionCookie = cookie
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("expected session cookie")
	}

	protectedReq := httptest.NewRequest(http.MethodGet, "/api/sites", nil)
	protectedReq.AddCookie(sessionCookie)
	protectedResp := httptest.NewRecorder()
	application.HTTPServer.Handler().ServeHTTP(protectedResp, protectedReq)
	if protectedResp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without boot cookie, got %d", protectedResp.Code)
	}
}
