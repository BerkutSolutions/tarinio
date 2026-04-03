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
			SiteID:              "site-a",
			AllowedMethods:      []string{"GET", "POST"},
			MaxClientSize:       "50m",
			UseAuthBasic:        true,
			AuthBasicUser:       "admin",
			AuthBasicPassword:   "secret",
			AuthBasicText:       "Restricted area",
			AntibotChallenge:    "recaptcha",
			AntibotURI:          "/challenge",
			AntibotRecaptchaKey: "site-key",
			UseModSecurity:      true,
			UseModSecurityCRSPlugins: true,
			ModSecurityCRSVersion:    "4",
			ModSecurityCRSPlugins:    []string{"plugin-a"},
			ModSecurityCustomPath:    "modsec/anomaly_score.conf",
			ModSecurityCustomContent: "SecRuleEngine On",
			UseLimitConn:             true,
			LimitConnMaxHTTP1:        200,
			UseLimitReq:              true,
			LimitReqRate:             "100r/s",
			CORSAllowedOrigins:       []string{"*"},
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
	if !strings.Contains(siteConf, "if ($request_uri = \"/challenge\") { set $waf_antibot_verified 1; }") {
		t.Fatalf("expected antibot challenge uri rule, got: %s", siteConf)
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
	if !strings.Contains(modsecEasy, "crs_version: 4") || !strings.Contains(modsecEasy, "Include /etc/waf/modsecurity/crs-overrides/plugin-a.conf") {
		t.Fatalf("unexpected modsecurity easy artifact: %s", modsecEasy)
	}
	if !strings.Contains(modsecEasy, "SecRuleEngine On") {
		t.Fatalf("expected custom modsecurity content, got: %s", modsecEasy)
	}
	if !strings.Contains(l4guardConfig, "\"conn_limit\": 200") || !strings.Contains(l4guardConfig, "\"rate_per_second\": 100") {
		t.Fatalf("expected high default l4guard limits, got: %s", l4guardConfig)
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
	if !strings.Contains(siteConf, "if ($request_method !~ ^(GET|POST|HEAD|OPTIONS)$)") {
		t.Fatalf("expected default allowed methods guard in template, got: %s", siteConf)
	}
	if !strings.Contains(siteConf, "proxy_ssl_server_name off;") {
		t.Fatalf("expected safe ssl sni default in template, got: %s", siteConf)
	}
	if !strings.Contains(l4guardConfig, "\"conn_limit\": 200") || !strings.Contains(l4guardConfig, "\"rate_per_second\": 100") {
		t.Fatalf("expected safe l4guard defaults in template, got: %s", l4guardConfig)
	}
}
