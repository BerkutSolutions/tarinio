package compiler

import "testing"

func TestRenderSiteUpstreamArtifacts_SetsProxyHeadersHashSizing(t *testing.T) {
	artifacts, err := RenderSiteUpstreamArtifacts(
		[]SiteInput{{
			ID:                "site-a",
			Enabled:           true,
			PrimaryHost:       "a.example.com",
			ListenHTTP:        true,
			ListenHTTPS:       true,
			DefaultUpstreamID: "up-a",
		}},
		[]UpstreamInput{{
			ID:             "up-a",
			SiteID:         "site-a",
			Scheme:         "http",
			Host:           "app-a",
			Port:           8080,
			BasePath:       "/",
			PassHostHeader: true,
		}},
	)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	content := ""
	for _, artifact := range artifacts {
		if artifact.Path == "nginx/nginx.conf" {
			content = string(artifact.Content)
			break
		}
	}
	if content == "" {
		t.Fatal("expected nginx main config artifact")
	}
	if !containsAll(content,
		"proxy_headers_hash_max_size 1024;",
		"proxy_headers_hash_bucket_size 128;",
	) {
		t.Fatalf("expected proxy headers hash sizing in nginx main config, got: %s", content)
	}
}
