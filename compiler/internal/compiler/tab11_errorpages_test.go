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
	if !strings.Contains(conf, "/__waf_errors/_global/geo_block.html") {
		t.Fatalf("expected global geo_block path for 451, got:\n%s", conf)
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
