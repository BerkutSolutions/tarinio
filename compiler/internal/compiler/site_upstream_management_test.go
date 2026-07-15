package compiler

import (
	"strings"
	"testing"
)

func TestRenderSiteUpstreamArtifacts_ManagementSiteRoutesAPIToControlPlane(t *testing.T) {
	artifacts, err := RenderSiteUpstreamArtifacts(
		[]SiteInput{
			{
				ID:                "control-plane-access",
				Enabled:           true,
				PrimaryHost:       "waf.example.com",
				ListenHTTP:        true,
				ListenHTTPS:       true,
				DefaultUpstreamID: "mgmt-upstream",
			},
		},
		[]UpstreamInput{
			{
				ID:             "mgmt-upstream",
				SiteID:         "control-plane-access",
				Scheme:         "http",
				Host:           "ui",
				Port:           80,
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
		if artifact.Path == "nginx/sites/control-plane-access.conf" {
			siteArtifact = artifact
			break
		}
	}

	content := string(siteArtifact.Content)
	apiStart := strings.Index(content, "location ^~ /api/ {")
	if apiStart < 0 {
		t.Fatal("expected management site config to declare a dedicated API location")
	}
	rootStart := strings.Index(content, "location / {")
	if rootStart < 0 || rootStart <= apiStart {
		t.Fatal("expected management site config to keep a root location after easy-locations include")
	}

	apiBlock := content[apiStart:rootStart]
	rootBlock := content[rootStart:]
	if strings.Contains(apiBlock, "modsecurity off;") {
		t.Fatal("management API must inherit the easy anti-bot and ModSecurity guards")
	}
	if !strings.Contains(apiBlock, "proxy_pass http://control-plane:8080;") {
		t.Fatal("expected management API location to proxy to control-plane")
	}
	for _, fragment := range []string{
		"include /etc/waf/nginx/ratelimits/control-plane-access.conf;",
		"include /etc/waf/nginx/easy/control-plane-access.conf;",
	} {
		if !strings.Contains(apiBlock, fragment) {
			t.Fatalf("management API must include %q, got: %s", fragment, apiBlock)
		}
	}
	if !strings.Contains(rootBlock, "proxy_pass http://site_control-plane-access_upstream_mgmt-upstream;") {
		t.Fatal("expected management site root location to keep proxying to the UI upstream")
	}
	for _, path := range []string{"/login", "/login/2fa"} {
		start := strings.Index(content, "location = "+path+" {")
		if start < 0 {
			t.Fatalf("expected dedicated management %s location", path)
		}
		block := content[start:]
		if next := strings.Index(block, "\n    location "); next >= 0 {
			block = block[:next]
		}
		if !strings.Contains(block, "include /etc/waf/nginx/easy/control-plane-access.conf;") {
			t.Fatalf("management %s must include the anti-bot guard, got: %s", path, block)
		}
		if !strings.Contains(block, "include /etc/waf/nginx/ratelimits/control-plane-access.conf;") {
			t.Fatalf("management %s must include site-wide rate and connection limits, got: %s", path, block)
		}
		if !strings.Contains(block, `add_header Cache-Control "no-store" always;`) {
			t.Fatalf("management %s must not be served from a stale browser cache, got: %s", path, block)
		}
	}

	staticStart := strings.Index(content, "location ^~ /static/ {")
	if staticStart < 0 {
		t.Fatal("expected management site static location")
	}
	staticBlock := content[staticStart:]
	if next := strings.Index(staticBlock, "\n    location "); next >= 0 {
		staticBlock = staticBlock[:next]
	}
	if strings.Contains(staticBlock, "include /etc/waf/nginx/easy/control-plane-access.conf;") {
		t.Fatalf("management static assets must bypass the easy anti-bot guard, got: %s", staticBlock)
	}
	if !strings.Contains(staticBlock, "modsecurity off;") || !strings.Contains(staticBlock, "proxy_pass http://ui:80;") {
		t.Fatalf("management static assets must be served directly by UI, got: %s", staticBlock)
	}
}

func TestRenderSiteUpstreamArtifacts_DefaultManagementLoginUsesEasyGuard(t *testing.T) {
	site := SiteInput{ID: "control-plane-access", Enabled: true, PrimaryHost: "waf.example.com", ListenHTTP: true, DefaultUpstreamID: "mgmt-upstream"}
	artifacts, err := RenderSiteUpstreamArtifacts([]SiteInput{site}, []UpstreamInput{{ID: "mgmt-upstream", SiteID: site.ID, Scheme: "http", Host: "ui", Port: 80}})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	var base string
	for _, artifact := range artifacts {
		if artifact.Path == "nginx/conf.d/base.conf" {
			base = string(artifact.Content)
			break
		}
	}
	for _, path := range []string{"/login", "/login/2fa"} {
		start := strings.Index(base, "location = "+path+" {")
		if start < 0 {
			t.Fatalf("missing default management %s location", path)
		}
		block := base[start:]
		if next := strings.Index(block, "\n    location "); next >= 0 {
			block = block[:next]
		}
		for _, include := range []string{"include /etc/waf/nginx/ratelimits/control-plane-access.conf;", "include /etc/waf/nginx/easy/control-plane-access.conf;"} {
			if !strings.Contains(block, include) {
				t.Fatalf("default management %s must include %q, got: %s", path, include, block)
			}
		}
	}
}

func TestRenderSiteUpstreamArtifacts_ManagementAPIDoesNotReplaceUpstreamErrors(t *testing.T) {
	site := SiteInput{ID: "control-plane-access", Enabled: true, PrimaryHost: "waf.example.com", ListenHTTP: true, DefaultUpstreamID: "mgmt-upstream", UseEasyConfig: true, UseCustomErrorPages: true}
	artifacts, err := RenderSiteUpstreamArtifacts([]SiteInput{site}, []UpstreamInput{{ID: "mgmt-upstream", SiteID: site.ID, Scheme: "http", Host: "ui", Port: 80}})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	for _, artifact := range artifacts {
		if artifact.Path != "nginx/sites/control-plane-access.conf" {
			continue
		}
		content := string(artifact.Content)
		apiStart := strings.Index(content, "location ^~ /api/ {")
		if apiStart < 0 {
			t.Fatalf("management API location is missing: %s", content)
		}
		apiBlock := content[apiStart:]
		if next := strings.Index(apiBlock, "\n    location "); next >= 0 {
			apiBlock = apiBlock[:next]
		}
		if !strings.Contains(apiBlock, "proxy_intercept_errors off;") {
			t.Fatalf("management API must preserve upstream error responses, got: %s", apiBlock)
		}
		if strings.Contains(apiBlock, "proxy_intercept_errors on;") {
			t.Fatalf("management API must never replace upstream errors with HTML pages, got: %s", apiBlock)
		}
	}
}
