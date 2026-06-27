package compiler

import (
	"strings"
	"testing"
)

func TestRenderEasyArtifacts_JA3Blacklist_InConfig(t *testing.T) {
	artifacts, err := RenderEasyArtifacts(
		[]SiteInput{
			{ID: "site-ja3", Enabled: true, PrimaryHost: "ja3.example.com", ListenHTTP: true},
		},
		[]EasyProfileInput{
			{
				SiteID:           "site-ja3",
				SecurityMode:     "block",
				UseModSecurity:   false,
				AllowedMethods:   []string{"GET"},
				MaxClientSize:    "100m",
				UseLimitConn:     true,
				LimitConnMaxHTTP1: 50,
				UseLimitReq:      true,
				LimitReqRate:     "30r/s",
				BlacklistJA3:     []string{"abc123deadbeef0000000000000000aa", "def456deadbeef0000000000000000bb"},
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
	if !strings.Contains(conf, "waf_ja3_blacklist") {
		t.Fatalf("expected waf_ja3_blacklist in nginx config, got:\n%s", conf)
	}
	if !strings.Contains(conf, "abc123deadbeef0000000000000000aa") {
		t.Fatalf("expected first JA3 fingerprint in config, got:\n%s", conf)
	}
	if !strings.Contains(conf, "def456deadbeef0000000000000000bb") {
		t.Fatalf("expected second JA3 fingerprint in config, got:\n%s", conf)
	}
	if !strings.Contains(conf, "waf_ja3_block_guard") {
		t.Fatalf("expected waf_ja3_block_guard in config, got:\n%s", conf)
	}
}

func TestRenderEasyArtifacts_JA3Blacklist_EmptyNoDirective(t *testing.T) {
	artifacts, err := RenderEasyArtifacts(
		[]SiteInput{
			{ID: "site-noja3", Enabled: true, PrimaryHost: "noja3.example.com", ListenHTTP: true},
		},
		[]EasyProfileInput{
			{
				SiteID:           "site-noja3",
				SecurityMode:     "block",
				UseModSecurity:   false,
				AllowedMethods:   []string{"GET"},
				MaxClientSize:    "100m",
				UseLimitConn:     true,
				LimitConnMaxHTTP1: 50,
				UseLimitReq:      true,
				LimitReqRate:     "30r/s",
				BlacklistJA3:     nil,
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
				SiteID:           "site-mon",
				SecurityMode:     "monitor",
				UseModSecurity:   false,
				AllowedMethods:   []string{"GET"},
				MaxClientSize:    "100m",
				BlacklistJA3:     []string{"abc123deadbeef0000000000000000aa"},
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
