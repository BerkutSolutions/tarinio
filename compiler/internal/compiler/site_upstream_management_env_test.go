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
	if !containsAll(content, "location ^~ /api/ {", "modsecurity off;", "proxy_pass http://control-plane:8080;") {
		t.Fatalf("expected env-configured management site to route API without ModSecurity, got: %s", content)
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
	if !containsAll(content, "location ^~ /api/ {", "modsecurity off;", "proxy_pass http://control-plane-test:8080;") {
		t.Fatalf("expected configured management site to route API without ModSecurity, got: %s", content)
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

func TestRenderSiteUpstreamArtifacts_LocalhostUsesBuiltInManagementRouting(t *testing.T) {
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
		}
	}
	if content == "" {
		t.Fatal("expected site artifact for localhost site")
	}
	if !containsAll(content, "location ^~ /api/ {", "modsecurity off;", "proxy_pass http://control-plane:8080;") {
		t.Fatalf("expected localhost management API routing without ModSecurity, got: %s", content)
	}
}
