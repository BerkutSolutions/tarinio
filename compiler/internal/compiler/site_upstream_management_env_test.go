package compiler

import (
	"strings"
	"testing"
)

func TestRenderSiteUpstreamArtifacts_ManagementSiteRoutesAPIToControlPlaneFromEnvID(t *testing.T) {
	t.Setenv("CONTROL_PLANE_DEV_FAST_START_MANAGEMENT_SITE_ID", "waf.hantico.ru")

	artifacts, err := RenderSiteUpstreamArtifacts(
		[]SiteInput{{
			ID:                "waf.hantico.ru",
			Enabled:           true,
			PrimaryHost:       "waf.hantico.ru",
			ListenHTTP:        true,
			ListenHTTPS:       true,
			DefaultUpstreamID: "mgmt-upstream",
		}},
		[]UpstreamInput{{
			ID:             "mgmt-upstream",
			SiteID:         "waf.hantico.ru",
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
		if artifact.Path == "nginx/sites/waf.hantico.ru.conf" {
			content = string(artifact.Content)
			break
		}
	}
	if content == "" {
		t.Fatal("expected site artifact for env-configured management site")
	}
	if !containsAll(content,
		"location ^~ /api/ {",
		"proxy_pass http://control-plane:8080;",
	) {
		t.Fatalf("expected env-configured management site to route /api to control-plane, got: %s", content)
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
