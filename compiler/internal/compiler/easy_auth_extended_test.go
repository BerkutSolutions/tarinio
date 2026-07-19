package compiler

import (
	"strings"
	"testing"
)

func TestRenderEasyArtifacts_AuthTokenOrderAndExclusions(t *testing.T) {
	artifacts, err := RenderEasyArtifacts(
		[]SiteInput{{ID: "site-a", Enabled: true, PrimaryHost: "a.example.com", ListenHTTP: true, DefaultUpstreamID: "site-a-upstream"}},
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
	if !strings.Contains(siteConf, `if ($uri = /auth/verify/basic) {`) || !strings.Contains(siteConf, `if ($uri = /auth/verify/token) {`) {
		t.Fatalf("expected auth verification endpoints to bypass site method filtering, got: %s", siteConf)
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
		[]SiteInput{{ID: "management-site", Enabled: true, PrimaryHost: "ui", ListenHTTP: true, DefaultUpstreamID: "management-upstream"}},
		[]EasyProfileInput{{
			SiteID:           "management-site",
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
	siteConf := byPath["nginx/easy/management-site.conf"]
	if !strings.Contains(siteConf, `if ($waf_auth_exclusion_match ~* "^(?:DELETE|GET|HEAD|PATCH|POST|PUT):^/api/administration$")`) || !strings.Contains(siteConf, `if ($waf_auth_exclusion_match ~* "^(?:DELETE|GET|HEAD|PATCH|POST|PUT):^/api/administration/$")`) {
		t.Fatalf("expected management auth exclusion for write api paths, got: %s", siteConf)
	}
	if !strings.Contains(siteConf, `if ($waf_auth_exclusion_match ~* "^(?:DELETE|GET|HEAD|PATCH|POST|PUT):^/services$")`) {
		t.Fatalf("expected management auth exclusion for services pages, got: %s", siteConf)
	}
	if !strings.Contains(siteConf, `modsecurity off;`) && !strings.Contains(siteConf, `ctl:ruleEngine=Off`) {
		t.Fatalf("expected management modsecurity self-bypass safeguard rule, got: %s", siteConf)
	}
}

func TestRenderEasyArtifacts_BasicAuthTemplateVariants(t *testing.T) {
	for _, variant := range []string{"v1", "v2", "v3", "v4", "v5", "v6", "v7", "v8", "v9"} {
		t.Run(variant, func(t *testing.T) {
			artifacts, err := RenderEasyArtifacts(
				[]SiteInput{{ID: "site-a", Enabled: true, PrimaryHost: "a.example.com", ListenHTTP: true}},
				[]EasyProfileInput{{
					SiteID: "site-a", AllowedMethods: []string{"GET"}, UseAuthBasic: true,
					AuthMode: "basic", AuthBasicText: "Restricted area", AuthBasicTemplate: variant,
					AuthUsers: []ServiceAuthUserInput{{Username: "admin", Password: "secret", Enabled: true}},
				}},
			)
			if err != nil {
				t.Fatalf("render easy artifacts: %v", err)
			}
			page := mapArtifactsByPath(artifacts)["errors/site-a/auth.html"]
			if !strings.Contains(page, `body class="`+variant+`"`) || !strings.Contains(page, `fetch("/auth/verify/basic"`) {
				t.Fatalf("expected themed, working Basic Auth page for %s", variant)
			}
			if !strings.Contains(page, "logo800x300_no-text.png") || !strings.Contains(page, "logo512.png") {
				t.Fatalf("expected local logos in %s", variant)
			}
			if !strings.Contains(page, "https://github.com/BerkutSolutions/tarinio") || !strings.Contains(page, "Secured by") {
				t.Fatalf("expected Tarinio footer link in %s", variant)
			}
			for _, language := range []string{"en:", "ru:", "de:", "sr:", "zh:"} {
				if !strings.Contains(page, language) {
					t.Fatalf("expected %s localization in %s", language, variant)
				}
			}
		})
	}
}
