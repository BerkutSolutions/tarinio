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
		t.Fatal("expected management site config to declare dedicated /api location")
	}
	rootStart := strings.Index(content, "location / {")
	if rootStart < 0 || rootStart <= apiStart {
		t.Fatal("expected management site config to keep a root location after /api")
	}

	apiBlock := content[apiStart:rootStart]
	rootBlock := content[rootStart:]
	if !strings.Contains(apiBlock, "proxy_pass http://control-plane:8080;") {
		t.Fatal("expected management site /api location to proxy to control-plane")
	}
	if strings.Contains(apiBlock, "proxy_pass http://site_control-plane-access_upstream_mgmt-upstream;") {
		t.Fatal("did not expect management site /api location to proxy to the UI upstream")
	}
	if !strings.Contains(rootBlock, "proxy_pass http://site_control-plane-access_upstream_mgmt-upstream;") {
		t.Fatal("expected management site root location to keep proxying to the UI upstream")
	}
}
