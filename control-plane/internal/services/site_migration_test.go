package services

import (
	"context"
	"testing"

	"waf/control-plane/internal/accesspolicies"
	"waf/control-plane/internal/certificatematerials"
	"waf/control-plane/internal/certificates"
	"waf/control-plane/internal/easysiteprofiles"
	"waf/control-plane/internal/managementhosts"
	"waf/control-plane/internal/ratelimitpolicies"
	"waf/control-plane/internal/sites"
	"waf/control-plane/internal/tlsconfigs"
	"waf/control-plane/internal/upstreams"
	"waf/control-plane/internal/virtualpatches"
	"waf/control-plane/internal/wafpolicies"
)

func TestSiteMigrationMovesIPIdentityToDomainAndRemovesOldBindings(t *testing.T) {
	root := t.TempDir()
	siteStore, _ := sites.NewStore(root+"/sites")
	upstreamStore, _ := upstreams.NewStore(root+"/upstreams")
	certificateStore, _ := certificates.NewStore(root+"/certificates")
	materialStore, _ := certificatematerials.NewStore(root+"/materials")
	tlsStore, _ := tlsconfigs.NewStore(root+"/tls")
	wafStore, _ := wafpolicies.NewStore(root+"/waf")
	accessStore, _ := accesspolicies.NewStore(root+"/access")
	rateStore, _ := ratelimitpolicies.NewStore(root+"/rate")
	easyStore, _ := easysiteprofiles.NewStore(root+"/easy")
	patchStore, _ := virtualpatches.NewStore(root+"/patches")
	managementStore, _ := managementhosts.NewStore(root+"/management", nil)
	oldID, newID := "198.51.100.54", "waf.site.com"
	if _, err := siteStore.Create(sites.Site{ID: oldID, PrimaryHost: oldID, Enabled: true}); err != nil { t.Fatal(err) }
	if _, err := upstreamStore.Create(upstreams.Upstream{ID: "panel-upstream", SiteID: oldID, Host: "ui", Port: 80, Scheme: "http"}); err != nil { t.Fatal(err) }
	if _, err := certificateStore.Create(certificates.Certificate{ID: oldID + "-tls", CommonName: oldID, Status: "active"}); err != nil { t.Fatal(err) }
	if _, err := materialStore.Put(oldID+"-tls", []byte("certificate"), []byte("private-key")); err != nil { t.Fatal(err) }
	if _, err := tlsStore.Create(tlsconfigs.TLSConfig{SiteID: oldID, CertificateID: oldID + "-tls"}); err != nil { t.Fatal(err) }
	if _, err := easyStore.Create(easysiteprofiles.DefaultProfile(oldID)); err != nil { t.Fatal(err) }

	migration := NewSiteMigration(siteStore, upstreamStore, certificateStore, materialStore, tlsStore, wafStore, accessStore, rateStore, easyStore, patchStore, managementStore)
	if _, err := migration.Rename(context.Background(), oldID, sites.Site{ID: newID, PrimaryHost: newID, Enabled: true}); err != nil { t.Fatal(err) }
	items, _ := siteStore.List(); if len(items) != 1 || items[0].ID != newID { t.Fatalf("unexpected sites: %+v", items) }
	upstreams, _ := upstreamStore.List(); if len(upstreams) != 1 || upstreams[0].SiteID != newID { t.Fatalf("upstream was not moved: %+v", upstreams) }
	tlsItems, _ := tlsStore.List(); if len(tlsItems) != 1 || tlsItems[0].SiteID != newID || tlsItems[0].CertificateID != newID+"-tls" { t.Fatalf("TLS was not moved: %+v", tlsItems) }
	if _, _, _, err := materialStore.Read(newID+"-tls"); err != nil { t.Fatalf("new certificate material missing: %v", err) }
	if _, _, _, err := materialStore.Read(oldID+"-tls"); err == nil { t.Fatal("old certificate material must be removed") }
	profiles, _ := easyStore.List(); if len(profiles) != 1 || profiles[0].SiteID != newID { t.Fatalf("easy profile was not moved: %+v", profiles) }
	runtimeSites, runtimeUpstreams := mapSiteUpstreamInputs(items, upstreams, tlsItems, profiles, nil, true)
	if len(runtimeSites) != 1 || runtimeSites[0].ID != newID || len(runtimeUpstreams) != 1 || runtimeUpstreams[0].SiteID != newID {
		t.Fatalf("runtime inputs retained the retired identity: sites=%+v upstreams=%+v", runtimeSites, runtimeUpstreams)
	}
	certs, _ := certificateStore.List(); if len(certs) != 1 { t.Fatalf("unexpected certificates: %+v", certs) }
	certificateUpdatedAt := certs[0].UpdatedAt
	service := NewSiteService(siteStore, nil)
	if _, err := service.Update(withAutoApplyDisabled(context.Background()), sites.Site{ID: newID, PrimaryHost: newID, Enabled: true}); err != nil { t.Fatal(err) }
	certs, _ = certificateStore.List()
	if len(certs) != 1 || certs[0].ID != newID+"-tls" || certs[0].UpdatedAt != certificateUpdatedAt {
		t.Fatalf("ordinary save must not recreate or renew the migrated certificate: %+v", certs)
	}
}
