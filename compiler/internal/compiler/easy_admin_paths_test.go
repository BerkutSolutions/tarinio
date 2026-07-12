package compiler

import (
	"regexp"
	"strings"
	"testing"
)

func TestEasyAdminBypassPathPatternForSite_Localhost(t *testing.T) {
	t.Setenv("CONTROL_PLANE_DEV_FAST_START_MANAGEMENT_SITE_ID", "control-plane-access")
	pattern := easyAdminBypassPathPatternForSite(SiteInput{ID: "localhost", PrimaryHost: "localhost"})
	if pattern != "^$" {
		t.Fatalf("expected localhost to stay a regular site unless configured explicitly, got %q", pattern)
	}
}

func TestEasyModSecurityBypassPathPatternForSite_Localhost(t *testing.T) {
	t.Setenv("CONTROL_PLANE_DEV_FAST_START_MANAGEMENT_SITE_ID", "control-plane-access")
	pattern := easyModSecurityBypassPathPatternForSite(SiteInput{ID: "localhost", PrimaryHost: "localhost"})
	if pattern != "" {
		t.Fatalf("expected localhost to skip ModSecurity bypass unless configured explicitly, got %q", pattern)
	}
}

func TestEasyAdminBypassPathPatternForSite_ConfiguredManagementSiteID(t *testing.T) {
	t.Setenv("CONTROL_PLANE_DEV_FAST_START_MANAGEMENT_SITE_ID", "localhost")
	pattern := easyAdminBypassPathPatternForSite(SiteInput{ID: "localhost", PrimaryHost: "localhost"})
	if pattern == "^$" {
		t.Fatalf("expected localhost to be treated as management site when configured explicitly")
	}
}

func TestEasyModSecurityBypassPathPatternForSite_ConfiguredManagementSiteID(t *testing.T) {
	t.Setenv("CONTROL_PLANE_DEV_FAST_START_MANAGEMENT_SITE_ID", "localhost")
	pattern := easyModSecurityBypassPathPatternForSite(SiteInput{ID: "localhost", PrimaryHost: "localhost"})
	if pattern == "" {
		t.Fatalf("expected localhost ModSecurity bypass when configured explicitly as management site")
	}
}

func TestEasyModSecurityBypassPathPatternForSite_MatchesAppPingOnlyForManagementSite(t *testing.T) {
	managementPattern := easyModSecurityBypassPathPatternForSite(SiteInput{ID: "management-site", PrimaryHost: "ui"})
	if !regexp.MustCompile(managementPattern).MatchString("/api/app/ping") {
		t.Fatalf("expected management ModSecurity bypass to match /api/app/ping, got %q", managementPattern)
	}
	if ordinaryPattern := easyModSecurityBypassPathPatternForSite(SiteInput{ID: "site-a", PrimaryHost: "app.example.com"}); ordinaryPattern != "" {
		t.Fatalf("expected ordinary site to have no ModSecurity bypass, got %q", ordinaryPattern)
	}
}

func TestEasyAdminBypassPathPatternForSite_UIProxyPrimaryHost(t *testing.T) {
	pattern := easyAdminBypassPathPatternForSite(SiteInput{ID: "site-a", PrimaryHost: "ui"})
	if pattern == "^$" {
		t.Fatalf("expected UI proxy primary host to be treated as management site")
	}
}

func TestEasyAdminBypassPathPatternForSite_MatchesManagementMutationEndpoints(t *testing.T) {
	pattern := easyAdminBypassPathPatternForSite(SiteInput{ID: "management-site", PrimaryHost: "ui"})
	if pattern == "^$" {
		t.Fatal("expected management site bypass pattern")
	}
	re := regexp.MustCompile(pattern)
	for _, path := range []string{"/api/sites/service-1", "/api/access-policies/policy-1", "/services/service-1", "/dashboard"} {
		if !re.MatchString(path) {
			t.Fatalf("expected management bypass pattern to match %q, got %q", path, pattern)
		}
	}
	for _, path := range []string{"/login", "/login/2fa"} {
		if !re.MatchString(path) {
			t.Fatalf("expected management bypass pattern to protect %q, got %q", path, pattern)
		}
	}
	for _, path := range []string{"/apiary", "/service-worker.js", "/foo/bar"} {
		if re.MatchString(path) {
			t.Fatalf("expected management bypass pattern not to match %q, got %q", path, pattern)
		}
	}
}

func TestEasyAdminAntibotExclusionRulesForSite_ManagementSite(t *testing.T) {
	t.Setenv("CONTROL_PLANE_DEV_FAST_START_MANAGEMENT_SITE_ID", "control-plane-access")
	rules := easyAdminAntibotExclusionRulesForSite(SiteInput{ID: "control-plane-access", PrimaryHost: "ui"})
	if len(rules) == 0 {
		t.Fatal("expected management site antibot exclusions")
	}
	byPath := map[string][]string{}
	for _, rule := range rules {
		byPath[rule.Path] = append([]string(nil), rule.Methods...)
	}
	for _, path := range []string{"/", "/api/", "/static/", "/services", "/dashboard", "/auth", "/auth/verify"} {
		methods, ok := byPath[path]
		if !ok {
			t.Fatalf("expected antibot exclusion for %s, got %#v", path, byPath)
		}
		if len(methods) != 2 || methods[0] != "GET" || methods[1] != "HEAD" {
			t.Fatalf("expected GET/HEAD methods for %s, got %#v", path, methods)
		}
	}
	if extra := easyAdminAntibotExclusionRulesForSite(SiteInput{ID: "site-a", PrimaryHost: "example.com"}); len(extra) != 0 {
		t.Fatalf("expected no management exclusions for regular site, got %#v", extra)
	}
}

func TestEasyAdminAuthExclusionRulesForSite_ManagementSite(t *testing.T) {
	t.Setenv("CONTROL_PLANE_DEV_FAST_START_MANAGEMENT_SITE_ID", "control-plane-access")
	rules := easyAdminAuthExclusionRulesForSite(SiteInput{ID: "control-plane-access", PrimaryHost: "ui"})
	if len(rules) == 0 {
		t.Fatal("expected management site auth exclusions")
	}
	byPath := map[string][]string{}
	for _, rule := range rules {
		byPath[rule.Path] = append([]string(nil), rule.Methods...)
	}
	for _, path := range []string{"/api/", "/services", "/healthcheck"} {
		methods, ok := byPath[path]
		if !ok {
			t.Fatalf("expected rule for %s", path)
		}
		joined := strings.Join(methods, ",")
		if joined != "GET,HEAD,POST,PUT,PATCH,DELETE" {
			t.Fatalf("expected write-capable methods for %s, got %#v", path, methods)
		}
	}
}

func TestAppendAntibotExclusionRules_DeduplicatesAndKeepsExistingRules(t *testing.T) {
	base := []AntibotExclusionRuleInput{{Path: "/static/", Methods: []string{"HEAD", "GET"}}, {Path: "/custom", Methods: []string{"POST"}}}
	extra := []AntibotExclusionRuleInput{{Path: "/static/", Methods: []string{"GET", "HEAD"}}, {Path: "/services", Methods: []string{"GET", "HEAD"}}}
	merged := appendAntibotExclusionRules(base, extra)
	if len(merged) != 3 {
		t.Fatalf("expected deduplicated merge, got %#v", merged)
	}
	joined := make([]string, 0, len(merged))
	for _, rule := range merged {
		joined = append(joined, rule.Path+":"+strings.Join(rule.Methods, ","))
	}
	actual := strings.Join(joined, "|")
	for _, expected := range []string{"/static/:GET,HEAD", "/services:GET,HEAD", "/custom:POST"} {
		if !strings.Contains(actual, expected) {
			t.Fatalf("expected merged exclusions to contain %s, got %s", expected, actual)
		}
	}
}
