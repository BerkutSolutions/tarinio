package compiler

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAssembleRevisionBundle_Deterministic(t *testing.T) {
	siteArtifacts, err := RenderSiteUpstreamArtifacts(
		[]SiteInput{
			{
				ID:                "site-a",
				Enabled:           true,
				PrimaryHost:       "a.example.com",
				ListenHTTP:        true,
				ListenHTTPS:       true,
				DefaultUpstreamID: "up-a",
			},
		},
		[]UpstreamInput{
			{
				ID:             "up-a",
				SiteID:         "site-a",
				Scheme:         "http",
				Host:           "app-a",
				Port:           8080,
				BasePath:       "/",
				PassHostHeader: true,
			},
		},
	)
	if err != nil {
		t.Fatalf("render site/upstream failed: %v", err)
	}

	tlsArtifacts, err := RenderTLSArtifacts(
		[]SiteInput{
			{
				ID:                "site-a",
				Enabled:           true,
				PrimaryHost:       "a.example.com",
				ListenHTTPS:       true,
				DefaultUpstreamID: "up-a",
			},
		},
		[]TLSConfigInput{
			{ID: "tls-a", SiteID: "site-a", CertificateID: "cert-a"},
		},
		[]CertificateInput{
			{ID: "cert-a", SiteID: "site-a", StorageRef: "/certs/a.crt", PrivateKeyRef: "/certs/a.key"},
		},
	)
	if err != nil {
		t.Fatalf("render tls failed: %v", err)
	}

	wafArtifacts, err := RenderWAFArtifacts(
		[]SiteInput{
			{ID: "site-a", Enabled: true, PrimaryHost: "a.example.com", ListenHTTP: true, DefaultUpstreamID: "up-a"},
		},
		[]WAFPolicyInput{
			{ID: "waf-a", SiteID: "site-a", Enabled: true, Mode: WAFModePrevention, CRSEnabled: true},
		},
	)
	if err != nil {
		t.Fatalf("render waf failed: %v", err)
	}

	accessArtifacts, err := RenderAccessRateLimitArtifacts(
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
		t.Fatalf("render access/rate failed: %v", err)
	}

	revision := RevisionInput{
		ID:        "rev-001",
		Version:   1,
		CreatedAt: "2026-03-31T12:00:00Z",
	}

	first, err := AssembleRevisionBundle(revision, siteArtifacts, tlsArtifacts, wafArtifacts, accessArtifacts)
	if err != nil {
		t.Fatalf("first assembly failed: %v", err)
	}
	second, err := AssembleRevisionBundle(revision, siteArtifacts, tlsArtifacts, wafArtifacts, accessArtifacts)
	if err != nil {
		t.Fatalf("second assembly failed: %v", err)
	}

	if first.Manifest.BundleChecksum != second.Manifest.BundleChecksum {
		t.Fatal("bundle checksum must be deterministic")
	}
	if len(first.Files) != len(second.Files) {
		t.Fatalf("bundle file count differs: %d vs %d", len(first.Files), len(second.Files))
	}
	for i := range first.Files {
		if first.Files[i].Path != second.Files[i].Path {
			t.Fatalf("bundle path differs at %d: %s vs %s", i, first.Files[i].Path, second.Files[i].Path)
		}
		if string(first.Files[i].Content) != string(second.Files[i].Content) {
			t.Fatalf("bundle file content differs for %s", first.Files[i].Path)
		}
	}
}

func TestAssembleRevisionBundle_SelfContainedManifest(t *testing.T) {
	bundle, err := AssembleRevisionBundle(
		RevisionInput{
			ID:        "rev-001",
			Version:   1,
			CreatedAt: "2026-03-31T12:00:00Z",
		},
		[]ArtifactOutput{
			newArtifact("nginx/nginx.conf", ArtifactKindNginxConfig, []byte("worker_processes auto;\n")),
			newArtifact("modsecurity/modsecurity.conf", ArtifactKindModSecurity, []byte("SecRuleEngine Off\n")),
		},
	)
	if err != nil {
		t.Fatalf("assembly failed: %v", err)
	}

	if bundle.Files[len(bundle.Files)-1].Path != "manifest.json" {
		t.Fatal("expected manifest.json to be part of the bundle")
	}

	var manifest RevisionManifest
	if err := json.Unmarshal(bundle.Files[len(bundle.Files)-1].Content, &manifest); err != nil {
		t.Fatalf("unmarshal manifest failed: %v", err)
	}

	if manifest.RevisionID != "rev-001" || manifest.RevisionVersion != 1 {
		t.Fatal("manifest revision identity mismatch")
	}
	if len(manifest.Contents) != 2 {
		t.Fatalf("expected 2 manifest contents, got %d", len(manifest.Contents))
	}
	for _, entry := range manifest.Contents {
		if strings.Contains(entry.Path, "Site{") {
			t.Fatal("manifest must not contain domain model data")
		}
	}
}

func TestAssembleRevisionBundle_RejectsDuplicateArtifactPath(t *testing.T) {
	_, err := AssembleRevisionBundle(
		RevisionInput{
			ID:        "rev-001",
			Version:   1,
			CreatedAt: "2026-03-31T12:00:00Z",
		},
		[]ArtifactOutput{
			newArtifact("nginx/nginx.conf", ArtifactKindNginxConfig, []byte("a")),
			newArtifact("nginx/nginx.conf", ArtifactKindNginxConfig, []byte("b")),
		},
	)
	if err == nil {
		t.Fatal("expected duplicate artifact path error")
	}
}
