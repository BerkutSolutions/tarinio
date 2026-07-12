package compiler

import (
	"strings"
	"testing"
)

func TestEasySecurityMode_TransparentAndMonitorDisableActiveProtection(t *testing.T) {
	modes := []string{"transparent", "monitor"}
	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			site := SiteInput{
				ID:                "mode-site-" + mode,
				Enabled:           true,
				PrimaryHost:       mode + ".example.com",
				ListenHTTP:        true,
				DefaultUpstreamID: "up-" + mode,
			}
			profile := EasyProfileInput{
				SiteID:                    site.ID,
				SecurityMode:              mode,
				AllowedMethods:            []string{"GET", "POST"},
				MaxClientSize:             "100m",
				UseModSecurity:            true,
				UseModSecurityCRSPlugins:  true,
				UseLimitConn:              true,
				LimitConnMaxHTTP1:         120,
				UseLimitReq:               true,
				LimitReqRate:              "40r/s",
				UseBadBehavior:            true,
				BadBehaviorStatusCodes:    []int{429, 444},
				BadBehaviorBanTimeSeconds: 180,
				AntibotChallenge:          "javascript",
				AntibotURI:                "/challenge",
				AntibotScannerAutoBan:     true,
				BlacklistIP:               []string{"198.51.100.10"},
				BlacklistUserAgent:        []string{"curl"},
				BlacklistURI:              []string{"/wp-admin"},
				BlacklistCountry:          []string{"RU"},
				WhitelistCountry:          []string{"US"},
				UseAuthBasic:              true,
				AuthUsers: []ServiceAuthUserInput{
					{Username: "admin", Password: "secret", Enabled: true},
				},
				CustomLimitRules: []CustomRateLimitRuleInput{
					{Path: "/limited/", Rate: "10r/s"},
				},
			}

			artifacts, err := RenderEasyArtifacts([]SiteInput{site}, []EasyProfileInput{profile})
			if err != nil {
				t.Fatalf("render easy artifacts: %v", err)
			}
			byPath := mapArtifactsByPath(artifacts)
			siteConf := byPath["nginx/easy/"+site.ID+".conf"]
			if siteConf == "" {
				t.Fatalf("missing site config for %s", site.ID)
			}

			if mode == "transparent" {
				if strings.Contains(siteConf, "modsecurity_rules_file /etc/waf/modsecurity/easy/") {
					t.Fatalf("transparent mode must not attach easy modsecurity rules, got: %s", siteConf)
				}
				if _, ok := byPath["modsecurity/easy/"+site.ID+".conf"]; ok {
					t.Fatalf("transparent mode must not emit modsecurity easy artifact")
				}
			} else {
				modsec := byPath["modsecurity/easy/"+site.ID+".conf"]
				if modsec == "" {
					t.Fatalf("monitor mode must emit modsecurity easy artifact")
				}
				if !strings.Contains(modsec, "SecRuleEngine DetectionOnly") {
					t.Fatalf("monitor mode must keep modsecurity in detection-only mode, got: %s", modsec)
				}
				if !strings.Contains(siteConf, "modsecurity_rules_file /etc/waf/modsecurity/easy/"+site.ID+".conf;") {
					t.Fatalf("monitor mode must attach easy modsecurity rules, got: %s", siteConf)
				}
			}

			if strings.Contains(siteConf, "if ($waf_antibot_guard") {
				t.Fatalf("%s mode must disable antibot blocking, got: %s", mode, siteConf)
			}
			if strings.Contains(siteConf, "deny 198.51.100.10;") {
				t.Fatalf("%s mode must disable blacklist deny, got: %s", mode, siteConf)
			}
			if strings.Contains(siteConf, "if ($waf_blacklist_ua_guard") || strings.Contains(siteConf, "if ($waf_blacklist_uri_guard") {
				t.Fatalf("%s mode must disable blacklist guards, got: %s", mode, siteConf)
			}
			if strings.Contains(siteConf, "return 302 /auth") {
				t.Fatalf("%s mode must disable auth gate enforcement, got: %s", mode, siteConf)
			}
			if mode == "transparent" {
				if strings.Contains(siteConf, "modsecurity on;") || strings.Contains(siteConf, "modsecurity_rules_file") {
					t.Fatalf("transparent mode must not attach modsecurity directives, got: %s", siteConf)
				}
			}
			if !strings.Contains(siteConf, "set $waf_rate_limited_max_age 0;") {
				t.Fatalf("%s mode must disable bad-behavior ban gate, got: %s", mode, siteConf)
			}
			if _, ok := byPath["l4guard/config.json"]; ok {
				t.Fatalf("%s mode must disable l4 protection config", mode)
			}

			rateArtifacts, err := RenderEasyRateLimitArtifacts(
				[]SiteInput{site},
				[]UpstreamInput{{ID: site.DefaultUpstreamID, SiteID: site.ID, Scheme: "http", Host: "127.0.0.1", Port: 8080}},
				[]EasyProfileInput{profile},
			)
			if err != nil {
				t.Fatalf("render easy rate-limit artifacts: %v", err)
			}
			rateByPath := mapArtifactsByPath(rateArtifacts)
			rateConf := rateByPath["nginx/conf.d/easy-ratelimits.conf"]
			if strings.TrimSpace(rateConf) != "" {
				t.Fatalf("%s mode must not emit limit_req zones from custom rules, got: %s", mode, rateConf)
			}
		})
	}
}

func TestEasySecurityMode_BlockKeepsEnabledProtectionModules(t *testing.T) {
	site := SiteInput{
		ID:                "mode-site-block",
		Enabled:           true,
		PrimaryHost:       "block.example.com",
		ListenHTTP:        true,
		DefaultUpstreamID: "up-block",
	}
	profile := EasyProfileInput{
		SiteID:                    site.ID,
		SecurityMode:              "block",
		AllowedMethods:            []string{"GET", "POST"},
		MaxClientSize:             "100m",
		UseModSecurity:            true,
		UseModSecurityCRSPlugins:  true,
		UseLimitConn:              true,
		LimitConnMaxHTTP1:         120,
		UseLimitReq:               true,
		LimitReqRate:              "40r/s",
		UseBadBehavior:            true,
		BadBehaviorStatusCodes:    []int{429, 444},
		BadBehaviorBanTimeSeconds: 180,
		AntibotChallenge:          "javascript",
		AntibotURI:                "/challenge",
		AntibotScannerAutoBan:     true,
		BlacklistIP:               []string{"198.51.100.10"},
		UseAuthBasic:              true,
		AuthUsers: []ServiceAuthUserInput{
			{Username: "admin", Password: "secret", Enabled: true},
		},
		CustomLimitRules: []CustomRateLimitRuleInput{
			{Path: "/limited/", Rate: "10r/s"},
		},
	}

	artifacts, err := RenderEasyArtifacts([]SiteInput{site}, []EasyProfileInput{profile})
	if err != nil {
		t.Fatalf("render easy artifacts: %v", err)
	}
	byPath := mapArtifactsByPath(artifacts)
	siteConf := byPath["nginx/easy/"+site.ID+".conf"]
	modsec := byPath["modsecurity/easy/"+site.ID+".conf"]
	if !strings.Contains(modsec, "SecRuleEngine On") {
		t.Fatalf("block mode must keep modsecurity blocking, got: %s", modsec)
	}
	if !strings.Contains(siteConf, "if ($waf_antibot_guard") {
		t.Fatalf("block mode must keep antibot guard, got: %s", siteConf)
	}
	if !strings.Contains(siteConf, "deny 198.51.100.10;") {
		t.Fatalf("block mode must keep blacklist deny, got: %s", siteConf)
	}
	if !strings.Contains(siteConf, "return 302 /auth") {
		t.Fatalf("block mode must keep auth gate enforcement, got: %s", siteConf)
	}
	if !strings.Contains(siteConf, "set $waf_rate_limited_max_age 180;") {
		t.Fatalf("block mode must keep bad-behavior ban gate, got: %s", siteConf)
	}
	if _, ok := byPath["l4guard/config.json"]; !ok {
		t.Fatal("block mode must keep l4 protection config when limits are enabled")
	}

	rateArtifacts, err := RenderEasyRateLimitArtifacts(
		[]SiteInput{site},
		[]UpstreamInput{{ID: site.DefaultUpstreamID, SiteID: site.ID, Scheme: "http", Host: "127.0.0.1", Port: 8080}},
		[]EasyProfileInput{profile},
	)
	if err != nil {
		t.Fatalf("render easy rate-limit artifacts: %v", err)
	}
	rateByPath := mapArtifactsByPath(rateArtifacts)
	rateConf := rateByPath["nginx/conf.d/easy-ratelimits.conf"]
	if !strings.Contains(rateConf, "limit_req_zone") {
		t.Fatalf("block mode must keep custom limit zones, got: %s", rateConf)
	}
}

func mapArtifactsByPath(items []ArtifactOutput) map[string]string {
	out := make(map[string]string, len(items))
	for _, item := range items {
		out[item.Path] = string(item.Content)
	}
	return out
}
