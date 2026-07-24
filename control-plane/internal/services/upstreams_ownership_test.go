package services

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"waf/control-plane/internal/sites"
	"waf/control-plane/internal/upstreams"
)

func TestUpstreamServiceRejectsMissingAndCrossSiteOwnership(t *testing.T) {
	root := t.TempDir()
	siteStore, err := sites.NewStore(filepath.Join(root, "sites"))
	if err != nil {
		t.Fatal(err)
	}
	upstreamStore, err := upstreams.NewStore(filepath.Join(root, "upstreams"))
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"site-a", "site-b"} {
		if _, err := siteStore.Create(sites.Site{ID: id, PrimaryHost: id + ".example.test", Enabled: true}); err != nil {
			t.Fatal(err)
		}
	}
	service := NewUpstreamService(upstreamStore, siteStore, nil)
	ctx := context.Background()
	if _, err := service.Create(ctx, upstreams.Upstream{ID: "missing", SiteID: "no-site", Host: "backend", Port: 80, Scheme: "http"}); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("missing site must fail, got %v", err)
	}
	created, err := service.Create(ctx, upstreams.Upstream{ID: "shared", SiteID: "site-a", Host: "backend", Port: 80, Scheme: "http"})
	if err != nil {
		t.Fatal(err)
	}
	created.SiteID = "site-b"
	if _, err := service.Update(ctx, created); err == nil || !strings.Contains(err.Error(), "belongs to site site-a") {
		t.Fatalf("cross-site move must fail, got %v", err)
	}
	items, err := upstreamStore.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].SiteID != "site-a" {
		t.Fatalf("failed ownership validation changed upstream: %+v", items)
	}
}
