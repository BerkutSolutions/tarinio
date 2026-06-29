package compiler

import (
	"strings"
	"testing"
)

func TestRenderAccessRateLimitArtifacts_Deterministic(t *testing.T) {
	sites := []SiteInput{
		{ID: "site-b", Enabled: true, PrimaryHost: "b.example.com", ListenHTTP: true, DefaultUpstreamID: "up-b"},
		{ID: "site-a", Enabled: true, PrimaryHost: "a.example.com", ListenHTTP: true, DefaultUpstreamID: "up-a"},
	}

	accessPolicies := []AccessPolicyInput{
		{ID: "access-b", SiteID: "site-b", DefaultAction: "allow", AllowCIDRs: []string{"10.0.0.0/8"}, DenyCIDRs: []string{"192.0.3.0/24"}},
		{ID: "access-a", SiteID: "site-a", DefaultAction: "deny", AllowCIDRs: []string{"203.0.113.0/24"}, TrustedProxyCIDRs: []string{"172.16.0.0/12", "172.16.0.0/12"}},
	}

	ratePolicies := []RateLimitPolicyInput{
		{ID: "rate-b", SiteID: "site-b", Enabled: true, Requests: 60, WindowSeconds: 60, Burst: 10, StatusCode: 429},
		{ID: "rate-a", SiteID: "site-a", Enabled: true, Requests: 10, WindowSeconds: 1, Burst: 5, StatusCode: 429},
	}

	first, err := RenderAccessRateLimitArtifacts(sites, accessPolicies, ratePolicies)
	if err != nil {
		t.Fatalf("first render failed: %v", err)
	}
	second, err := RenderAccessRateLimitArtifacts(sites, accessPolicies, ratePolicies)
	if err != nil {
		t.Fatalf("second render failed: %v", err)
	}

	if len(first) != len(second) {
		t.Fatalf("artifact counts differ: %d vs %d", len(first), len(second))
	}

	for i := range first {
		if first[i].Path != second[i].Path {
			t.Fatalf("artifact path differs at %d: %s vs %s", i, first[i].Path, second[i].Path)
		}
		if first[i].Checksum != second[i].Checksum {
			t.Fatalf("artifact checksum differs for %s", first[i].Path)
		}
		if strings.TrimSpace(string(first[i].Content)) == "" {
			t.Fatalf("empty content for %s", first[i].Path)
		}
	}
}

func TestRenderAccessRateLimitArtifacts_ManifestCompatibleKinds(t *testing.T) {
	artifacts, err := RenderAccessRateLimitArtifacts(
		[]SiteInput{
			{ID: "site-a", Enabled: true, PrimaryHost: "a.example.com", ListenHTTP: true, DefaultUpstreamID: "up-a"},
		},
		[]AccessPolicyInput{
			{ID: "access-a", SiteID: "site-a", DefaultAction: "deny", AllowCIDRs: []string{"203.0.113.0/24"}},
		},
		[]RateLimitPolicyInput{
			{ID: "rate-a", SiteID: "site-a", Enabled: true, Requests: 120, WindowSeconds: 60, Burst: 20, StatusCode: 429},
		},
	)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	if len(artifacts) != 3 {
		t.Fatalf("expected 3 access/rate artifacts, got %d", len(artifacts))
	}

	for _, artifact := range artifacts {
		if artifact.Kind != ArtifactKindNginxConfig {
			t.Fatalf("unexpected artifact kind for %s: %s", artifact.Path, artifact.Kind)
		}
		if artifact.Checksum == "" {
			t.Fatalf("missing checksum for %s", artifact.Path)
		}
	}
}

func TestRenderAccessRateLimitArtifacts_RejectsInvalidRatePolicy(t *testing.T) {
	_, err := RenderAccessRateLimitArtifacts(
		[]SiteInput{
			{ID: "site-a", Enabled: true, PrimaryHost: "a.example.com", ListenHTTP: true, DefaultUpstreamID: "up-a"},
		},
		nil,
		[]RateLimitPolicyInput{
			{ID: "rate-a", SiteID: "site-a", Enabled: true, Requests: 0, WindowSeconds: 60, Burst: 10},
		},
	)
	if err == nil {
		t.Fatal("expected error for invalid rate limit policy")
	}
}

func TestRenderAccessRateLimitArtifacts_AllowlistKeepsRateLimitKeying(t *testing.T) {
	artifacts, err := RenderAccessRateLimitArtifacts(
		[]SiteInput{
			{ID: "site-a", Enabled: true, PrimaryHost: "a.example.com", ListenHTTP: true, DefaultUpstreamID: "up-a"},
		},
		[]AccessPolicyInput{
			{ID: "access-a", SiteID: "site-a", DefaultAction: "allow", AllowCIDRs: []string{"10.0.0.1", "10.0.0.0/24"}},
		},
		[]RateLimitPolicyInput{
			{ID: "rate-a", SiteID: "site-a", Enabled: true, Requests: 120, WindowSeconds: 60, Burst: 20, StatusCode: 429},
		},
	)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	var httpConf string
	for _, artifact := range artifacts {
		if artifact.Path == "nginx/conf.d/ratelimits.conf" {
			httpConf = string(artifact.Content)
			break
		}
	}
	if httpConf == "" {
		t.Fatal("expected ratelimits.conf artifact")
	}
	if !strings.Contains(httpConf, "geo $waf_allow_bypass_site_a {") {
		t.Fatalf("expected allowlist bypass geo in ratelimits.conf, got: %s", httpConf)
	}
	if !strings.Contains(httpConf, "10.0.0.0/24 1;") {
		t.Fatalf("expected allowlist CIDR in bypass geo, got: %s", httpConf)
	}
	if !strings.Contains(httpConf, "map $waf_allow_bypass_site_a $waf_rate_limit_key_site_a {") {
		t.Fatalf("expected allowlist rate key map in ratelimits.conf, got: %s", httpConf)
	}
	if !strings.Contains(httpConf, "1 \"${binary_remote_addr}:allow\";") {
		t.Fatalf("expected allowlisted clients to keep dedicated rate-limit key, got: %s", httpConf)
	}
	if !strings.Contains(httpConf, "limit_req_zone $waf_rate_limit_key_site_a zone=site_site-a_req:10m rate=120r/m;") {
		t.Fatalf("expected rate limit key variable in ratelimits.conf, got: %s", httpConf)
	}
}

func TestRenderAccessRateLimitArtifacts_ManagementAllowlistDefaultsToDeny(t *testing.T) {
	artifacts, err := RenderAccessRateLimitArtifacts(
		[]SiteInput{
			{ID: "control-plane-access", Enabled: true, PrimaryHost: "waf.example.com", ListenHTTP: true, DefaultUpstreamID: "up-a"},
		},
		[]AccessPolicyInput{
			{ID: "access-a", SiteID: "control-plane-access", DefaultAction: "allow", AllowCIDRs: []string{"10.0.0.0/24"}},
		},
		nil,
	)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	var accessConf string
	for _, artifact := range artifacts {
		if artifact.Path == "nginx/access/control-plane-access.conf" {
			accessConf = string(artifact.Content)
			break
		}
	}
	if !strings.Contains(accessConf, "deny all;") {
		t.Fatalf("expected management site allowlist to imply default deny, got: %s", accessConf)
	}
}

func TestRenderAccessRateLimitArtifacts_LimitReqURLScopesDirective(t *testing.T) {
	artifacts, err := RenderAccessRateLimitArtifacts(
		[]SiteInput{
			{ID: "site-a", Enabled: true, PrimaryHost: "a.example.com", ListenHTTP: true, DefaultUpstreamID: "up-a"},
		},
		nil,
		[]RateLimitPolicyInput{
			{ID: "rate-a", SiteID: "site-a", Enabled: true, Requests: 30, WindowSeconds: 1, Burst: 30, StatusCode: 429, LimitReqURL: "/login"},
		},
	)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	var siteConf string
	for _, artifact := range artifacts {
		if artifact.Path == "nginx/ratelimits/site-a.conf" {
			siteConf = string(artifact.Content)
			break
		}
	}
	if siteConf == "" {
		t.Fatal("expected nginx/ratelimits/site-a.conf artifact")
	}
	// limit_req stays in server context; URL scoping is handled by an http-level
	// map so generated nginx remains valid.
	if strings.Contains(siteConf, "if ($uri") {
		t.Fatalf("limit_req must not be wrapped in an nginx if-block, got: %s", siteConf)
	}
	if !strings.Contains(siteConf, "limit_req zone=site_site-a_req") {
		t.Fatalf("expected limit_req directive, got: %s", siteConf)
	}
	// limit_conn must remain unconditional (connection limit is site-wide)
	if !strings.Contains(siteConf, "limit_conn site_site-a_conn") {
		t.Fatalf("expected unconditional limit_conn directive, got: %s", siteConf)
	}

	var httpConf string
	for _, artifact := range artifacts {
		if artifact.Path == "nginx/conf.d/ratelimits.conf" {
			httpConf = string(artifact.Content)
			break
		}
	}
	if !strings.Contains(httpConf, "map $uri $waf_rate_limit_req_key_site_a") {
		t.Fatalf("expected URI-scoped rate-limit key map, got: %s", httpConf)
	}
	if !strings.Contains(httpConf, `~*^/login $waf_rate_limit_key_site_a;`) {
		t.Fatalf("expected /login to feed the base rate-limit key, got: %s", httpConf)
	}
	if !strings.Contains(httpConf, "limit_req_zone $waf_rate_limit_req_key_site_a zone=site_site-a_req:10m") {
		t.Fatalf("expected limit_req_zone to use the scoped key, got: %s", httpConf)
	}
}

func TestRenderAccessRateLimitArtifacts_NoLimitReqURLAppliesGlobally(t *testing.T) {
	artifacts, err := RenderAccessRateLimitArtifacts(
		[]SiteInput{
			{ID: "site-b", Enabled: true, PrimaryHost: "b.example.com", ListenHTTP: true, DefaultUpstreamID: "up-b"},
		},
		nil,
		[]RateLimitPolicyInput{
			{ID: "rate-b", SiteID: "site-b", Enabled: true, Requests: 100, WindowSeconds: 1, Burst: 50, StatusCode: 429},
		},
	)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	var siteConf string
	for _, artifact := range artifacts {
		if artifact.Path == "nginx/ratelimits/site-b.conf" {
			siteConf = string(artifact.Content)
			break
		}
	}
	if siteConf == "" {
		t.Fatal("expected nginx/ratelimits/site-b.conf artifact")
	}
	// Without LimitReqURL, limit_req must NOT be wrapped in an if-block
	if strings.Contains(siteConf, "if ($uri") {
		t.Fatalf("expected global limit_req without if-block, got: %s", siteConf)
	}
	if !strings.Contains(siteConf, "limit_req zone=site_site-b_req") {
		t.Fatalf("expected unconditional limit_req directive, got: %s", siteConf)
	}
}
