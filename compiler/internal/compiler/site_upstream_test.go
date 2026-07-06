package compiler

import (
	"strings"
	"testing"
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
		"set $waf_unknown_host 1;",
		"if ($waf_site_id = \"_global\") {",
		"rewrite ^ /__waf_errors/_global/421.html?rid=$request_id&ip=$remote_addr&ts=$msec last;",
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
		"return 421;",
	} {
		if !strings.Contains(content, fragment) {
			t.Fatalf("expected nginx main config to contain %q, got: %s", fragment, content)
		}
	}
}

func TestRenderSiteUpstreamArtifacts_PrefersNamedHTTPSSitesBeforeIPHTTPSSites(t *testing.T) {
	artifacts, err := RenderSiteUpstreamArtifacts(
		[]SiteInput{
			{
				ID:                "135.136.191.54",
				Enabled:           true,
				PrimaryHost:       "135.136.191.54",
				ListenHTTP:        true,
				ListenHTTPS:       true,
				DefaultUpstreamID: "up-ip",
			},
			{
				ID:                "prewaf.hantico.ru",
				Enabled:           true,
				PrimaryHost:       "prewaf.hantico.ru",
				ListenHTTP:        true,
				ListenHTTPS:       true,
				DefaultUpstreamID: "up-domain",
			},
		},
		[]UpstreamInput{
			{
				ID:             "up-ip",
				SiteID:         "135.136.191.54",
				Scheme:         "http",
				Host:           "app-ip",
				Port:           8080,
				BasePath:       "/",
				PassHostHeader: true,
			},
			{
				ID:             "up-domain",
				SiteID:         "prewaf.hantico.ru",
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
	if !strings.Contains(nginxContent, "include /etc/waf/tls/prewaf.hantico.ru.conf;") {
		t.Fatalf("expected default HTTPS server to use domain TLS ref, got: %s", nginxContent)
	}
	if strings.Contains(nginxContent, "include /etc/waf/tls/135.136.191.54.conf;") {
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
	if sitesInOrder[0] != "nginx/sites/prewaf.hantico.ru.conf" || sitesInOrder[1] != "nginx/sites/135.136.191.54.conf" {
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
				PrimaryHost:       "sentry.hantico.ru",
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
		`sentry.hantico.ru "site-b";`,
		`default "_global";`,
	} {
		if !strings.Contains(content, fragment) {
			t.Fatalf("expected base host map to contain %q, got: %s", fragment, content)
		}
	}
}
