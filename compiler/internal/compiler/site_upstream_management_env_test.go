package compiler

import (
	"strings"
	"testing"
)

func TestRenderSiteUpstreamArtifacts_ManagementSiteRoutesAPIToControlPlaneFromEnvID(t *testing.T) {
	t.Setenv("CONTROL_PLANE_DEV_FAST_START_MANAGEMENT_SITE_ID", "ui.example.test")

	artifacts, err := RenderSiteUpstreamArtifacts(
		[]SiteInput{{
			ID:                "ui.example.test",
			Enabled:           true,
			PrimaryHost:       "ui.example.test",
			ListenHTTP:        true,
			ListenHTTPS:       true,
			DefaultUpstreamID: "mgmt-upstream",
		}},
		[]UpstreamInput{{
			ID:             "mgmt-upstream",
			SiteID:         "ui.example.test",
			Scheme:         "http",
			Host:           "ui",
			Port:           80,
			PassHostHeader: true,
		}},
	)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	content := ""
	for _, artifact := range artifacts {
		if artifact.Path == "nginx/sites/ui.example.test.conf" {
			content = string(artifact.Content)
			break
		}
	}
	if content == "" {
		t.Fatal("expected site artifact for env-configured management site")
	}
	if strings.Contains(content, "location ^~ /api/ {") {
		t.Fatalf("did not expect env-configured management site to keep catch-all /api location in site template, got: %s", content)
	}
	if !strings.Contains(content, "include /etc/waf/nginx/easy-locations/ui.example.test.conf;") {
		t.Fatalf("expected env-configured management site to include easy-locations for management APIs, got: %s", content)
	}
}

func TestRenderSiteUpstreamArtifacts_ManagementSiteRoutesAPIToConfiguredManagementUpstream(t *testing.T) {
	t.Setenv("WAF_MANAGEMENT_API_UPSTREAM_HOST", "control-plane-test")

	artifacts, err := RenderSiteUpstreamArtifacts(
		[]SiteInput{{
			ID:                "control-plane-access",
			Enabled:           true,
			PrimaryHost:       "ui.example.test",
			ListenHTTP:        true,
			ListenHTTPS:       true,
			DefaultUpstreamID: "mgmt-upstream",
		}},
		[]UpstreamInput{{
			ID:             "mgmt-upstream",
			SiteID:         "control-plane-access",
			Scheme:         "http",
			Host:           "ui",
			Port:           80,
			PassHostHeader: true,
		}},
	)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	content := ""
	for _, artifact := range artifacts {
		if artifact.Path == "nginx/sites/control-plane-access.conf" {
			content = string(artifact.Content)
			break
		}
	}
	if content == "" {
		t.Fatal("expected site artifact for configured management site")
	}
	if strings.Contains(content, "location ^~ /api/ {") {
		t.Fatalf("did not expect configured management site to keep catch-all /api location in site template, got: %s", content)
	}
	if !strings.Contains(content, "include /etc/waf/nginx/easy-locations/control-plane-access.conf;") {
		t.Fatalf("expected configured management site to include easy-locations for management APIs, got: %s", content)
	}
}

func containsAll(content string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(content, part) {
			return false
		}
	}
	return true
}

func TestRenderSiteUpstreamArtifacts_LocalhostStaysRegularSiteWithoutExplicitManagementID(t *testing.T) {
	t.Setenv("CONTROL_PLANE_DEV_FAST_START_MANAGEMENT_SITE_ID", "control-plane-access")

	artifacts, err := RenderSiteUpstreamArtifacts(
		[]SiteInput{{
			ID:                "localhost",
			Enabled:           true,
			PrimaryHost:       "localhost",
			ListenHTTP:        true,
			ListenHTTPS:       true,
			DefaultUpstreamID: "site-upstream",
		}},
		[]UpstreamInput{{
			ID:             "site-upstream",
			SiteID:         "localhost",
			Scheme:         "http",
			Host:           "app",
			Port:           8081,
			PassHostHeader: true,
		}},
	)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	content := ""
	for _, artifact := range artifacts {
		if artifact.Path == "nginx/sites/localhost.conf" {
			content = string(artifact.Content)
			break
		}
	}
	if content == "" {
		t.Fatal("expected site artifact for localhost site")
	}
	if !containsAll(content,
		"location ^~ /api/ {",
		"proxy_pass http://site_localhost_upstream_site-upstream;",
	) {
		t.Fatalf("expected localhost site to keep /api on its own upstream, got: %s", content)
	}
	if strings.Contains(content, "proxy_pass http://control-plane:8080;") {
		t.Fatalf("expected localhost site to avoid management API upstream, got: %s", content)
	}
}
