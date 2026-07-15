package compiler

import (
	"fmt"
	"strings"
	"testing"

	rootcompiler "waf/compiler"
)

func TestRenderSiteUpstreamArtifacts_Deterministic(t *testing.T) {
	sites := []SiteInput{
		{
			ID:                "site-b",
			Name:              "Site B",
			Enabled:           true,
			PrimaryHost:       "b.example.com",
			Aliases:           []string{"www.b.example.com", "www.b.example.com"},
			ListenHTTP:        true,
			ListenHTTPS:       false,
			DefaultUpstreamID: "up-b",
		},
		{
			ID:                "site-a",
			Name:              "Site A",
			Enabled:           true,
			PrimaryHost:       "a.example.com",
			Aliases:           []string{"www.a.example.com"},
			ListenHTTP:        true,
			ListenHTTPS:       true,
			DefaultUpstreamID: "up-a",
		},
	}

	upstreams := []UpstreamInput{
		{
			ID:             "up-b",
			SiteID:         "site-b",
			Name:           "main",
			Scheme:         "http",
			Host:           "app-b",
			Port:           8080,
			BasePath:       "/",
			PassHostHeader: false,
		},
		{
			ID:             "up-a",
			SiteID:         "site-a",
			Name:           "main",
			Scheme:         "http",
			Host:           "app-a",
			Port:           8080,
			BasePath:       "/",
			PassHostHeader: true,
		},
	}

	first, err := RenderSiteUpstreamArtifacts(sites, upstreams)
	if err != nil {
		t.Fatalf("first render failed: %v", err)
	}
	second, err := RenderSiteUpstreamArtifacts(sites, upstreams)
	if err != nil {
		t.Fatalf("second render failed: %v", err)
	}

	if len(first) != len(second) {
		t.Fatalf("artifact counts differ: %d vs %d", len(first), len(second))
	}

	for i := range first {
		if first[i].Path != second[i].Path {
			t.Fatalf("artifact path differs at %d: %s vs %s", i, first[i].Path, second[i].Path)
		}
		if first[i].Checksum != second[i].Checksum {
			t.Fatalf("artifact checksum differs for %s", first[i].Path)
		}
	}
}

func TestRenderSiteErrorArtifactsUsesOnlyLocalizedCanonicalPreviewPages(t *testing.T) {
	artifacts, err := renderSiteErrorArtifacts("site-a")
	if err != nil {
		t.Fatalf("render error artifacts: %v", err)
	}
	if len(artifacts) != len(supportedErrorStatusCodes) {
		t.Fatalf("unexpected error artifact count: %d", len(artifacts))
	}
	for _, code := range responseErrorStatusCodes() {
		path := errorPagePreviewPath(code)
		content, err := rootcompiler.TemplatesFS.ReadFile(path)
		if err != nil {
			t.Fatalf("canonical preview for %d: %v", code, err)
		}
		for _, locale := range []string{"en:", "ru:", "de:", "sr:", "zh:"} {
			if !strings.Contains(string(content), locale) {
				t.Fatalf("preview %s lacks locale %s", path, locale)
			}
		}
	}
}

func TestRenderSiteErrorArtifactsUsesDedicatedGeneratedExtendedPages(t *testing.T) {
	artifacts, err := renderSiteErrorArtifacts("site-a")
	if err != nil {
		t.Fatalf("render error artifacts: %v", err)
	}
	for _, expected := range []struct{ path, title string }{
		{"errors/site-a/495.html", "SSL Certificate Error"},
		{"errors/site-a/522.html", "Connection Timed Out"},
		{"errors/site-a/526.html", "Invalid SSL Certificate"},
	} {
		found := false
		for _, artifact := range artifacts {
			if artifact.Path == expected.path {
				found = true
				if !strings.Contains(string(artifact.Content), expected.title) || strings.Contains(string(artifact.Content), "HTTP 400") {
					t.Fatalf("artifact %s does not contain its dedicated page", expected.path)
				}
			}
		}
		if !found {
			t.Fatalf("expected artifact %s", expected.path)
		}
	}
}

func TestRenderSiteUpstreamArtifacts_ManifestCompatibleKinds(t *testing.T) {
	artifacts, err := RenderSiteUpstreamArtifacts(
		[]SiteInput{
			{
				ID:                "site-a",
				Enabled:           true,
				PrimaryHost:       "a.example.com",
				ListenHTTP:        true,
				ListenHTTPS:       true,
				DefaultUpstreamID: "up-a",
			},
		},
		[]UpstreamInput{
			{
				ID:             "up-a",
				SiteID:         "site-a",
				Scheme:         "http",
				Host:           "app-a",
				Port:           8080,
				BasePath:       "/",
				PassHostHeader: true,
			},
		},
	)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	expectedArtifacts := 4 + (2 * len(supportedErrorStatusCodes)) // nginx.conf + base.conf + site.conf + geo_block.html + per-site/global error pages
	if len(artifacts) != expectedArtifacts {
		t.Fatalf("expected %d site/runtime artifacts, got %d", expectedArtifacts, len(artifacts))
	}

	for _, artifact := range artifacts {
		if artifact.Kind != ArtifactKindNginxConfig {
			t.Fatalf("unexpected artifact kind for %s: %s", artifact.Path, artifact.Kind)
		}
		if artifact.Checksum == "" {
			t.Fatalf("missing checksum for %s", artifact.Path)
		}
		if strings.TrimSpace(string(artifact.Content)) == "" {
			t.Fatalf("empty content for %s", artifact.Path)
		}
	}
}

func TestRenderSiteUpstreamArtifacts_DefaultServerRoutesUnknownHostsToBranded421(t *testing.T) {
	artifacts, err := RenderSiteUpstreamArtifacts(
		[]SiteInput{
			{
				ID:                "site-a",
				Enabled:           true,
				PrimaryHost:       "a.example.com",
				ListenHTTP:        true,
				ListenHTTPS:       true,
				DefaultUpstreamID: "up-a",
			},
		},
		[]UpstreamInput{
			{
				ID:             "up-a",
				SiteID:         "site-a",
				Scheme:         "http",
				Host:           "app-a",
				Port:           8080,
				BasePath:       "/",
				PassHostHeader: true,
			},
		},
	)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	var baseArtifact ArtifactOutput
	for _, artifact := range artifacts {
		if artifact.Path == "nginx/conf.d/base.conf" {
			baseArtifact = artifact
			break
		}
	}
	content := string(baseArtifact.Content)
	for _, fragment := range []string{
		`map "$waf_site_id:$uri" $waf_unknown_host {`,
		`~^_global:(?!/\.well-known/acme-challenge/) 1;`,
		"location ^~ /.well-known/acme-challenge/ {",
		"modsecurity off;",
		"error_page 421 /__waf_errors/_global/421.html?rid=$request_id&ip=$remote_addr&ts=$msec;",
		"rewrite ^ /__waf_errors/_global/421.html last;",
		"error_page 421 /__waf_errors/_global/421.html?rid=$request_id&ip=$remote_addr&ts=$msec;",
		"location = /__waf_errors/_global/421.html {",
		"alias /etc/waf/errors/_global/421.html;",
	} {
		if !strings.Contains(content, fragment) {
			t.Fatalf("expected base config to contain %q, got: %s", fragment, content)
		}
	}
}

func TestRenderSiteUpstreamArtifacts_DefaultHTTPSServerUsesFirstActiveTLSRefAndReturns421(t *testing.T) {
	artifacts, err := RenderSiteUpstreamArtifacts(
		[]SiteInput{
			{
				ID:                "site-a",
				Enabled:           true,
				PrimaryHost:       "a.example.com",
				ListenHTTP:        true,
				ListenHTTPS:       true,
				DefaultUpstreamID: "up-a",
			},
		},
		[]UpstreamInput{
			{
				ID:             "up-a",
				SiteID:         "site-a",
				Scheme:         "http",
				Host:           "app-a",
				Port:           8080,
				BasePath:       "/",
				PassHostHeader: true,
			},
		},
	)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	var nginxArtifact ArtifactOutput
	for _, artifact := range artifacts {
		if artifact.Path == "nginx/nginx.conf" {
			nginxArtifact = artifact
			break
		}
	}
	content := string(nginxArtifact.Content)
	for _, fragment := range []string{
		"listen 443 ssl default_server;",
		"server_name _;",
		"include /etc/waf/tls/site-a.conf;",
		"error_page 418 =421 /__waf_errors/_global/421.html?rid=$request_id&ip=$remote_addr&ts=$msec;",
		"return 418;",
	} {
		if !strings.Contains(content, fragment) {
			t.Fatalf("expected nginx main config to contain %q, got: %s", fragment, content)
		}
	}
	for _, code := range responseErrorStatusCodes() {
		if !strings.Contains(content, fmt.Sprintf("error_page %d /__waf_errors/_global/%d.html", code, code)) {
			t.Fatalf("expected branded default TLS error page for %d, got: %s", code, content)
		}
	}
}

func TestRenderSiteUpstreamArtifacts_BlockDirectIPAccessUses444AndSecurityReasons(t *testing.T) {
	artifacts, err := RenderSiteUpstreamArtifactsWithOptions(
		[]SiteInput{{ID: "site-a", Enabled: true, PrimaryHost: "a.example.com", ListenHTTP: true, ListenHTTPS: true, DefaultUpstreamID: "up-a"}},
		[]UpstreamInput{{ID: "up-a", SiteID: "site-a", Scheme: "http", Host: "app-a", Port: 8080, BasePath: "/"}},
		DefaultServerOptions{BlockDirectIPAccess: true},
	)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	contents := map[string]string{}
	for _, artifact := range artifacts {
		contents[artifact.Path] = string(artifact.Content)
	}
	for path, fragment := range map[string]string{
		"nginx/nginx.conf":       "if ($waf_direct_ip_request = 1) {\n            return 444;",
		"nginx/conf.d/base.conf": `"_global:1" "direct_ip_blocked";`,
	} {
		if !strings.Contains(contents[path], fragment) {
			t.Fatalf("%s must contain %q, got: %s", path, fragment, contents[path])
		}
	}
	if !strings.Contains(contents["nginx/conf.d/base.conf"], `"security_reason":"$waf_security_reason"`) {
		t.Fatalf("access log must include normalized security_reason: %s", contents["nginx/conf.d/base.conf"])
	}
	if !strings.Contains(contents["nginx/conf.d/base.conf"], `~^[0-9][0-9]?[0-9]?\.[0-9][0-9]?[0-9]?\.[0-9][0-9]?[0-9]?\.[0-9][0-9]?[0-9]?$ 1;`) {
		t.Fatalf("direct-IP regex must avoid brace quantifiers for nginx parsing: %s", contents["nginx/conf.d/base.conf"])
	}
}

func TestRenderSiteUpstreamArtifacts_DisabledServiceLeavesNoRuntimeArtifacts(t *testing.T) {
	artifacts, err := RenderSiteUpstreamArtifacts(
		[]SiteInput{
			{ID: "198.51.100.54", Enabled: false, PrimaryHost: "198.51.100.54", ListenHTTPS: true, DefaultUpstreamID: "old"},
			{ID: "waf.site.com", Enabled: true, PrimaryHost: "waf.site.com", ListenHTTPS: true, DefaultUpstreamID: "new"},
		},
		[]UpstreamInput{
			{ID: "old", SiteID: "198.51.100.54", Scheme: "http", Host: "old-app", Port: 80},
			{ID: "new", SiteID: "waf.site.com", Scheme: "http", Host: "ui", Port: 80},
		},
	)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	for _, artifact := range artifacts {
		if strings.Contains(artifact.Path, "198.51.100.54") || strings.Contains(string(artifact.Content), "198.51.100.54") {
			t.Fatalf("disabled service leaked into runtime artifact %s: %s", artifact.Path, artifact.Content)
		}
	}
	var main string
	for _, artifact := range artifacts {
		if artifact.Path == "nginx/nginx.conf" {
			main = string(artifact.Content)
		}
	}
	if !strings.Contains(main, "alias /etc/waf/errors/_global/421.html;") || !strings.Contains(main, "error_page 418 =421 /__waf_errors/_global/421.html") || !strings.Contains(main, "return 418;") {
		t.Fatalf("default TLS server must return branded 421, got: %s", main)
	}
}

func TestRenderSiteUpstreamArtifacts_PrefersNamedHTTPSSitesBeforeIPHTTPSSites(t *testing.T) {
	artifacts, err := RenderSiteUpstreamArtifacts(
		[]SiteInput{
			{
				ID:                "198.51.100.54",
				Enabled:           true,
				PrimaryHost:       "198.51.100.54",
				ListenHTTP:        true,
				ListenHTTPS:       true,
				DefaultUpstreamID: "up-ip",
			},
			{
				ID:                "panel.example.test",
				Enabled:           true,
				PrimaryHost:       "panel.example.test",
				ListenHTTP:        true,
				ListenHTTPS:       true,
				DefaultUpstreamID: "up-domain",
			},
		},
		[]UpstreamInput{
			{
				ID:             "up-ip",
				SiteID:         "198.51.100.54",
				Scheme:         "http",
				Host:           "app-ip",
				Port:           8080,
				BasePath:       "/",
				PassHostHeader: true,
			},
			{
				ID:             "up-domain",
				SiteID:         "panel.example.test",
				Scheme:         "http",
				Host:           "app-domain",
				Port:           8080,
				BasePath:       "/",
				PassHostHeader: true,
			},
		},
	)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	var nginxArtifact ArtifactOutput
	for _, artifact := range artifacts {
		if artifact.Path == "nginx/nginx.conf" {
			nginxArtifact = artifact
			break
		}
	}
	nginxContent := string(nginxArtifact.Content)
	if !strings.Contains(nginxContent, "include /etc/waf/tls/panel.example.test.conf;") {
		t.Fatalf("expected default HTTPS server to use domain TLS ref, got: %s", nginxContent)
	}
	if strings.Contains(nginxContent, "include /etc/waf/tls/198.51.100.54.conf;") {
		t.Fatalf("did not expect default HTTPS server to use IP TLS ref when domain HTTPS site exists, got: %s", nginxContent)
	}

	var sitesInOrder []string
	for _, artifact := range artifacts {
		if strings.HasPrefix(artifact.Path, "nginx/sites/") {
			sitesInOrder = append(sitesInOrder, artifact.Path)
		}
	}
	if len(sitesInOrder) < 2 {
		t.Fatalf("expected at least two site artifacts, got: %v", sitesInOrder)
	}
	if sitesInOrder[0] != "nginx/sites/panel.example.test.conf" || sitesInOrder[1] != "nginx/sites/198.51.100.54.conf" {
		t.Fatalf("expected domain HTTPS site before IP HTTPS site, got: %v", sitesInOrder)
	}
}

func TestRenderSiteUpstreamArtifacts_DefaultServerUsesConfiguredManagementUpstream(t *testing.T) {
	t.Setenv("WAF_MANAGEMENT_API_UPSTREAM_HOST", "control-plane-test")

	artifacts, err := RenderSiteUpstreamArtifacts(
		[]SiteInput{
			{
				ID:                "site-a",
				Enabled:           true,
				PrimaryHost:       "a.example.com",
				ListenHTTP:        true,
				ListenHTTPS:       true,
				DefaultUpstreamID: "up-a",
			},
		},
		[]UpstreamInput{
			{
				ID:             "up-a",
				SiteID:         "site-a",
				Scheme:         "http",
				Host:           "app-a",
				Port:           8080,
				BasePath:       "/",
				PassHostHeader: true,
			},
		},
	)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	var baseArtifact ArtifactOutput
	for _, artifact := range artifacts {
		if artifact.Path == "nginx/conf.d/base.conf" {
			baseArtifact = artifact
			break
		}
	}

	content := string(baseArtifact.Content)
	if !strings.Contains(content, "proxy_pass http://control-plane-test:8080;") {
		t.Fatalf("expected base config to proxy /api to configured management upstream, got: %s", content)
	}
}

func TestRenderSiteUpstreamArtifacts_MapsHostsToSiteIDsInBaseConfig(t *testing.T) {
	artifacts, err := RenderSiteUpstreamArtifacts(
		[]SiteInput{
			{
				ID:                "site-a",
				Enabled:           true,
				PrimaryHost:       "a.example.com",
				Aliases:           []string{"www.a.example.com"},
				ListenHTTP:        true,
				ListenHTTPS:       true,
				DefaultUpstreamID: "up-a",
			},
			{
				ID:                "site-b",
				Enabled:           true,
				PrimaryHost:       "logs.example.test",
				ListenHTTP:        true,
				ListenHTTPS:       false,
				DefaultUpstreamID: "up-b",
			},
		},
		[]UpstreamInput{
			{
				ID:             "up-a",
				SiteID:         "site-a",
				Scheme:         "http",
				Host:           "app-a",
				Port:           8080,
				BasePath:       "/",
				PassHostHeader: true,
			},
			{
				ID:             "up-b",
				SiteID:         "site-b",
				Scheme:         "http",
				Host:           "app-b",
				Port:           9000,
				BasePath:       "/",
				PassHostHeader: true,
			},
		},
	)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	var baseArtifact ArtifactOutput
	for _, artifact := range artifacts {
		if artifact.Path == "nginx/conf.d/base.conf" {
			baseArtifact = artifact
			break
		}
	}
	content := string(baseArtifact.Content)
	for _, fragment := range []string{
		`a.example.com "site-a";`,
		`www.a.example.com "site-a";`,
		`logs.example.test "site-b";`,
		`default "_global";`,
	} {
		if !strings.Contains(content, fragment) {
			t.Fatalf("expected base host map to contain %q, got: %s", fragment, content)
		}
	}
}
