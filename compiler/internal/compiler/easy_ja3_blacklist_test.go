package compiler

import (
	"strings"
	"testing"
)

// JA3 nginx blocking is disabled — the ngx_ssl_ja3 module requires a patched nginx
// that is incompatible with the debian modsecurity module. The blacklist field is
// preserved in the data model and UI, but no nginx directives are generated.

func TestRenderEasyArtifacts_JA3Blacklist_InConfig(t *testing.T) {
	artifacts, err := RenderEasyArtifacts(
		[]SiteInput{
			{ID: "site-ja3", Enabled: true, PrimaryHost: "ja3.example.com", ListenHTTP: true},
		},
		[]EasyProfileInput{
			{
				SiteID:            "site-ja3",
				SecurityMode:      "block",
				UseModSecurity:    false,
				AllowedMethods:    []string{"GET"},
				MaxClientSize:     "100m",
				UseLimitConn:      true,
				LimitConnMaxHTTP1: 50,
				UseLimitReq:       true,
				LimitReqRate:      "30r/s",
				BlacklistJA3:      []string{"abc123deadbeef0000000000000000aa", "def456deadbeef0000000000000000bb"},
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

	conf := byPath["nginx/easy/site-ja3.conf"]
	// JA3 nginx directives are intentionally absent — module not available in runtime image.
	// Verify the config still renders without error and does NOT reference unknown variables.
	if strings.Contains(conf, "ssl_ja3") {
		t.Fatalf("unexpected ssl_ja3 variable in nginx config (module not available): %s", conf)
	}
}

func TestRenderEasyArtifacts_JA3Blacklist_EmptyNoDirective(t *testing.T) {
	artifacts, err := RenderEasyArtifacts(
		[]SiteInput{
			{ID: "site-noja3", Enabled: true, PrimaryHost: "noja3.example.com", ListenHTTP: true},
		},
		[]EasyProfileInput{
			{
				SiteID:            "site-noja3",
				SecurityMode:      "block",
				UseModSecurity:    false,
				AllowedMethods:    []string{"GET"},
				MaxClientSize:     "100m",
				UseLimitConn:      true,
				LimitConnMaxHTTP1: 50,
				UseLimitReq:       true,
				LimitReqRate:      "30r/s",
				BlacklistJA3:      nil,
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

	conf := byPath["nginx/easy/site-noja3.conf"]
	if strings.Contains(conf, "waf_ja3_blacklist") {
		t.Fatalf("expected NO waf_ja3_blacklist directive when BlacklistJA3 is empty, got:\n%s", conf)
	}
}

func TestRenderEasyArtifacts_JA3Blacklist_ClearedInMonitorMode(t *testing.T) {
	artifacts, err := RenderEasyArtifacts(
		[]SiteInput{
			{ID: "site-mon", Enabled: true, PrimaryHost: "mon.example.com", ListenHTTP: true},
		},
		[]EasyProfileInput{
			{
				SiteID:         "site-mon",
				SecurityMode:   "monitor",
				UseModSecurity: false,
				AllowedMethods: []string{"GET"},
				MaxClientSize:  "100m",
				BlacklistJA3:   []string{"abc123deadbeef0000000000000000aa"},
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

	conf := byPath["nginx/easy/site-mon.conf"]
	if strings.Contains(conf, "waf_ja3_blacklist") {
		t.Fatalf("expected NO JA3 blacklist in monitor mode, got:\n%s", conf)
	}
}
