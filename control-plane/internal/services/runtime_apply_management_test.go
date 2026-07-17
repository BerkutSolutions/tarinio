package services

import (
	"testing"

	"waf/compiler/pipeline"
	"waf/control-plane/internal/antiddos"
)

func TestApplyAntiDDoSRateOverrides_SkipsConfiguredManagementSite(t *testing.T) {
	t.Setenv("CONTROL_PLANE_DEV_FAST_START_MANAGEMENT_SITE_ID", "ui.example.test")

	items := applyAntiDDoSRateOverrides(
		[]pipeline.SiteInput{
			{ID: "ui.example.test", Enabled: true},
			{ID: "site-a", Enabled: true},
		},
		nil,
		antiddos.Settings{
			EnforceL7Rate: true,
			L7RequestsPS:  120,
			L7Burst:       240,
			L7StatusCode:  429,
		},
	)

	for _, item := range items {
		if item.SiteID == "ui.example.test" {
			t.Fatalf("did not expect anti-ddos override for configured management site: %+v", item)
		}
	}
	if len(items) != 1 || items[0].SiteID != "site-a" {
		t.Fatalf("expected only non-management site override, got %+v", items)
	}
}

func TestApplyAntiDDoSRateOverrides_SkipsExplicitManagementSiteWithCustomID(t *testing.T) {
	items := applyAntiDDoSRateOverrides(
		[]pipeline.SiteInput{
			{ID: "easy-waf.example.test", Enabled: true, Management: true},
			{ID: "site-a", Enabled: true},
		},
		nil,
		antiddos.Settings{
			EnforceL7Rate: true,
			L7RequestsPS:  120,
			L7Burst:       240,
			L7StatusCode:  429,
		},
	)

	if len(items) != 1 || items[0].SiteID != "site-a" {
		t.Fatalf("expected only non-management site override, got %+v", items)
	}
}
