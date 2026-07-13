package main

import "testing"

func TestRequestTelemetryKeepsPublicManagementHostTraffic(t *testing.T) {
	public := parsedAccess{siteID: "waf_site_com", host: "waf.site.com", path: "/api/sites"}
	if shouldSkipRequestTelemetry(public) {
		t.Fatal("public management-host traffic must be retained for telemetry")
	}
	internal := parsedAccess{siteID: "", host: "localhost", path: "/api/sites"}
	if !shouldSkipRequestTelemetry(internal) {
		t.Fatal("internal control-plane traffic must stay excluded")
	}
}
