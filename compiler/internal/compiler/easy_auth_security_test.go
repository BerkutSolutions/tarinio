package compiler

import (
	"strings"
	"testing"
)

func TestRenderEasyArtifacts_BasicAuthKeepsUsersSiteScoped(t *testing.T) {
	sites := []SiteInput{{ID: "site-a", Enabled: true, PrimaryHost: "a.example.test", ListenHTTP: true, DefaultUpstreamID: "upstream-a"}, {ID: "site-b", Enabled: true, PrimaryHost: "b.example.test", ListenHTTP: true, DefaultUpstreamID: "upstream-b"}}
	profiles := []EasyProfileInput{
		{SiteID: "site-a", UseAuthBasic: true, AuthMode: authModeBasic, AuthUsers: []ServiceAuthUserInput{{Username: "disabled-a", Password: "disabled-password", Enabled: false}, {Username: "enabled-a", Password: "enabled-password", Enabled: true}}},
		{SiteID: "site-b", UseAuthBasic: true, AuthMode: authModeBasic, AuthUsers: []ServiceAuthUserInput{{Username: "enabled-b", Password: "other-password", Enabled: true}}},
	}
	artifacts, err := RenderEasyArtifacts(sites, profiles)
	if err != nil {
		t.Fatalf("render Easy artifacts: %v", err)
	}

	byPath := mapArtifactsByPath(artifacts)
	locationArtifacts, err := RenderEasyRateLimitArtifacts(sites, []UpstreamInput{{ID: "upstream-a", SiteID: "site-a", Host: "backend-a.test", Port: 8080, Scheme: "http"}, {ID: "upstream-b", SiteID: "site-b", Host: "backend-b.test", Port: 8080, Scheme: "http"}}, profiles)
	if err != nil {
		t.Fatalf("render Easy location artifacts: %v", err)
	}
	for path, content := range mapArtifactsByPath(locationArtifacts) {
		byPath[path] = content
	}
	siteAUsers := byPath["nginx/auth-basic/site-a.htpasswd"]
	siteBUsers := byPath["nginx/auth-basic/site-b.htpasswd"]
	if strings.Contains(siteAUsers, "disabled-a:") || !strings.Contains(siteAUsers, "enabled-a:") {
		t.Fatalf("site A auth file must contain only enabled users, got: %q", siteAUsers)
	}
	if strings.Contains(siteBUsers, "enabled-a:") || !strings.Contains(siteBUsers, "enabled-b:") {
		t.Fatalf("site B auth file must contain only its own users, got: %q", siteBUsers)
	}
	if !strings.Contains(byPath["nginx/easy-locations/site-a.conf"], "auth_basic_user_file /etc/waf/nginx/auth-basic/site-a.htpasswd;") || !strings.Contains(byPath["nginx/easy-locations/site-b.conf"], "auth_basic_user_file /etc/waf/nginx/auth-basic/site-b.htpasswd;") {
		t.Fatal("each Basic Auth verification location must reference its site-scoped auth file")
	}
	if !strings.Contains(byPath["nginx/easy-locations/site-a.conf"], "try_files /__waf_auth_verified__ =204;") {
		t.Fatal("Basic Auth verification must finish after the auth_basic access phase")
	}
}

func TestRenderEasyArtifacts_DisabledUsersDoNotReactivateLegacyCredentials(t *testing.T) {
	artifacts, err := RenderEasyArtifacts(
		[]SiteInput{{ID: "site-a", Enabled: true, PrimaryHost: "a.example.test", ListenHTTP: true}},
		[]EasyProfileInput{{SiteID: "site-a", UseAuthBasic: true, AuthMode: authModeBasic, AuthBasicUser: "legacy-user", AuthBasicPassword: "legacy-password", AuthUsers: []ServiceAuthUserInput{{Username: "legacy-user", Password: "legacy-password", Enabled: false}}}},
	)
	if err != nil {
		t.Fatalf("render Easy artifacts: %v", err)
	}
	byPath := mapArtifactsByPath(artifacts)
	if _, exists := byPath["nginx/auth-basic/site-a.htpasswd"]; exists {
		t.Fatal("disabled users must not produce a Basic Auth credential file")
	}
	if strings.Contains(byPath["nginx/easy-locations/site-a.conf"], "location = /auth/verify/basic") {
		t.Fatal("Basic Auth verification location must be omitted when no enabled Basic Auth users exist")
	}
}
