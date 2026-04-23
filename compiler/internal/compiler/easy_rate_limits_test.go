package compiler

import (
	"strings"
	"testing"
)

func TestRenderEasyRateLimitArtifacts_GeneratesRouteSpecificArtifacts(t *testing.T) {
	artifacts, err := RenderEasyRateLimitArtifacts(
		[]SiteInput{{
			ID:                "control-plane-access",
			Enabled:           true,
			PrimaryHost:       "waf.example.com",
			ListenHTTP:        true,
			DefaultUpstreamID: "control-plane-access-upstream",
		}},
		[]UpstreamInput{{
			ID:             "control-plane-access-upstream",
			SiteID:         "control-plane-access",
			Name:           "control-plane-access-upstream",
			Scheme:         "http",
			Host:           "ui",
			Port:           80,
			BasePath:       "/",
			PassHostHeader: true,
		}},
		[]EasyProfileInput{{
			SiteID:           "control-plane-access",
			AntibotChallenge: "turnstile",
			AntibotURI:       "/challenge",
			CustomLimitRules: []CustomRateLimitRuleInput{
				{Path: "/login", Rate: "6r/s"},
				{Path: "/api/auth/", Rate: "12r/s"},
			},
		}},
	)
	if err != nil {
		t.Fatalf("render easy rate limit artifacts: %v", err)
	}

	byPath := map[string]string{}
	for _, item := range artifacts {
		byPath[item.Path] = string(item.Content)
	}

	httpConf := byPath["nginx/conf.d/easy-ratelimits.conf"]
	if !strings.Contains(httpConf, "limit_req_zone $waf_rate_limit_key_control_plane_access zone=easy_control_plane_access_req_v2_0:10m rate=6r/s;") {
		t.Fatalf("expected exact login zone in easy rate limit http conf, got: %s", httpConf)
	}
	if !strings.Contains(httpConf, "limit_req_zone $waf_rate_limit_key_control_plane_access zone=easy_control_plane_access_req_v2_1:10m rate=12r/s;") {
		t.Fatalf("expected prefix api auth zone in easy rate limit http conf, got: %s", httpConf)
	}

	locationsConf := byPath["nginx/easy-locations/control-plane-access.conf"]
	if !strings.Contains(locationsConf, "location = /challenge") || !strings.Contains(locationsConf, "Set-Cookie \"waf_antibot_") {
		t.Fatalf("expected antibot challenge location in easy locations conf, got: %s", locationsConf)
	}
	if !strings.Contains(locationsConf, "location = /login {") {
		t.Fatalf("expected exact location for /login, got: %s", locationsConf)
	}
	if !strings.Contains(locationsConf, "location ^~ /api/auth/ {") {
		t.Fatalf("expected prefix location for /api/auth/, got: %s", locationsConf)
	}
	if !strings.Contains(locationsConf, "limit_req zone=easy_control_plane_access_req_v2_0 burst=6 nodelay;") {
		t.Fatalf("expected exact route limit_req directive, got: %s", locationsConf)
	}
	if !strings.Contains(locationsConf, "limit_req_status 429;") {
		t.Fatalf("expected custom route limit_req_status directive, got: %s", locationsConf)
	}
	if !strings.Contains(locationsConf, "limit_req zone=easy_control_plane_access_req_v2_1 burst=12 nodelay;") {
		t.Fatalf("expected prefix route limit_req directive, got: %s", locationsConf)
	}
	if !strings.Contains(locationsConf, "proxy_set_header Host $http_host;") {
		t.Fatalf("expected host header pass-through for route-specific proxying, got: %s", locationsConf)
	}
}

func TestRenderEasyRateLimitArtifacts_SkipsReservedBaseLocations(t *testing.T) {
	artifacts, err := RenderEasyRateLimitArtifacts(
		[]SiteInput{{
			ID:                "control-plane-access",
			Enabled:           true,
			PrimaryHost:       "waf.example.com",
			ListenHTTP:        true,
			DefaultUpstreamID: "control-plane-access-upstream",
		}},
		[]UpstreamInput{{
			ID:             "control-plane-access-upstream",
			SiteID:         "control-plane-access",
			Name:           "control-plane-access-upstream",
			Scheme:         "http",
			Host:           "ui",
			Port:           80,
			BasePath:       "/",
			PassHostHeader: true,
		}},
		[]EasyProfileInput{{
			SiteID: "control-plane-access",
			CustomLimitRules: []CustomRateLimitRuleInput{
				{Path: "/api/", Rate: "30r/s"},
				{Path: "/static/", Rate: "30r/s"},
				{Path: "/dashboard", Rate: "20r/s"},
				{Path: "/", Rate: "50r/s"},
				{Path: "/api/auth/", Rate: "12r/s"},
			},
		}},
	)
	if err != nil {
		t.Fatalf("render easy rate limit artifacts: %v", err)
	}

	byPath := map[string]string{}
	for _, item := range artifacts {
		byPath[item.Path] = string(item.Content)
	}

	httpConf := byPath["nginx/conf.d/easy-ratelimits.conf"]
	if strings.Contains(httpConf, "rate=30r/s") || strings.Contains(httpConf, "rate=20r/s") || strings.Contains(httpConf, "rate=50r/s") {
		t.Fatalf("expected reserved admin paths to be skipped, got: %s", httpConf)
	}
	if !strings.Contains(httpConf, "rate=12r/s") {
		t.Fatalf("expected non-reserved route rate to remain, got: %s", httpConf)
	}

	locationsConf := byPath["nginx/easy-locations/control-plane-access.conf"]
	if strings.Contains(locationsConf, "location ^~ /api/ {") || strings.Contains(locationsConf, "location ^~ /static/ {") || strings.Contains(locationsConf, "location = /dashboard {") || strings.Contains(locationsConf, "location ^~ / {") || strings.Contains(locationsConf, "location = / {") {
		t.Fatalf("expected reserved locations to be skipped, got: %s", locationsConf)
	}
	if !strings.Contains(locationsConf, "location ^~ /api/auth/ {") {
		t.Fatalf("expected non-reserved location to remain, got: %s", locationsConf)
	}
}

func TestRenderEasyRateLimitArtifacts_WildcardPathSupportsSentryEnvelope(t *testing.T) {
	artifacts, err := RenderEasyRateLimitArtifacts(
		[]SiteInput{{
			ID:                "sentry.example.com",
			Enabled:           true,
			PrimaryHost:       "sentry.example.com",
			ListenHTTP:        true,
			DefaultUpstreamID: "sentry-upstream",
		}},
		[]UpstreamInput{{
			ID:             "sentry-upstream",
			SiteID:         "sentry.example.com",
			Name:           "sentry-upstream",
			Scheme:         "http",
			Host:           "upstream",
			Port:           9000,
			BasePath:       "/",
			PassHostHeader: true,
		}},
		[]EasyProfileInput{{
			SiteID: "sentry.example.com",
			CustomLimitRules: []CustomRateLimitRuleInput{
				{Path: "/api/2/envelope/", Rate: "30r/s"},
			},
		}},
	)
	if err != nil {
		t.Fatalf("render easy rate limit artifacts: %v", err)
	}

	byPath := map[string]string{}
	for _, item := range artifacts {
		byPath[item.Path] = string(item.Content)
	}

	httpConf := byPath["nginx/conf.d/easy-ratelimits.conf"]
	if !strings.Contains(httpConf, "rate=30r/s;") {
		t.Fatalf("expected sentry envelope rate in easy rate limit http conf, got: %s", httpConf)
	}

	locationsConf := byPath["nginx/easy-locations/sentry.example.com.conf"]
	if !strings.Contains(locationsConf, "location ~* ^/api/.*/envelope/ {") {
		t.Fatalf("expected wildcard location for sentry envelope path, got: %s", locationsConf)
	}
	if !strings.Contains(locationsConf, "proxy_intercept_errors off;") {
		t.Fatalf("expected api wildcard location to disable proxy intercept errors, got: %s", locationsConf)
	}
	if !strings.Contains(locationsConf, "limit_req_status 429;") {
		t.Fatalf("expected wildcard custom location to return 429 on limit, got: %s", locationsConf)
	}
}

func TestRenderEasyRateLimitArtifacts_CustomStatusFromBadBehaviorCodes(t *testing.T) {
	artifacts, err := RenderEasyRateLimitArtifacts(
		[]SiteInput{{
			ID:                "site-a",
			Enabled:           true,
			PrimaryHost:       "a.example.com",
			ListenHTTP:        true,
			DefaultUpstreamID: "upstream-a",
		}},
		[]UpstreamInput{{
			ID:             "upstream-a",
			SiteID:         "site-a",
			Name:           "upstream-a",
			Scheme:         "http",
			Host:           "upstream",
			Port:           8080,
			BasePath:       "/",
			PassHostHeader: true,
		}},
		[]EasyProfileInput{{
			SiteID:                 "site-a",
			BadBehaviorStatusCodes: []int{403, 444},
			CustomLimitRules: []CustomRateLimitRuleInput{
				{Path: "/login", Rate: "5r/s"},
			},
		}},
	)
	if err != nil {
		t.Fatalf("render easy rate limit artifacts: %v", err)
	}

	byPath := map[string]string{}
	for _, item := range artifacts {
		byPath[item.Path] = string(item.Content)
	}
	locationsConf := byPath["nginx/easy-locations/site-a.conf"]
	if !strings.Contains(locationsConf, "limit_req_status 403;") {
		t.Fatalf("expected custom location to use status from bad_behavior_status_codes, got: %s", locationsConf)
	}
}

func TestRenderEasyRateLimitArtifacts_CustomStatusSupportsNon429Codes(t *testing.T) {
	artifacts, err := RenderEasyRateLimitArtifacts(
		[]SiteInput{{
			ID:                "site-b",
			Enabled:           true,
			PrimaryHost:       "b.example.com",
			ListenHTTP:        true,
			DefaultUpstreamID: "upstream-b",
		}},
		[]UpstreamInput{{
			ID:             "upstream-b",
			SiteID:         "site-b",
			Name:           "upstream-b",
			Scheme:         "http",
			Host:           "upstream",
			Port:           8081,
			BasePath:       "/",
			PassHostHeader: true,
		}},
		[]EasyProfileInput{{
			SiteID:                 "site-b",
			BadBehaviorStatusCodes: []int{451},
			CustomLimitRules: []CustomRateLimitRuleInput{
				{Path: "/login", Rate: "5r/s"},
			},
		}},
	)
	if err != nil {
		t.Fatalf("render easy rate limit artifacts: %v", err)
	}

	byPath := map[string]string{}
	for _, item := range artifacts {
		byPath[item.Path] = string(item.Content)
	}
	locationsConf := byPath["nginx/easy-locations/site-b.conf"]
	if !strings.Contains(locationsConf, "limit_req_status 451;") {
		t.Fatalf("expected custom location to use configured non-429 status code, got: %s", locationsConf)
	}
}
