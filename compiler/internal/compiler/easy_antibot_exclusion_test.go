package compiler

import (
	"strings"
	"testing"
)

func TestRenderEasyArtifacts_AntibotExclusionsBypassChallenge(t *testing.T) {
	site := SiteInput{
		ID:                "site-a",
		Enabled:           true,
		PrimaryHost:       "a.example.com",
		ListenHTTP:        true,
		DefaultUpstreamID: "upstream-a",
	}
	profile := EasyProfileInput{
		SiteID:           "site-a",
		SecurityMode:     "block",
		AllowedMethods:   []string{"GET", "POST", "HEAD"},
		MaxClientSize:    "50m",
		UseModSecurity:   true,
		AntibotChallenge: "javascript",
		AntibotURI:       "/challenge",
		AntibotExclusionRules: []AntibotExclusionRuleInput{
			{Path: "/api/2/envelope/", Methods: []string{"POST"}},
			{Path: "/healthz", Methods: []string{"HEAD", "GET"}},
			{Path: "/metrics", Methods: []string{"*"}},
		},
	}

	artifacts, err := RenderEasyArtifacts([]SiteInput{site}, []EasyProfileInput{profile})
	if err != nil {
		t.Fatalf("render easy artifacts: %v", err)
	}

	siteConf := ""
	for _, item := range artifacts {
		if item.Path == "nginx/easy/site-a.conf" {
			siteConf = string(item.Content)
			break
		}
	}
	if siteConf == "" {
		t.Fatal("expected nginx/easy/site-a.conf artifact")
	}
	if !strings.Contains(siteConf, `set $waf_antibot_exclusion_match "$request_method:$uri";`) {
		t.Fatalf("expected antibot exclusion matcher, got: %s", siteConf)
	}
	if !strings.Contains(siteConf, `if ($waf_antibot_exclusion_match ~* "^(?:POST):/api/2/envelope/") { set $waf_antibot_exception_guard 1; }`) {
		t.Fatalf("expected POST-only exclusion for envelope api, got: %s", siteConf)
	}
	if !strings.Contains(siteConf, `if ($waf_antibot_exclusion_match ~* "^(?:GET|HEAD):/healthz$") { set $waf_antibot_exception_guard 1; }`) {
		t.Fatalf("expected exact GET/HEAD health exclusion, got: %s", siteConf)
	}
	if !strings.Contains(siteConf, `if ($waf_antibot_exclusion_match ~* "^[A-Z]+:/metrics$") { set $waf_antibot_exception_guard 1; }`) {
		t.Fatalf("expected wildcard exclusion for /metrics, got: %s", siteConf)
	}
}
