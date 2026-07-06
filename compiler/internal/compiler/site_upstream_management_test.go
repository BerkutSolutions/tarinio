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
	if strings.Contains(content, "location ^~ /api/ {") {
		t.Fatal("did not expect management site config to declare catch-all /api location in site template")
	}
	apiStart := strings.Index(content, "include /etc/waf/nginx/easy-locations/control-plane-access.conf;")
	if apiStart < 0 {
		t.Fatal("expected management site config to include easy-locations before root location")
	}
	rootStart := strings.Index(content, "location / {")
	if rootStart < 0 || rootStart <= apiStart {
		t.Fatal("expected management site config to keep a root location after easy-locations include")
	}

	apiBlock := content[apiStart:rootStart]
	rootBlock := content[rootStart:]
	if !strings.Contains(apiBlock, "include /etc/waf/nginx/easy-locations/control-plane-access.conf;") {
		t.Fatal("expected management site config to include dedicated easy-locations for management APIs")
	}
	if strings.Contains(apiBlock, "location ^~ /api/") {
		t.Fatal("did not expect management site management-api section to define catch-all /api routing in site template")
	}
	if !strings.Contains(rootBlock, "proxy_pass http://site_control-plane-access_upstream_mgmt-upstream;") {
		t.Fatal("expected management site root location to keep proxying to the UI upstream")
	}
}
