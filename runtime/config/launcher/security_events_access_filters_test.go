package main

import "testing"

func TestRequestTelemetryKeepsPublicManagementHostTraffic(t *testing.T) {
	public := parsedAccess{siteID: "waf_site_com", host: "waf.site.com", path: "/api/sites"}
	if shouldSkipRequestTelemetry(public) {
		t.Fatal("public management-host traffic must be retained for telemetry")
	}
	for _, asset := range []parsedAccess{
		{siteID: "waf_site_com", host: "waf.site.com", path: "/static/icons/svg/dashboard-32x32.svg"},
		{siteID: "waf_site_com", host: "waf.site.com", path: "/favicon.ico"},
	} {
		if !shouldSkipRequestTelemetry(asset) {
			t.Fatalf("static asset %q must not enter request telemetry", asset.path)
		}
	}
	internal := parsedAccess{siteID: "", host: "localhost", path: "/api/sites"}
	if !shouldSkipRequestTelemetry(internal) {
		t.Fatal("internal control-plane traffic must stay excluded")
	}
}
