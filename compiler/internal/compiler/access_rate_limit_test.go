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
		{ID: "access-b", SiteID: "site-b", DefaultAction: "allow", AllowCIDRs: []string{"10.0.0.0/8"}, DenyCIDRs: []string{"192.0.2.0/24"}},
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
