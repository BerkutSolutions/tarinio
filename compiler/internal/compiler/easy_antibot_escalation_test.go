package compiler

import (
	"strings"
	"testing"
)

func TestRenderEasyArtifacts_AntibotTwoLayerAndRules(t *testing.T) {
	site := SiteInput{
		ID:                "site-a",
		Enabled:           true,
		PrimaryHost:       "a.example.com",
		ListenHTTP:        true,
		DefaultUpstreamID: "upstream-a",
	}
	upstream := UpstreamInput{
		ID:             "upstream-a",
		SiteID:         "site-a",
		Name:           "upstream-a",
		Scheme:         "http",
		Host:           "upstream",
		Port:           8080,
		BasePath:       "/",
		PassHostHeader: true,
	}
	profile := EasyProfileInput{
		SiteID:                     "site-a",
		SecurityMode:               "block",
		AllowedMethods:             []string{"GET", "POST"},
		MaxClientSize:              "50m",
		UseModSecurity:             true,
		UseLimitConn:               true,
		LimitConnMaxHTTP1:          200,
		UseLimitReq:                true,
		LimitReqRate:               "100r/s",
		AntibotChallenge:           "javascript",
		AntibotURI:                 "/challenge",
		ChallengeEscalationEnabled: true,
		ChallengeEscalationMode:    "turnstile",
		AntibotChallengeRules: []AntibotChallengeRuleInput{
			{Path: "/login", Challenge: "recaptcha"},
			{Path: "/api/auth/*", Challenge: "cookie"},
		},
	}

	easyArtifacts, err := RenderEasyArtifacts([]SiteInput{site}, []EasyProfileInput{profile})
	if err != nil {
		t.Fatalf("render easy artifacts: %v", err)
	}
	rateArtifacts, err := RenderEasyRateLimitArtifacts([]SiteInput{site}, []UpstreamInput{upstream}, []EasyProfileInput{profile})
	if err != nil {
		t.Fatalf("render easy rate-limit artifacts: %v", err)
	}

	byPath := map[string]string{}
	for _, item := range append(easyArtifacts, rateArtifacts...) {
		byPath[item.Path] = string(item.Content)
	}

	siteConf := byPath["nginx/easy/site-a.conf"]
	if !strings.Contains(siteConf, `if ($waf_antibot_stage_guard = "0:0:1:0") { return 302 /challenge/stage1/verify?return_uri=$uri&return_args=$args; }`) {
		t.Fatalf("expected stage1 redirect in easy site conf, got: %s", siteConf)
	}
	if !strings.Contains(siteConf, `if ($uri ~* "^/login$")`) || !strings.Contains(siteConf, `set $waf_antibot_effective_challenge "recaptcha";`) {
		t.Fatalf("expected exact challenge override for /login, got: %s", siteConf)
	}
	if !strings.Contains(siteConf, `if ($uri ~* "^/api/auth/")`) || !strings.Contains(siteConf, `set $waf_antibot_effective_redirect "/challenge/verify";`) {
		t.Fatalf("expected prefix cookie challenge override, got: %s", siteConf)
	}
	if !strings.Contains(siteConf, `set $waf_antibot_security_reason "antibot";`) {
		t.Fatalf("expected blocked antibot request to emit its telemetry reason, got: %s", siteConf)
	}

	locationsConf := byPath["nginx/easy-locations/site-a.conf"]
	if !strings.Contains(locationsConf, "location = /challenge/stage1 {") || !strings.Contains(locationsConf, "location = /challenge/stage1/verify {") {
		t.Fatalf("expected stage1 antibot locations, got: %s", locationsConf)
	}
}

func TestRenderEasyArtifacts_AntibotBypassesRateLimitFallback(t *testing.T) {
	site := SiteInput{
		ID:                "site-a",
		Enabled:           true,
		PrimaryHost:       "a.example.com",
		ListenHTTP:        true,
		DefaultUpstreamID: "upstream-a",
	}
	profile := EasyProfileInput{
		SiteID:                     "site-a",
		SecurityMode:               "block",
		AllowedMethods:             []string{"GET"},
		MaxClientSize:              "50m",
		UseBadBehavior:             true,
		BadBehaviorStatusCodes:     []int{429},
		BadBehaviorBanTimeSeconds:  60,
		AntibotChallenge:           "javascript",
		AntibotURI:                 "/challenge",
		ChallengeEscalationEnabled: true,
		ChallengeEscalationMode:    "captcha",
	}

	artifacts, err := RenderEasyArtifacts([]SiteInput{site}, []EasyProfileInput{profile})
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
		t.Fatal("expected nginx/easy/site-a.conf artifact")
	}

	if !strings.Contains(siteConf, `if ($waf_antibot_exception_guard = 1) {`) {
		t.Fatalf("expected rate-limit bypass for antibot challenge endpoints, got: %s", siteConf)
	}
	if !strings.Contains(siteConf, `if ($waf_antibot_stage_guard = "0:0:1:0") { return 302 /challenge/stage1/verify?return_uri=$uri&return_args=$args; }`) {
		t.Fatalf("expected stage1 redirect to remain present, got: %s", siteConf)
	}
	bypassMarker := "if ($waf_antibot_exception_guard = 1) {"
	antibotGuardPos := strings.Index(siteConf, `# Easy antibot guard: requests are challenged unless a site-scoped challenge cookie is present.`)
	bypassPos := strings.Index(siteConf, bypassMarker)
	if bypassPos == -1 {
		t.Fatalf("expected antibot bypass to clear rate-limit flags, got: %s", siteConf)
	}
	if antibotGuardPos == -1 || bypassPos < antibotGuardPos {
		t.Fatalf("expected antibot rate-limit bypass to be computed after antibot guard variables are prepared, got: %s", siteConf)
	}
	rateLimitSetupPos := strings.Index(siteConf, `if ($cookie_{{ .RateLimitEscalationCookieVar }} = "1") {`)
	_ = rateLimitSetupPos
	rateLimitReturnPos := strings.Index(siteConf, `if ($waf_rate_limited_active = 1) {`)
	if rateLimitReturnPos == -1 {
		t.Fatalf("expected legacy 429 return guard, got: %s", siteConf)
	}
}

func TestRenderEasyArtifacts_AntibotRuleAllowsNoOverride(t *testing.T) {
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
		AllowedMethods:   []string{"GET"},
		MaxClientSize:    "50m",
		AntibotChallenge: "javascript",
		AntibotURI:       "/challenge",
		AntibotChallengeRules: []AntibotChallengeRuleInput{
			{Path: "/captcha", Challenge: "no"},
			{Path: "/login", Challenge: "captcha"},
		},
	}

	artifacts, err := RenderEasyArtifacts([]SiteInput{site}, []EasyProfileInput{profile})
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
		t.Fatal("expected nginx/easy/site-a.conf artifact")
	}

	if !strings.Contains(siteConf, `if ($uri ~* "^/captcha$")`) {
		t.Fatalf("expected /captcha antibot override, got: %s", siteConf)
	}
	if !strings.Contains(siteConf, `set $waf_antibot_effective_challenge "no";`) {
		t.Fatalf("expected /captcha override to support no challenge mode, got: %s", siteConf)
	}
	if !strings.Contains(siteConf, `set $waf_antibot_effective_redirect "";`) {
		t.Fatalf("expected no-challenge override to clear redirect target, got: %s", siteConf)
	}
}

func TestRenderEasyArtifacts_AntibotTwoLayerSamePathReloadsToSecondStage(t *testing.T) {
	site := SiteInput{
		ID:                "site-a",
		Enabled:           true,
		PrimaryHost:       "a.example.com",
		ListenHTTP:        true,
		DefaultUpstreamID: "upstream-a",
	}
	profile := EasyProfileInput{
		SiteID:                     "site-a",
		SecurityMode:               "block",
		AllowedMethods:             []string{"GET"},
		MaxClientSize:              "50m",
		AntibotChallenge:           "javascript",
		AntibotURI:                 "/challenge",
		ChallengeEscalationEnabled: true,
		ChallengeEscalationMode:    "captcha",
		AntibotChallengeRules: []AntibotChallengeRuleInput{
			{Path: "/challenge", Challenge: "captcha"},
		},
	}

	artifacts, err := RenderEasyArtifacts([]SiteInput{site}, []EasyProfileInput{profile})
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
		t.Fatal("expected nginx/easy/site-a.conf artifact")
	}

	if !strings.Contains(siteConf, `if ($uri ~* "^/challenge$")`) {
		t.Fatalf("expected same-path /challenge override for second stage, got: %s", siteConf)
	}
	if !strings.Contains(siteConf, `set $waf_antibot_effective_challenge "captcha";`) {
		t.Fatalf("expected /challenge override to switch second stage to captcha, got: %s", siteConf)
	}
	if !strings.Contains(siteConf, `if ($waf_antibot_stage_guard = "0:0:1:1") { return 302 $waf_antibot_effective_redirect?return_uri=$uri&return_args=$args; }`) {
		t.Fatalf("expected second stage to reuse the same path via effective redirect, got: %s", siteConf)
	}
}
