package compiler

import (
	"strings"
	"testing"
)

func TestRenderEasyArtifacts_GeneratesSiteAndAuthBasicFiles(t *testing.T) {
	artifacts, err := RenderEasyArtifacts(
		[]SiteInput{{
			ID:          "site-a",
			Enabled:     true,
			PrimaryHost: "a.example.com",
			ListenHTTP:  true,
		}},
		[]EasyProfileInput{{
			SiteID:                            "site-a",
			SecurityMode:                      "block",
			AllowedMethods:                    []string{"GET", "POST"},
			MaxClientSize:                     "50m",
			UseAuthBasic:                      true,
			AuthBasicUser:                     "admin",
			AuthBasicPassword:                 "secret",
			AuthBasicText:                     "Restricted area",
			AntibotChallenge:                  "recaptcha",
			AntibotURI:                        "/challenge",
			AntibotRecaptchaKey:               "site-key",
			UseModSecurity:                    true,
			UseModSecurityCRSPlugins:          true,
			UseModSecurityCustomConfiguration: true,
			ModSecurityCRSVersion:             "4",
			ModSecurityCRSPlugins:             []string{"plugin-a"},
			ModSecurityCustomPath:             "modsec/anomaly_score.conf",
			ModSecurityCustomContent:          "SecRuleEngine On",
			UseLimitConn:                      true,
			LimitConnMaxHTTP1:                 200,
			UseLimitReq:                       true,
			LimitReqRate:                      "100r/s",
			CORSAllowedOrigins:                []string{"*"},
		}},
	)
	if err != nil {
		t.Fatalf("render easy artifacts: %v", err)
	}

	var siteConf string
	var htpasswd string
	var modsecEasy string
	var l4guardConfig string
	for _, item := range artifacts {
		if item.Path == "nginx/easy/site-a.conf" {
			siteConf = string(item.Content)
		}
		if item.Path == "nginx/auth-basic/site-a.htpasswd" {
			htpasswd = string(item.Content)
		}
		if item.Path == "modsecurity/easy/site-a.conf" {
			modsecEasy = string(item.Content)
		}
		if item.Path == "l4guard/config.json" {
			l4guardConfig = string(item.Content)
		}
	}
	if siteConf == "" {
		t.Fatal("expected easy site conf artifact")
	}
	if !strings.Contains(siteConf, "client_max_body_size 50m;") {
		t.Fatalf("expected max_client_size directive, got: %s", siteConf)
	}
	if !strings.Contains(siteConf, "auth_basic \"Restricted area\";") {
		t.Fatalf("expected auth_basic realm, got: %s", siteConf)
	}
	if !strings.Contains(siteConf, "if ($cookie_waf_antibot_") {
		t.Fatalf("expected site-scoped antibot cookie rule, got: %s", siteConf)
	}
	if !strings.Contains(siteConf, "if ($waf_antibot_guard = \"0:0:1\") { return 302 /challenge?return_uri=$uri&return_args=$args; }") {
		t.Fatalf("expected antibot redirect challenge in easy template, got: %s", siteConf)
	}
	if !strings.Contains(siteConf, `if ($uri ~* "^/(`) || !strings.Contains(siteConf, `api/.*`) || !strings.Contains(siteConf, `static/.*`) || !strings.Contains(siteConf, `dashboard(?:/.*)?`) || !strings.Contains(siteConf, `login/2fa`) {
		t.Fatalf("expected admin path bypass guard in easy template, got: %s", siteConf)
	}
	if !strings.Contains(siteConf, `if ($cookie_waf_session != "")`) || !strings.Contains(siteConf, `if ($cookie_waf_session_boot != "")`) {
		t.Fatalf("expected admin session cookie bypass guard in easy template, got: %s", siteConf)
	}
	if !strings.Contains(siteConf, "if ($waf_allow_bypass_site_a = 1)") {
		t.Fatalf("expected allowlist bypass guard in easy template, got: %s", siteConf)
	}
	if !strings.Contains(siteConf, "add_header X-WAF-Antibot-Mode \"recaptcha\" always;") {
		t.Fatalf("expected antibot mode header, got: %s", siteConf)
	}
	if !strings.Contains(siteConf, "add_header X-WAF-Antibot-Provider \"recaptcha\" always;") {
		t.Fatalf("expected antibot provider header, got: %s", siteConf)
	}
	if !strings.Contains(siteConf, "modsecurity_rules_file /etc/waf/modsecurity/easy/site-a.conf;") {
		t.Fatalf("expected modsecurity easy rules include, got: %s", siteConf)
	}
	if htpasswd != "admin:{SHA}5en6G6MezRroT3XKqkdPOmY/BfQ=\n" {
		t.Fatalf("unexpected htpasswd content: %q", htpasswd)
	}
	if !strings.Contains(modsecEasy, "# custom_path: modsec/anomaly_score.conf") {
		t.Fatalf("expected custom path marker in easy artifact, got: %s", modsecEasy)
	}
	if !strings.Contains(modsecEasy, "Include /etc/waf/modsecurity/coreruleset/rules/*.conf") {
		t.Fatalf("expected CRS include in easy modsecurity content, got: %s", modsecEasy)
	}
	if !strings.Contains(modsecEasy, "Include /etc/waf/modsecurity/crs-overrides/plugin-a.conf") {
		t.Fatalf("expected CRS plugin include in easy modsecurity content, got: %s", modsecEasy)
	}
	if !strings.Contains(modsecEasy, "SecRuleEngine On") {
		t.Fatalf("expected custom modsecurity content, got: %s", modsecEasy)
	}
	if !strings.Contains(l4guardConfig, "\"conn_limit\": 200") || !strings.Contains(l4guardConfig, "\"rate_per_second\": 100") {
		t.Fatalf("expected high default l4guard limits, got: %s", l4guardConfig)
	}

	var challengePage string
	for _, item := range artifacts {
		if item.Path == "errors/site-a/antibot.html" {
			challengePage = string(item.Content)
			break
		}
	}
	if !strings.Contains(challengePage, `var verifyURI = "/challenge/verify";`) {
		t.Fatalf("expected antibot interstitial artifact with verify uri, got: %s", challengePage)
	}
}

func TestRenderEasyArtifacts_ModSecurityEasyFileWithoutCustomConfig(t *testing.T) {
	artifacts, err := RenderEasyArtifacts(
		[]SiteInput{
			{ID: "site-a", Enabled: true, PrimaryHost: "a.example.com", ListenHTTP: true},
		},
		[]EasyProfileInput{
			{
				SiteID:                            "site-a",
				SecurityMode:                      "block",
				UseModSecurity:                    true,
				UseModSecurityCRSPlugins:          true,
				UseModSecurityCustomConfiguration: false,
				ModSecurityCRSVersion:             "4",
				AllowedMethods:                    []string{"GET"},
				MaxClientSize:                     "100m",
				UseLimitConn:                      true,
				LimitConnMaxHTTP1:                 200,
				UseLimitReq:                       true,
				LimitReqRate:                      "100r/s",
			},
		},
	)
	if err != nil {
		t.Fatalf("render easy artifacts: %v", err)
	}
	byPath := map[string]string{}
	for _, item := range artifacts {
		byPath[item.Path] = string(item.Content)
	}
	nginxEasy := byPath["nginx/easy/site-a.conf"]
	if !strings.Contains(nginxEasy, "modsecurity_rules_file /etc/waf/modsecurity/easy/site-a.conf;") {
		t.Fatalf("expected easy modsecurity file include: %s", nginxEasy)
	}
	modsecEasy, ok := byPath["modsecurity/easy/site-a.conf"]
	if !ok {
		t.Fatalf("expected modsecurity easy artifact without custom configuration")
	}
	if !strings.Contains(modsecEasy, "Include /etc/waf/modsecurity/coreruleset/rules/*.conf") {
		t.Fatalf("expected CRS include in easy modsecurity content, got: %s", modsecEasy)
	}
	if strings.Contains(modsecEasy, "# custom_path:") {
		t.Fatalf("did not expect custom path marker without custom configuration: %s", modsecEasy)
	}
}

func TestRenderEasyArtifacts_UsesSafeTemplateWhenProfileMissing(t *testing.T) {
	artifacts, err := RenderEasyArtifacts(
		[]SiteInput{{
			ID:          "monitor-site",
			Enabled:     true,
			PrimaryHost: "monitor.example.com",
			ListenHTTP:  true,
		}},
		nil,
	)
	if err != nil {
		t.Fatalf("render easy artifacts: %v", err)
	}

	var siteConf string
	var l4guardConfig string
	for _, item := range artifacts {
		if item.Path == "nginx/easy/monitor-site.conf" {
			siteConf = string(item.Content)
		}
		if item.Path == "l4guard/config.json" {
			l4guardConfig = string(item.Content)
		}
	}
	if siteConf == "" {
		t.Fatal("expected easy site conf artifact for monitor-site")
	}
	if !strings.Contains(siteConf, "client_max_body_size 100m;") {
		t.Fatalf("expected default max client size in template, got: %s", siteConf)
	}
	if !strings.Contains(siteConf, "if ($request_method !~ ^(GET|POST|HEAD|OPTIONS|PUT|PATCH|DELETE)$)") {
		t.Fatalf("expected default allowed methods guard in template, got: %s", siteConf)
	}
	if !strings.Contains(siteConf, "proxy_ssl_server_name off;") {
		t.Fatalf("expected safe ssl sni default in template, got: %s", siteConf)
	}
	if !strings.Contains(l4guardConfig, "\"conn_limit\": 200") || !strings.Contains(l4guardConfig, "\"rate_per_second\": 100") {
		t.Fatalf("expected safe l4guard defaults in template, got: %s", l4guardConfig)
	}
}

func TestRenderEasyArtifacts_BlacklistURIWildcardPattern(t *testing.T) {
	artifacts, err := RenderEasyArtifacts(
		[]SiteInput{{
			ID:          "site-a",
			Enabled:     true,
			PrimaryHost: "a.example.com",
			ListenHTTP:  true,
		}},
		[]EasyProfileInput{{
			SiteID:            "site-a",
			SecurityMode:      "block",
			AllowedMethods:    []string{"GET"},
			MaxClientSize:     "50m",
			UseLimitConn:      true,
			LimitConnMaxHTTP1: 100,
			UseLimitReq:       true,
			LimitReqRate:      "10r/s",
			BlacklistURI:      []string{"*.php"},
		}},
	)
	if err != nil {
		t.Fatalf("render easy artifacts: %v", err)
	}

	var siteConf string
	for _, item := range artifacts {
		if item.Path == "nginx/easy/site-a.conf" {
			siteConf = string(item.Content)
			break
		}
	}
	if siteConf == "" {
		t.Fatal("expected easy site conf artifact")
	}
	if !strings.Contains(siteConf, `if ($waf_blacklist_uri_guard ~* "^0:.*.*\.php") { return 403; }`) {
		t.Fatalf("expected wildcard blacklist uri to be rendered as safe regex, got: %s", siteConf)
	}
}
