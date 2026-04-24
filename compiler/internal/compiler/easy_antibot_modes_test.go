package compiler

import (
	"strings"
	"testing"
)

func TestRenderEasyArtifacts_AntiBotModesMatrix(t *testing.T) {
	modes := []struct {
		name             string
		mode             string
		enabled          bool
		usesInterstitial bool
	}{
		{name: "no", mode: "no", enabled: false, usesInterstitial: false},
		{name: "cookie", mode: "cookie", enabled: true, usesInterstitial: false},
		{name: "javascript", mode: "javascript", enabled: true, usesInterstitial: true},
		{name: "captcha", mode: "captcha", enabled: true, usesInterstitial: true},
		{name: "recaptcha", mode: "recaptcha", enabled: true, usesInterstitial: true},
		{name: "hcaptcha", mode: "hcaptcha", enabled: true, usesInterstitial: true},
		{name: "turnstile", mode: "turnstile", enabled: true, usesInterstitial: true},
		{name: "mcaptcha", mode: "mcaptcha", enabled: true, usesInterstitial: true},
	}

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

	for _, tc := range modes {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			profile := EasyProfileInput{
				SiteID:                 "site-a",
				SecurityMode:           "block",
				AllowedMethods:         []string{"GET", "POST"},
				MaxClientSize:          "50m",
				UseModSecurity:         true,
				UseLimitConn:           true,
				LimitConnMaxHTTP1:      200,
				UseLimitReq:            true,
				LimitReqRate:           "100r/s",
				AntibotChallenge:       tc.mode,
				AntibotURI:             "/challenge",
				AntibotRecaptchaKey:    "recaptcha-site-key",
				AntibotHcaptchaKey:     "hcaptcha-site-key",
				AntibotTurnstileKey:    "turnstile-site-key",
				CORSAllowedOrigins:     []string{"*"},
				PassHostHeader:         true,
				SendXForwardedFor:      true,
				SendXForwardedProto:    true,
				SendXRealIP:            false,
				UseBadBehavior:         true,
				BadBehaviorStatusCodes: []int{429},
			}

			easyArtifacts, err := RenderEasyArtifacts([]SiteInput{site}, []EasyProfileInput{profile})
			if err != nil {
				t.Fatalf("render easy artifacts for mode %s: %v", tc.mode, err)
			}
			rateArtifacts, err := RenderEasyRateLimitArtifacts([]SiteInput{site}, []UpstreamInput{upstream}, []EasyProfileInput{profile})
			if err != nil {
				t.Fatalf("render easy rate-limit artifacts for mode %s: %v", tc.mode, err)
			}

			byPath := map[string]string{}
			for _, item := range append(easyArtifacts, rateArtifacts...) {
				byPath[item.Path] = string(item.Content)
			}

			siteConf := byPath["nginx/easy/site-a.conf"]
			if siteConf == "" {
				t.Fatalf("missing nginx/easy/site-a.conf for mode %s", tc.mode)
			}

			if tc.enabled {
				if !strings.Contains(siteConf, "if ($cookie_waf_antibot_") {
					t.Fatalf("expected antibot cookie guard for mode %s, got: %s", tc.mode, siteConf)
				}
				if !strings.Contains(siteConf, `add_header X-WAF-Antibot-Mode "$waf_antibot_effective_challenge" always;`) {
					t.Fatalf("expected antibot mode header for mode %s, got: %s", tc.mode, siteConf)
				}
				if tc.usesInterstitial {
					if !strings.Contains(siteConf, `set $waf_antibot_effective_redirect "/challenge";`) {
						t.Fatalf("expected interstitial redirect target for mode %s, got: %s", tc.mode, siteConf)
					}
				} else {
					if !strings.Contains(siteConf, `set $waf_antibot_effective_redirect "/challenge/verify";`) {
						t.Fatalf("expected direct verify redirect target for mode %s, got: %s", tc.mode, siteConf)
					}
				}
			} else {
				if strings.Contains(siteConf, "X-WAF-Antibot-Mode") || strings.Contains(siteConf, "waf_antibot_guard") {
					t.Fatalf("did not expect antibot directives for mode %s, got: %s", tc.mode, siteConf)
				}
			}

			locationsConf := byPath["nginx/easy-locations/site-a.conf"]
			if tc.enabled {
				if !strings.Contains(locationsConf, "location = /challenge {") || !strings.Contains(locationsConf, "location = /challenge/verify {") {
					t.Fatalf("expected challenge + verify locations for mode %s, got: %s", tc.mode, locationsConf)
				}
				if tc.usesInterstitial {
					if !strings.Contains(locationsConf, "alias /etc/waf/errors/site-a/antibot.html;") {
						t.Fatalf("expected interstitial alias for mode %s, got: %s", tc.mode, locationsConf)
					}
					if byPath["errors/site-a/antibot.html"] == "" {
						t.Fatalf("expected challenge page artifact for mode %s", tc.mode)
					}
				} else {
					if !strings.Contains(locationsConf, "return 204;") {
						t.Fatalf("expected cookie challenge 204 for mode %s, got: %s", tc.mode, locationsConf)
					}
					if _, ok := byPath["errors/site-a/antibot.html"]; ok {
						t.Fatalf("did not expect challenge page artifact for mode %s", tc.mode)
					}
				}
			} else {
				if strings.Contains(locationsConf, "location = /challenge") || strings.Contains(locationsConf, "waf_antibot_") {
					t.Fatalf("did not expect antibot locations for mode %s, got: %s", tc.mode, locationsConf)
				}
			}
		})
	}
}
