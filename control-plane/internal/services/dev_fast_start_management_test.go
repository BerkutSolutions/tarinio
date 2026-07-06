package services

import (
	"context"
	"testing"

	"waf/control-plane/internal/certificatematerials"
	"waf/control-plane/internal/certificates"
	"waf/control-plane/internal/config"
	"waf/control-plane/internal/easysiteprofiles"
	"waf/control-plane/internal/sites"
	"waf/control-plane/internal/tlsconfigs"
	"waf/control-plane/internal/upstreams"
)

type dfsTestEasyStore struct {
	profile easysiteprofiles.EasySiteProfile
	ok      bool
}

func (s *dfsTestEasyStore) Get(siteID string) (easysiteprofiles.EasySiteProfile, bool, error) {
	return s.profile, s.ok, nil
}

type dfsTestEasyProfiles struct {
	upserted []easysiteprofiles.EasySiteProfile
}

func (s *dfsTestEasyProfiles) Upsert(ctx context.Context, profile easysiteprofiles.EasySiteProfile) (easysiteprofiles.EasySiteProfile, error) {
	s.upserted = append(s.upserted, profile)
	return profile, nil
}

type dfsTestSites struct{}

func (s *dfsTestSites) List() ([]sites.Site, error) { return []sites.Site{{ID: "localhost", PrimaryHost: "localhost", Enabled: true}}, nil }
func (s *dfsTestSites) Create(ctx context.Context, site sites.Site) (sites.Site, error) { return site, nil }
func (s *dfsTestSites) Update(ctx context.Context, site sites.Site) (sites.Site, error) { return site, nil }

type dfsTestUpstreams struct{}

func (s *dfsTestUpstreams) List() ([]upstreams.Upstream, error) {
	return []upstreams.Upstream{{ID: "localhost-upstream", SiteID: "localhost", Host: "ui", Port: 80, Scheme: "http"}}, nil
}
func (s *dfsTestUpstreams) Create(ctx context.Context, item upstreams.Upstream) (upstreams.Upstream, error) { return item, nil }
func (s *dfsTestUpstreams) Update(ctx context.Context, item upstreams.Upstream) (upstreams.Upstream, error) { return item, nil }

type dfsTestCertificates struct{}

func (s *dfsTestCertificates) List() ([]certificates.Certificate, error) {
	return []certificates.Certificate{{ID: "localhost-cert", CommonName: "localhost", Status: "active"}}, nil
}
func (s *dfsTestCertificates) Update(ctx context.Context, item certificates.Certificate) (certificates.Certificate, error) {
	return item, nil
}

type dfsTestMaterials struct{}

func (s *dfsTestMaterials) Read(certificateID string) (certificatematerials.MaterialRecord, []byte, []byte, error) {
	return certificatematerials.MaterialRecord{CertificateID: certificateID}, []byte("cert"), []byte("key"), nil
}

type dfsTestTLSConfigs struct{}

func (s *dfsTestTLSConfigs) List() ([]tlsconfigs.TLSConfig, error) {
	return []tlsconfigs.TLSConfig{{SiteID: "localhost", CertificateID: "localhost-cert"}}, nil
}
func (s *dfsTestTLSConfigs) Create(ctx context.Context, item tlsconfigs.TLSConfig) (tlsconfigs.TLSConfig, error) { return item, nil }
func (s *dfsTestTLSConfigs) Update(ctx context.Context, item tlsconfigs.TLSConfig) (tlsconfigs.TLSConfig, error) { return item, nil }

func TestNeedsDevFastStartEasyProfileUpdate_DetectsSecurityDrift(t *testing.T) {
	t.Setenv("CONTROL_PLANE_DEV_FAST_START_MANAGEMENT_SITE_ID", "localhost")
	desired := defaultDevFastStartEasyProfile("localhost", "localhost")
	current := desired
	current.FrontService.SecurityMode = easysiteprofiles.SecurityModeBlock
	current.SecurityAntibot.AntibotChallenge = easysiteprofiles.AntibotChallengeCaptcha
	current.SecurityModSecurity.UseModSecurity = true
	current.SecurityModSecurity.UseModSecurityCRSPlugins = true

	if !needsDevFastStartEasyProfileUpdate(current, desired) {
		t.Fatal("expected dev fast start updater to detect management security drift")
	}
}

func TestDevFastStartUpdatesExistingProfileSecurityFields(t *testing.T) {
	t.Setenv("CONTROL_PLANE_DEV_FAST_START_MANAGEMENT_SITE_ID", "localhost")
	current := easysiteprofiles.DefaultProfile("localhost")
	current.FrontService.ServerName = "localhost"
	current.FrontService.SecurityMode = easysiteprofiles.SecurityModeBlock
	current.SecurityAntibot.AntibotChallenge = easysiteprofiles.AntibotChallengeCaptcha
	current.SecurityModSecurity.UseModSecurity = true
	current.SecurityModSecurity.UseModSecurityCRSPlugins = true

	easyStore := &dfsTestEasyStore{profile: current, ok: true}
	easyProfiles := &dfsTestEasyProfiles{}
	bootstrapper := &DevFastStartBootstrapper{
		cfg: config.DevFastStartConfig{
			Enabled:          true,
			Host:             "localhost",
			ManagementSiteID: "localhost",
			CertificateID:    "localhost-cert",
			UpstreamHost:     "ui",
			UpstreamPort:     80,
			MaxAttempts:      1,
		},
		sites:        &dfsTestSites{},
		upstreams:    &dfsTestUpstreams{},
		certificates: &dfsTestCertificates{},
		materials:    &dfsTestMaterials{},
		tlsConfigs:   &dfsTestTLSConfigs{},
		easyStore:    easyStore,
		easyProfiles: easyProfiles,
	}

	changed, err := bootstrapper.ensureManagementResources(context.Background())
	if err != nil {
		t.Fatalf("ensureManagementResources: %v", err)
	}
	if !changed {
		t.Fatal("expected bootstrapper to rewrite drifted localhost profile")
	}
	if len(easyProfiles.upserted) != 1 {
		t.Fatalf("expected one upserted profile, got %d", len(easyProfiles.upserted))
	}
	updated := easyProfiles.upserted[0]
	if updated.FrontService.SecurityMode != easysiteprofiles.SecurityModeTransparent {
		t.Fatalf("expected transparent security mode, got %q", updated.FrontService.SecurityMode)
	}
	if updated.SecurityAntibot.AntibotChallenge != easysiteprofiles.AntibotChallengeNo {
		t.Fatalf("expected antibot challenge %q, got %q", easysiteprofiles.AntibotChallengeNo, updated.SecurityAntibot.AntibotChallenge)
	}
	if updated.SecurityModSecurity.UseModSecurity {
		t.Fatal("expected modsecurity to be disabled for dev fast start localhost")
	}
	if updated.SecurityModSecurity.UseModSecurityCRSPlugins {
		t.Fatal("expected CRS plugins to be disabled for dev fast start localhost")
	}
}
