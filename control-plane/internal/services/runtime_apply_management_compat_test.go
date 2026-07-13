package services

import (
	"testing"

	"waf/control-plane/internal/sites"
	"waf/control-plane/internal/upstreams"
)

func TestLegacyManagementUIProxySiteIDsUsesOnlyOneEnabledUIProxy(t *testing.T) {
	sites := []sites.Site{
		{ID: "retired-ip", Enabled: false},
		{ID: "panel", Enabled: true, PrimaryHost: "panel.example"},
	}
	upstreams := []upstreams.Upstream{
		{ID: "retired-upstream", SiteID: "retired-ip", Scheme: "http", Host: "ui", Port: 80},
		{ID: "panel-upstream", SiteID: "panel", Scheme: "http", Host: "ui", Port: 80},
	}

	got := legacyManagementUIProxySiteIDs(sites, upstreams, false)
	if _, ok := got["panel"]; !ok || len(got) != 1 {
		t.Fatalf("expected the sole enabled ui:80 proxy to be management, got %#v", got)
	}
}

func TestLegacyManagementUIProxySiteIDsDoesNotGuessWhenExplicitOrAmbiguous(t *testing.T) {
	sites := []sites.Site{{ID: "panel-a", Enabled: true}, {ID: "panel-b", Enabled: true}}
	upstreams := []upstreams.Upstream{
		{ID: "a", SiteID: "panel-a", Scheme: "http", Host: "ui", Port: 80},
		{ID: "b", SiteID: "panel-b", Scheme: "http", Host: "ui", Port: 80},
	}
	if got := legacyManagementUIProxySiteIDs(sites, upstreams, false); len(got) != 0 {
		t.Fatalf("ambiguous UI proxies must not be guessed, got %#v", got)
	}
	if got := legacyManagementUIProxySiteIDs(sites[:1], upstreams[:1], true); len(got) != 0 {
		t.Fatalf("explicit configuration must disable compatibility inference, got %#v", got)
	}
}
