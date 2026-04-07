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
			SiteID: "control-plane-access",
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
	if !strings.Contains(httpConf, "limit_req_zone $waf_rate_limit_key_control_plane_access zone=easy_control_plane_access_req_0:10m rate=6r/s;") {
		t.Fatalf("expected exact login zone in easy rate limit http conf, got: %s", httpConf)
	}
	if !strings.Contains(httpConf, "limit_req_zone $waf_rate_limit_key_control_plane_access zone=easy_control_plane_access_req_1:10m rate=12r/s;") {
		t.Fatalf("expected prefix api auth zone in easy rate limit http conf, got: %s", httpConf)
	}

	locationsConf := byPath["nginx/easy-locations/control-plane-access.conf"]
	if !strings.Contains(locationsConf, "location = /login {") {
		t.Fatalf("expected exact location for /login, got: %s", locationsConf)
	}
	if !strings.Contains(locationsConf, "location ^~ /api/auth/ {") {
		t.Fatalf("expected prefix location for /api/auth/, got: %s", locationsConf)
	}
	if !strings.Contains(locationsConf, "limit_req zone=easy_control_plane_access_req_0 burst=6 nodelay;") {
		t.Fatalf("expected exact route limit_req directive, got: %s", locationsConf)
	}
	if !strings.Contains(locationsConf, "limit_req zone=easy_control_plane_access_req_1 burst=12 nodelay;") {
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
	if strings.Contains(httpConf, "rate=30r/s") || strings.Contains(httpConf, "rate=50r/s") {
		t.Fatalf("expected reserved /api/ and / paths to be skipped, got: %s", httpConf)
	}
	if !strings.Contains(httpConf, "rate=12r/s") {
		t.Fatalf("expected non-reserved route rate to remain, got: %s", httpConf)
	}

	locationsConf := byPath["nginx/easy-locations/control-plane-access.conf"]
	if strings.Contains(locationsConf, "location ^~ /api/ {") || strings.Contains(locationsConf, "location ^~ / {") || strings.Contains(locationsConf, "location = / {") {
		t.Fatalf("expected reserved locations to be skipped, got: %s", locationsConf)
	}
	if !strings.Contains(locationsConf, "location ^~ /api/auth/ {") {
		t.Fatalf("expected non-reserved location to remain, got: %s", locationsConf)
	}
}
