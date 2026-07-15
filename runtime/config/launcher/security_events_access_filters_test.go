package main

import "testing"

func TestRequestTelemetrySkipsManagementHostButKeepsProtectedServicePaths(t *testing.T) {
	management := parsedAccess{siteID: "waf_site_com", host: "waf.site.com", path: "/api/sites", management: true}
	if !shouldSkipRequestTelemetry(management) {
		t.Fatal("management-host traffic must not enter product telemetry")
	}
	if shouldTrackRequestBurst(management) {
		t.Fatal("management-host traffic must not affect burst telemetry")
	}
	protected := parsedAccess{siteID: "customer_site", host: "customer.example", path: "/api/sites"}
	if shouldSkipRequestTelemetry(protected) {
		t.Fatal("a protected service path must not be filtered by its name alone")
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
