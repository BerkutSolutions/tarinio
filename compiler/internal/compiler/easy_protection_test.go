package compiler

import (
	"strings"
	"testing"
)

func TestRenderEasyArtifacts_ProtectionModesPerSite(t *testing.T) {
	artifacts, err := RenderEasyArtifacts(
		[]SiteInput{
			{ID: "site-block", Enabled: true, PrimaryHost: "block.example.com", ListenHTTP: true},
			{ID: "site-monitor", Enabled: true, PrimaryHost: "monitor.example.com", ListenHTTP: true},
			{ID: "site-transparent", Enabled: true, PrimaryHost: "transparent.example.com", ListenHTTP: true},
			{ID: "site-disabled", Enabled: true, PrimaryHost: "disabled.example.com", ListenHTTP: true},
		},
		[]EasyProfileInput{
			{
				SiteID:                   "site-block",
				SecurityMode:             "block",
				UseModSecurity:           true,
				UseModSecurityCRSPlugins: true,
				ModSecurityCRSVersion:    "4",
				AllowedMethods:           []string{"GET"},
				MaxClientSize:            "100m",
				UseLimitConn:             true,
				LimitConnMaxHTTP1:        50,
				UseLimitReq:              true,
				LimitReqRate:             "30r/s",
			},
			{
				SiteID:                   "site-monitor",
				SecurityMode:             "monitor",
				UseModSecurity:           true,
				UseModSecurityCRSPlugins: true,
				ModSecurityCRSVersion:    "4",
				AllowedMethods:           []string{"GET"},
				MaxClientSize:            "100m",
				UseLimitConn:             true,
				LimitConnMaxHTTP1:        50,
				UseLimitReq:              true,
				LimitReqRate:             "30r/s",
			},
			{
				SiteID:                   "site-transparent",
				SecurityMode:             "transparent",
				UseModSecurity:           true,
				UseModSecurityCRSPlugins: true,
				ModSecurityCRSVersion:    "4",
				AllowedMethods:           []string{"GET"},
				MaxClientSize:            "100m",
				UseLimitConn:             true,
				LimitConnMaxHTTP1:        50,
				UseLimitReq:              true,
				LimitReqRate:             "30r/s",
			},
			{
				SiteID:                   "site-disabled",
				SecurityMode:             "block",
				UseModSecurity:           false,
				UseModSecurityCRSPlugins: true,
				ModSecurityCRSVersion:    "4",
				AllowedMethods:           []string{"GET"},
				MaxClientSize:            "100m",
				UseLimitConn:             true,
				LimitConnMaxHTTP1:        50,
				UseLimitReq:              true,
				LimitReqRate:             "30r/s",
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

	blockRules := byPath["modsecurity/easy/site-block.conf"]
	if !strings.Contains(blockRules, "SecRuleEngine On") {
		t.Fatalf("expected block mode to enable engine, got: %s", blockRules)
	}
	if !strings.Contains(blockRules, "Include /etc/waf/modsecurity/coreruleset/rules/*.conf") {
		t.Fatalf("expected block mode to include CRS, got: %s", blockRules)
	}

	monitorRules := byPath["modsecurity/easy/site-monitor.conf"]
	if !strings.Contains(monitorRules, "SecRuleEngine DetectionOnly") {
		t.Fatalf("expected monitor mode to use DetectionOnly, got: %s", monitorRules)
	}
	if !strings.Contains(monitorRules, "Include /etc/waf/modsecurity/coreruleset/rules/*.conf") {
		t.Fatalf("expected monitor mode to include CRS, got: %s", monitorRules)
	}

	transparentRules := byPath["modsecurity/easy/site-transparent.conf"]
	if !strings.Contains(transparentRules, "SecRuleEngine Off") {
		t.Fatalf("expected transparent mode to disable engine, got: %s", transparentRules)
	}
	if strings.Contains(transparentRules, "Include /etc/waf/modsecurity/coreruleset/rules/*.conf") {
		t.Fatalf("expected transparent mode to skip CRS include, got: %s", transparentRules)
	}

	disabledNginx := byPath["nginx/easy/site-disabled.conf"]
	if !strings.Contains(disabledNginx, "modsecurity off;") {
		t.Fatalf("expected per-site modsecurity disable directive, got: %s", disabledNginx)
	}
	if _, ok := byPath["modsecurity/easy/site-disabled.conf"]; ok {
		t.Fatal("did not expect modsecurity/easy artifact when use_modsecurity=false")
	}
}

