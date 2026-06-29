package compiler

import (
	"strings"
	"testing"
)

func renderSiteConfForTest(t *testing.T, data nginxSiteData) string {
	t.Helper()
	tmplPath := "templates/nginx/sites/site.conf.tmpl"
	content, err := renderTemplate(tmplPath, data)
	if err != nil {
		t.Fatalf("renderTemplate(%s): %v", tmplPath, err)
	}
	return string(content)
}

func TestHealthCheck_Enabled_GeneratesProxyNextUpstream(t *testing.T) {
	conf := renderSiteConfForTest(t, nginxSiteData{
		SiteID:             "test-site",
		SiteIDSlug:         "test_site",
		ServerNames:        []string{"test.example.com"},
		ListenHTTP:         true,
		UpstreamName:       "site_test_site_upstream",
		UpstreamAddress:    "backend:8080",
		ProxyPassTarget:    "http://site_test_site_upstream",
		PassHostHeader:     true,
		HealthCheckEnabled: true,
	})
	if !strings.Contains(conf, "keepalive 32") {
		t.Errorf("expected keepalive 32 in upstream block when HealthCheckEnabled=true, got:\n%s", conf)
	}
	if !strings.Contains(conf, "proxy_next_upstream error timeout http_502 http_503") {
		t.Errorf("expected proxy_next_upstream when HealthCheckEnabled=true, got:\n%s", conf)
	}
	if !strings.Contains(conf, "proxy_next_upstream_tries 2") {
		t.Errorf("expected proxy_next_upstream_tries 2 when HealthCheckEnabled=true, got:\n%s", conf)
	}
}

func TestHealthCheck_Disabled_NoProxyNextUpstream(t *testing.T) {
	conf := renderSiteConfForTest(t, nginxSiteData{
		SiteID:             "test-site",
		SiteIDSlug:         "test_site",
		ServerNames:        []string{"test.example.com"},
		ListenHTTP:         true,
		UpstreamName:       "site_test_site_upstream",
		UpstreamAddress:    "backend:8080",
		ProxyPassTarget:    "http://site_test_site_upstream",
		PassHostHeader:     true,
		HealthCheckEnabled: false,
	})
	if strings.Contains(conf, "proxy_next_upstream") {
		t.Errorf("expected NO proxy_next_upstream when HealthCheckEnabled=false, got:\n%s", conf)
	}
	if strings.Contains(conf, "keepalive 32") {
		t.Errorf("expected NO keepalive 32 when HealthCheckEnabled=false, got:\n%s", conf)
	}
}
