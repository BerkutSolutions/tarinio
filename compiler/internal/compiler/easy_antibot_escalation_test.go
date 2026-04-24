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
	if !strings.Contains(siteConf, "if ($waf_antibot_stage1_verified = 0) { return 302 /challenge/stage1/verify?return_uri=$uri&return_args=$args; }") {
		t.Fatalf("expected stage1 redirect in easy site conf, got: %s", siteConf)
	}
	if !strings.Contains(siteConf, `if ($uri ~* "^/login$")`) || !strings.Contains(siteConf, `set $waf_antibot_effective_challenge "recaptcha";`) {
		t.Fatalf("expected exact challenge override for /login, got: %s", siteConf)
	}
	if !strings.Contains(siteConf, `if ($uri ~* "^/api/auth/")`) || !strings.Contains(siteConf, `set $waf_antibot_effective_redirect "/challenge/verify";`) {
		t.Fatalf("expected prefix cookie challenge override, got: %s", siteConf)
	}

	locationsConf := byPath["nginx/easy-locations/site-a.conf"]
	if !strings.Contains(locationsConf, "location = /challenge/stage1 {") || !strings.Contains(locationsConf, "location = /challenge/stage1/verify {") {
		t.Fatalf("expected stage1 antibot locations, got: %s", locationsConf)
	}
}
