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
	blockedManagement := management
	blockedManagement.status = 403
	if shouldSkipRequestTelemetry(blockedManagement) {
		t.Fatal("blocked management-host request must be retained for security diagnostics")
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

func TestBlockedManagementRequestIsMarkedAsSecurityRecord(t *testing.T) {
	record := newRequestLogRecord(parsedAccess{status: 403, securityReason: "modsecurity"})
	if record.RowType != "security" || record.SecurityReason != "modsecurity" {
		t.Fatalf("blocked management record must retain its security classification, got %+v", record)
	}
}

func TestRequestRecordUsesExactAuthAndAntibotReasons(t *testing.T) {
	for _, item := range []struct {
		path   string
		status int
		want   string
	}{
		{path: "/auth/verify/basic", status: 401, want: "auth"},
		{path: "/challenge/verify", status: 400, want: "antibot"},
		{path: "/checkout", status: 403, want: "access_blocked"},
	} {
		record := newRequestLogRecord(parsedAccess{path: item.path, status: item.status})
		if record.RowType != "security" || record.SecurityReason != item.want {
			t.Fatalf("%s status=%d: got %+v", item.path, item.status, record)
		}
	}
}
