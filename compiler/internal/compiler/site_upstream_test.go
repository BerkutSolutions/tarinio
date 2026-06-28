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

func TestRenderSiteUpstreamArtifacts_UsesFileBasedModSecurityConfig(t *testing.T) {
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

	var siteArtifact ArtifactOutput
	for _, artifact := range artifacts {
		if artifact.Path == "nginx/sites/site-a.conf" {
			siteArtifact = artifact
			break
		}
	}

	content := string(siteArtifact.Content)
	if !strings.Contains(content, "modsecurity on;") {
		t.Fatal("expected nginx site config to enable modsecurity")
	}
	if !strings.Contains(content, "modsecurity_rules_file /etc/waf/modsecurity/sites/site-a.conf;") {
		t.Fatal("expected nginx site config to use file-based modsecurity_rules_file")
	}
	if strings.Contains(content, "modsecurity_rules '") {
		t.Fatal("did not expect inline modsecurity_rules in nginx site config")
	}
}

func TestRenderSiteUpstreamArtifacts_WiresErrorPagesAndRateLimitIncludes(t *testing.T) {
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

	var siteArtifact ArtifactOutput
	for _, artifact := range artifacts {
		if artifact.Path == "nginx/sites/site-a.conf" {
			siteArtifact = artifact
			break
		}
	}

	content := string(siteArtifact.Content)
	if strings.Contains(content, "set $waf_site_id ") {
		t.Fatal("did not expect nginx site config to set waf_site_id explicitly")
	}
	if !strings.Contains(content, "proxy_intercept_errors on;") {
		t.Fatal("expected proxy_intercept_errors to be enabled in nginx site config")
	}
	if !strings.Contains(content, "location ^~ /api/ {") {
		t.Fatal("expected dedicated /api location in nginx site config")
	}
	if !strings.Contains(content, "proxy_intercept_errors off;") {
		t.Fatal("expected proxy_intercept_errors to be disabled for /api location")
	}
	if !strings.Contains(content, "include /etc/waf/nginx/ratelimits/site-a.conf;") {
		t.Fatal("expected per-site rate limit include in nginx site config")
	}
	if !strings.Contains(content, "error_page 400 /__waf_errors/site-a/400.html?rid=$request_id&ip=$remote_addr&ts=$msec;") {
		t.Fatal("expected 400 error page wiring in nginx site config")
	}
	if !strings.Contains(content, "error_page 403 /__waf_errors/site-a/403.html?rid=$request_id&ip=$remote_addr&ts=$msec;") {
		t.Fatal("expected 403 error page wiring in nginx site config")
	}
	if !strings.Contains(content, "alias /etc/waf/errors/site-a/403.html;") {
		t.Fatal("expected 403 alias to canonical runtime error asset path")
	}
	if !strings.Contains(content, "alias /etc/waf/errors/site-a/429.html;") {
		t.Fatal("expected 429 alias to canonical runtime error asset path")
	}
	if !strings.Contains(content, "alias /etc/waf/errors/site-a/502.html;") {
		t.Fatal("expected 502 alias to canonical runtime error asset path")
	}
}

func TestRenderSiteUpstreamArtifacts_DefaultServerKeepsAdminRoutesReachable(t *testing.T) {
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
		"location ^~ /api/ {",
		"proxy_pass http://control-plane:8080;",
		"location = /login {",
		"location = /login/2fa {",
		"location ^~ /onboarding/ {",
		"location ^~ /static/ {",
		"return 308 https://$host$request_uri;",
	} {
		if !strings.Contains(content, fragment) {
			t.Fatalf("expected base config to contain %q, got: %s", fragment, content)
		}
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
	} {
		if !strings.Contains(content, fragment) {
			t.Fatalf("expected base config to contain host map entry %q, got: %s", fragment, content)
		}
	}
}

func TestRenderSiteUpstreamArtifacts_DeduplicatesConflictingHostMapEntries(t *testing.T) {
	// Two enabled sites that claim the same host must not produce duplicate
	// keys in the `map $host $waf_site_id` block. nginx treats a duplicate
	// map key as a fatal "conflicting parameter" error, which crash-loops the
	// runtime. The first site (by sorted order) wins.
	artifacts, err := RenderSiteUpstreamArtifacts(
		[]SiteInput{
			{
				ID:                "mgmt-localhost-site",
				Enabled:           true,
				PrimaryHost:       "localhost",
				ListenHTTP:        true,
				ListenHTTPS:       true,
				DefaultUpstreamID: "up-mgmt",
			},
			{
				ID:                "app-localhost-site",
				Enabled:           true,
				PrimaryHost:       "localhost",
				ListenHTTP:        true,
				ListenHTTPS:       true,
				DefaultUpstreamID: "up-app",
			},
		},
		[]UpstreamInput{
			{ID: "up-mgmt", SiteID: "mgmt-localhost-site", Scheme: "http", Host: "app-mgmt", Port: 8080, BasePath: "/", PassHostHeader: true},
			{ID: "up-app", SiteID: "app-localhost-site", Scheme: "http", Host: "app-app", Port: 8081, BasePath: "/", PassHostHeader: true},
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
	if got := strings.Count(content, "    localhost \""); got != 2 {
		// Two map blocks (waf_site_id + waf_site_id_log) each get exactly one
		// localhost entry; never two within a single block.
		t.Fatalf("expected exactly 2 localhost map entries (one per map block), got %d in:\n%s", got, content)
	}
	// Sorted order puts "app-localhost-site" before "mgmt-localhost-site".
	if !strings.Contains(content, `localhost "app-localhost-site";`) {
		t.Fatalf("expected localhost to map to first sorted site, got:\n%s", content)
	}
	if strings.Contains(content, `localhost "mgmt-localhost-site";`) {
		t.Fatalf("did not expect a second conflicting localhost map entry, got:\n%s", content)
	}
}

func TestRenderSiteUpstreamArtifacts_UsesValidUpstreamServerAddress(t *testing.T) {
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

	var siteArtifact ArtifactOutput
	for _, artifact := range artifacts {
		if artifact.Path == "nginx/sites/site-a.conf" {
			siteArtifact = artifact
			break
		}
	}

	content := string(siteArtifact.Content)
	if !strings.Contains(content, "server app-a:8080;") {
		t.Fatal("expected upstream block to use host:port server address")
	}
	if strings.Contains(content, "server http://") {
		t.Fatal("did not expect upstream block to contain scheme-prefixed server address")
	}
}
