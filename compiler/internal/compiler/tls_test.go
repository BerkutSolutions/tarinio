package compiler

import "testing"

func TestRenderTLSArtifacts_Deterministic(t *testing.T) {
	sites := []SiteInput{
		{
			ID:                "site-b",
			Enabled:           true,
			PrimaryHost:       "b.example.com",
			ListenHTTPS:       true,
			DefaultUpstreamID: "up-b",
		},
		{
			ID:                "site-a",
			Enabled:           true,
			PrimaryHost:       "a.example.com",
			ListenHTTPS:       true,
			DefaultUpstreamID: "up-a",
		},
	}

	tlsConfigs := []TLSConfigInput{
		{ID: "tls-b", SiteID: "site-b", CertificateID: "cert-b"},
		{ID: "tls-a", SiteID: "site-a", CertificateID: "cert-a"},
	}

	certs := []CertificateInput{
		{ID: "cert-b", SiteID: "site-b", StorageRef: "/certs/b.crt", PrivateKeyRef: "/certs/b.key"},
		{ID: "cert-a", SiteID: "site-a", StorageRef: "/certs/a.crt", PrivateKeyRef: "/certs/a.key"},
	}

	first, err := RenderTLSArtifacts(sites, tlsConfigs, certs)
	if err != nil {
		t.Fatalf("first render failed: %v", err)
	}
	second, err := RenderTLSArtifacts(sites, tlsConfigs, certs)
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
		if first[i].Kind != ArtifactKindTLSRef {
			t.Fatalf("unexpected artifact kind for %s: %s", first[i].Path, first[i].Kind)
		}
	}
}

func TestRenderTLSArtifacts_RequiresTLSConfigForHTTPS(t *testing.T) {
	_, err := RenderTLSArtifacts(
		[]SiteInput{
			{
				ID:                "site-a",
				Enabled:           true,
				PrimaryHost:       "a.example.com",
				ListenHTTPS:       true,
				DefaultUpstreamID: "up-a",
			},
		},
		nil,
		nil,
	)
	if err == nil {
		t.Fatal("expected error when HTTPS site has no TLS config")
	}
}
