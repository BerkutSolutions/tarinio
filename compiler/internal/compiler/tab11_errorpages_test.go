package compiler

import (
	"strings"
	"testing"
)

// tab11_errorpages_test.go — тесты вкладки 11: Кастомные страницы ошибок
// Покрывает: UseCustomErrorPages=true включает proxy_intercept_errors и error_page,
// UseCustomErrorPages=false — директивы отсутствуют.

func TestErrorPages_Enabled_HasProxyInterceptAndErrorPages(t *testing.T) {
	conf := mustRenderSiteConf(t, "ep-site", EasyProfileInput{
		SiteID:              "ep-site",
		SecurityMode:        "block",
		AllowedMethods:      []string{"GET", "POST"},
		MaxClientSize:       "10m",
		UseCustomErrorPages: true,
	})

	if !strings.Contains(conf, "proxy_intercept_errors on") {
		t.Fatalf("expected proxy_intercept_errors on when UseCustomErrorPages=true, got:\n%s", conf)
	}
	if !strings.Contains(conf, "error_page 404") {
		t.Fatalf("expected error_page 404 directive when UseCustomErrorPages=true, got:\n%s", conf)
	}
	if !strings.Contains(conf, "error_page 500") {
		t.Fatalf("expected error_page 500 directive when UseCustomErrorPages=true, got:\n%s", conf)
	}
	if !strings.Contains(conf, "/__waf_errors/ep-site/404.html") {
		t.Fatalf("expected site-scoped error page path for 404, got:\n%s", conf)
	}
	if !strings.Contains(conf, "/__waf_errors/ep-site/451.html") {
		t.Fatalf("expected site-scoped legal page path for 451, got:\n%s", conf)
	}
}

func TestErrorPages_GeoBlockUsesInternalMarkerAndPublic403(t *testing.T) {
	conf := mustRenderSiteConf(t, "geo-site", EasyProfileInput{
		SiteID: "geo-site", SecurityMode: "block", AllowedMethods: []string{"GET"},
		MaxClientSize: "10m", UseCustomErrorPages: true, ShowGeoBlockPage: true,
		BlacklistCountry: []string{"RU"},
	})
	if !strings.Contains(conf, "return 599") || !strings.Contains(conf, "error_page 599 =403 /__waf_errors/_global/geo_block.html") {
		t.Fatalf("expected geo marker to use geo_block body with public 403, got:\n%s", conf)
	}
	if strings.Contains(conf, "return 451") {
		t.Fatalf("geo policy must not return legal-restriction status 451, got:\n%s", conf)
	}
}

func TestErrorPages_DoesNotAttemptToDeliver499(t *testing.T) {
	conf := mustRenderSiteConf(t, "closed-client", EasyProfileInput{
		SiteID: "closed-client", SecurityMode: "block", AllowedMethods: []string{"GET"}, MaxClientSize: "10m", UseCustomErrorPages: true,
	})
	if strings.Contains(conf, "error_page 499") {
		t.Fatalf("499 is diagnostic and must not produce an undeliverable response: %s", conf)
	}
}

func TestErrorPages_Disabled_NoProxyIntercept(t *testing.T) {
	conf := mustRenderSiteConf(t, "ep-site2", EasyProfileInput{
		SiteID:              "ep-site2",
		SecurityMode:        "block",
		AllowedMethods:      []string{"GET"},
		MaxClientSize:       "10m",
		UseCustomErrorPages: false,
	})

	if strings.Contains(conf, "proxy_intercept_errors on") {
		t.Fatalf("expected no proxy_intercept_errors when UseCustomErrorPages=false, got:\n%s", conf)
	}
	if strings.Contains(conf, "error_page 404") {
		t.Fatalf("expected no error_page directives when UseCustomErrorPages=false, got:\n%s", conf)
	}
}

func TestErrorPages_DisabledCodesAreRemovedWithoutDisablingOtherPages(t *testing.T) {
	conf := mustRenderSiteConf(t, "ep-selective", EasyProfileInput{
		SiteID: "ep-selective", SecurityMode: "block", AllowedMethods: []string{"GET"},
		MaxClientSize: "10m", UseCustomErrorPages: true,
		DisabledErrorPages: []string{"403", "429", "502", "geo_block"},
	})

	for _, code := range []string{"403", "429", "502"} {
		if strings.Contains(conf, "error_page "+code+" ") {
			t.Fatalf("disabled custom page %s is still emitted:\n%s", code, conf)
		}
	}
	if strings.Contains(conf, "error_page 599 =403") {
		t.Fatalf("disabled geo block page must not emit its internal 599 mapping:\n%s", conf)
	}
	for _, code := range []string{"404", "405", "500", "504"} {
		if !strings.Contains(conf, "error_page "+code+" ") {
			t.Fatalf("unrelated custom page %s disappeared:\n%s", code, conf)
		}
	}
}

func TestErrorPages_AcmeAndErrorArtifactsBypassInterceptionSafely(t *testing.T) {
	conf := mustRenderSiteConf(t, "ep-boundaries", EasyProfileInput{
		SiteID: "ep-boundaries", SecurityMode: "block", AllowedMethods: []string{"GET"},
		MaxClientSize: "10m", UseCustomErrorPages: true,
	})

	acmeStart := strings.Index(conf, "location ^~ /.well-known/acme-challenge/ {")
	if acmeStart < 0 {
		t.Fatalf("ACME location is missing:\n%s", conf)
	}
	acme := conf[acmeStart:]
	if next := strings.Index(acme, "\n    location "); next >= 0 {
		acme = acme[:next]
	}
	for _, want := range []string{"proxy_intercept_errors off;", "modsecurity off;", "default_type text/plain;"} {
		if !strings.Contains(acme, want) {
			t.Fatalf("ACME HTTP-01 must contain %q, got:\n%s", want, acme)
		}
	}

	for _, path := range []string{
		"location = /__waf_errors/ep-boundaries/404.html {\n        internal;",
		"location ~ ^/__waf_errors/([a-zA-Z0-9_-]+)/([0-9]+\\.html)$ {\n        internal;",
	} {
		if !strings.Contains(conf, path) {
			t.Fatalf("error artifacts must not be public, missing %q:\n%s", path, conf)
		}
	}
}
