package compiler

import (
	"strings"
	"testing"
)

func TestHttpStrictParsing_EnabledDirectivesPresent(t *testing.T) {
	artifacts, err := RenderEasyArtifacts(
		[]SiteInput{{
			ID:          "site-strict",
			Enabled:     true,
			PrimaryHost: "strict.example.com",
			ListenHTTP:  true,
		}},
		[]EasyProfileInput{{
			SiteID:            "site-strict",
			SecurityMode:      "block",
			AllowedMethods:    []string{"GET", "POST"},
			MaxClientSize:     "10m",
			HttpStrictParsing: true,
			UseLimitConn:      true,
			LimitConnMaxHTTP1: 100,
			UseLimitReq:       true,
			LimitReqRate:      "50r/s",
		}},
	)
	if err != nil {
		t.Fatalf("render easy artifacts: %v", err)
	}

	var siteConf string
	for _, item := range artifacts {
		if item.Path == "nginx/easy/site-strict.conf" {
			siteConf = string(item.Content)
		}
	}
	if siteConf == "" {
		t.Fatal("expected easy site conf artifact for site-strict")
	}

	directives := []string{
		"ignore_invalid_headers on;",
		"underscores_in_headers off;",
		`proxy_set_header Transfer-Encoding "";`,
	}
	for _, directive := range directives {
		if !strings.Contains(siteConf, directive) {
			t.Errorf("expected directive %q in site conf when HttpStrictParsing=true, got:\n%s", directive, siteConf)
		}
	}
}

func TestHttpStrictParsing_DisabledDirectivesAbsent(t *testing.T) {
	artifacts, err := RenderEasyArtifacts(
		[]SiteInput{{
			ID:          "site-nostrict",
			Enabled:     true,
			PrimaryHost: "nostrict.example.com",
			ListenHTTP:  true,
		}},
		[]EasyProfileInput{{
			SiteID:            "site-nostrict",
			SecurityMode:      "block",
			AllowedMethods:    []string{"GET", "POST"},
			MaxClientSize:     "10m",
			HttpStrictParsing: false,
			UseLimitConn:      true,
			LimitConnMaxHTTP1: 100,
			UseLimitReq:       true,
			LimitReqRate:      "50r/s",
		}},
	)
	if err != nil {
		t.Fatalf("render easy artifacts: %v", err)
	}

	var siteConf string
	for _, item := range artifacts {
		if item.Path == "nginx/easy/site-nostrict.conf" {
			siteConf = string(item.Content)
		}
	}
	if siteConf == "" {
		t.Fatal("expected easy site conf artifact for site-nostrict")
	}

	directives := []string{
		"ignore_invalid_headers on;",
		"underscores_in_headers off;",
	}
	for _, directive := range directives {
		if strings.Contains(siteConf, directive) {
			t.Errorf("directive %q must NOT be present when HttpStrictParsing=false", directive)
		}
	}
}

func TestHttpStrictParsing_DefaultIsOff(t *testing.T) {
	// Verify that the default (zero-value) profile has HttpStrictParsing=false.
	profile := defaultEasyProfileForSite("site-default")
	if profile.HttpStrictParsing {
		t.Error("defaultEasyProfileForSite: HttpStrictParsing must be false by default")
	}
}
