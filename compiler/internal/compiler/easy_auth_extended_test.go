package compiler

import (
	"strings"
	"testing"
)

func TestRenderEasyArtifacts_AuthTokenOrderAndExclusions(t *testing.T) {
	artifacts, err := RenderEasyArtifacts(
		[]SiteInput{{ID: "site-a", Enabled: true, PrimaryHost: "a.example.com", ListenHTTP: true, DefaultUpstreamID: "site-a-upstream"}},
		[]EasyProfileInput{{
			SiteID:            "site-a",
			SecurityMode:      "block",
			AllowedMethods:    []string{"GET", "POST"},
			MaxClientSize:     "50m",
			UseAuthBasic:      true,
			AuthMode:          "basic_or_token",
			AuthOrder:         "antibot_first",
			AuthBasicText:     "Restricted area",
			AuthUsers:         []ServiceAuthUserInput{{Username: "admin", Password: "secret", Enabled: true}},
			AuthServiceTokens: []ServiceAuthTokenInput{{ServiceName: "sentry-ingest", Token: "token-1", Enabled: true}},
			AuthExclusionRules: []AuthExclusionRuleInput{{Path: "/api/public/", Methods: []string{"GET", "OPTIONS"}}},
			AntibotChallenge:  "javascript",
			AntibotURI:        "/challenge",
		}},
	)
	if err != nil {
		t.Fatalf("render easy artifacts: %v", err)
	}
	locationArtifacts, err := RenderEasyRateLimitArtifacts(
		[]SiteInput{{ID: "site-a", Enabled: true, PrimaryHost: "a.example.com", ListenHTTP: true, DefaultUpstreamID: "site-a-upstream"}},
		[]UpstreamInput{{ID: "site-a-upstream", SiteID: "site-a", Host: "backend.internal", Port: 8080, Scheme: "http"}},
		[]EasyProfileInput{{
			SiteID:             "site-a",
			SecurityMode:       "block",
			AllowedMethods:     []string{"GET", "POST"},
			MaxClientSize:      "50m",
			UseAuthBasic:       true,
			AuthMode:           "basic_or_token",
			AuthOrder:          "antibot_first",
			AuthBasicText:      "Restricted area",
			AuthUsers:          []ServiceAuthUserInput{{Username: "admin", Password: "secret", Enabled: true}},
			AuthServiceTokens:  []ServiceAuthTokenInput{{ServiceName: "sentry-ingest", Token: "token-1", Enabled: true}},
			AuthExclusionRules: []AuthExclusionRuleInput{{Path: "/api/public/", Methods: []string{"GET", "OPTIONS"}}},
			AntibotChallenge:   "javascript",
			AntibotURI:         "/challenge",
		}},
	)
	if err != nil {
		t.Fatalf("render easy rate limit artifacts: %v", err)
	}
	byPath := mapArtifactsByPath(artifacts)
	for path, content := range mapArtifactsByPath(locationArtifacts) {
		byPath[path] = content
	}
	siteConf := byPath["nginx/easy/site-a.conf"]
	locationsConf := byPath["nginx/easy-locations/site-a.conf"]
	authPage := byPath["errors/site-a/auth.html"]

	if !strings.Contains(siteConf, `set $waf_auth_service_guard "$http_x_waf_service_name|$http_x_waf_service_token";`) {
		t.Fatalf("expected service token request guard, got: %s", siteConf)
	}
	if !strings.Contains(siteConf, `if ($waf_auth_exclusion_match ~* "^(?:GET|OPTIONS):`) {
		t.Fatalf("expected auth exclusion rule in site conf, got: %s", siteConf)
	}
	if strings.Index(siteConf, `# Easy antibot guard`) > strings.Index(siteConf, `set $waf_auth_gate_skip 0;`) {
		t.Fatalf("expected antibot-first auth order to place auth gate after antibot, got: %s", siteConf)
	}
	if !strings.Contains(locationsConf, "location = /auth/verify/basic {") || !strings.Contains(locationsConf, "location = /auth/verify/token {") {
		t.Fatalf("expected dedicated basic and token verify locations, got: %s", locationsConf)
	}
	if !strings.Contains(authPage, `request("/auth/verify/token"`) || !strings.Contains(authPage, `request("/auth/verify/basic"`) {
		t.Fatalf("expected auth page to support both verify flows, got: %s", authPage)
	}
}

func TestRenderEasyArtifacts_ManagementSiteAuthBypassesWriteAPI(t *testing.T) {
	artifacts, err := RenderEasyArtifacts(
		[]SiteInput{{ID: "prewaf.hantico.ru", Enabled: true, PrimaryHost: "ui", ListenHTTP: true, DefaultUpstreamID: "prewaf-upstream"}},
		[]EasyProfileInput{{
			SiteID:           "prewaf.hantico.ru",
			SecurityMode:     "block",
			AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
			MaxClientSize:    "50m",
			UseAuthBasic:     true,
			AuthMode:         "basic",
			AuthOrder:        "auth_first",
			AuthBasicText:    "Restricted area",
			AuthUsers:        []ServiceAuthUserInput{{Username: "admin", Password: "secret", Enabled: true}},
			AntibotChallenge: "javascript",
			AntibotURI:       "/challenge",
		}},
	)
	if err != nil {
		t.Fatalf("render easy artifacts: %v", err)
	}
	byPath := mapArtifactsByPath(artifacts)
	siteConf := byPath["nginx/easy/prewaf.hantico.ru.conf"]
	if !strings.Contains(siteConf, `if ($waf_auth_exclusion_match ~* "^(?:DELETE|GET|HEAD|PATCH|POST|PUT):^/api/$")`) {
		t.Fatalf("expected management auth exclusion for write api paths, got: %s", siteConf)
	}
	if !strings.Contains(siteConf, `if ($waf_auth_exclusion_match ~* "^(?:DELETE|GET|HEAD|PATCH|POST|PUT):^/services$")`) {
		t.Fatalf("expected management auth exclusion for services pages, got: %s", siteConf)
	}
}
