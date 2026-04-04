package services

import (
	"context"
	"fmt"
	"testing"

	"waf/control-plane/internal/certificatematerials"
	"waf/control-plane/internal/certificates"
	"waf/control-plane/internal/config"
	"waf/control-plane/internal/easysiteprofiles"
	"waf/control-plane/internal/jobs"
	"waf/control-plane/internal/ratelimitpolicies"
	"waf/control-plane/internal/revisions"
	"waf/control-plane/internal/sites"
	"waf/control-plane/internal/tlsconfigs"
	"waf/control-plane/internal/upstreams"
)

type fakeDevFastStartSites struct {
	items []sites.Site
}

func (f *fakeDevFastStartSites) List() ([]sites.Site, error) {
	return append([]sites.Site(nil), f.items...), nil
}
func (f *fakeDevFastStartSites) Create(ctx context.Context, site sites.Site) (sites.Site, error) {
	f.items = append(f.items, site)
	return site, nil
}
func (f *fakeDevFastStartSites) Update(ctx context.Context, site sites.Site) (sites.Site, error) {
	for i := range f.items {
		if f.items[i].ID == site.ID {
			f.items[i] = site
			return site, nil
		}
	}
	return sites.Site{}, fmt.Errorf("site %s not found", site.ID)
}

type fakeDevFastStartUpstreams struct {
	items []upstreams.Upstream
}

func (f *fakeDevFastStartUpstreams) List() ([]upstreams.Upstream, error) {
	return append([]upstreams.Upstream(nil), f.items...), nil
}
func (f *fakeDevFastStartUpstreams) Create(ctx context.Context, item upstreams.Upstream) (upstreams.Upstream, error) {
	f.items = append(f.items, item)
	return item, nil
}
func (f *fakeDevFastStartUpstreams) Update(ctx context.Context, item upstreams.Upstream) (upstreams.Upstream, error) {
	for i := range f.items {
		if f.items[i].ID == item.ID {
			f.items[i] = item
			return item, nil
		}
	}
	return upstreams.Upstream{}, fmt.Errorf("upstream %s not found", item.ID)
}

type fakeDevFastStartCertificates struct {
	items []certificates.Certificate
}

func (f *fakeDevFastStartCertificates) List() ([]certificates.Certificate, error) {
	return append([]certificates.Certificate(nil), f.items...), nil
}
func (f *fakeDevFastStartCertificates) Update(ctx context.Context, item certificates.Certificate) (certificates.Certificate, error) {
	for i := range f.items {
		if f.items[i].ID == item.ID {
			f.items[i] = item
			return item, nil
		}
	}
	return certificates.Certificate{}, fmt.Errorf("certificate %s not found", item.ID)
}

type fakeDevFastStartMaterials struct {
	exists bool
}

func (f *fakeDevFastStartMaterials) Read(certificateID string) (certificatematerials.MaterialRecord, []byte, []byte, error) {
	if !f.exists {
		return certificatematerials.MaterialRecord{}, nil, nil, fmt.Errorf("certificate material %s not found", certificateID)
	}
	return certificatematerials.MaterialRecord{CertificateID: certificateID}, []byte("CERT"), []byte("KEY"), nil
}

type fakeDevFastStartTLSConfigs struct {
	items []tlsconfigs.TLSConfig
}

func (f *fakeDevFastStartTLSConfigs) List() ([]tlsconfigs.TLSConfig, error) {
	return append([]tlsconfigs.TLSConfig(nil), f.items...), nil
}
func (f *fakeDevFastStartTLSConfigs) Create(ctx context.Context, item tlsconfigs.TLSConfig) (tlsconfigs.TLSConfig, error) {
	f.items = append(f.items, item)
	return item, nil
}
func (f *fakeDevFastStartTLSConfigs) Update(ctx context.Context, item tlsconfigs.TLSConfig) (tlsconfigs.TLSConfig, error) {
	for i := range f.items {
		if f.items[i].SiteID == item.SiteID {
			f.items[i] = item
			return item, nil
		}
	}
	return tlsconfigs.TLSConfig{}, fmt.Errorf("tls config %s not found", item.SiteID)
}

type fakeDevFastStartRevisions struct {
	active bool
}

func (f *fakeDevFastStartRevisions) CurrentActive() (revisions.Revision, bool, error) {
	if !f.active {
		return revisions.Revision{}, false, nil
	}
	return revisions.Revision{ID: "rev-000001"}, true, nil
}

type fakeDevFastStartCompile struct {
	calls int
}

func (f *fakeDevFastStartCompile) Create(ctx context.Context) (CompileRequestResult, error) {
	f.calls++
	return CompileRequestResult{
		Revision: revisions.Revision{ID: fmt.Sprintf("rev-%06d", f.calls)},
		Job:      jobs.Job{ID: fmt.Sprintf("compile-rev-%06d", f.calls), Status: jobs.StatusSucceeded},
	}, nil
}

type fakeDevFastStartApply struct {
	calls      int
	failBefore int
}

func (f *fakeDevFastStartApply) Apply(ctx context.Context, revisionID string) (jobs.Job, error) {
	f.calls++
	if f.calls <= f.failBefore {
		return jobs.Job{}, fmt.Errorf("runtime reload endpoint returned 502")
	}
	return jobs.Job{ID: "apply-" + revisionID, Status: jobs.StatusSucceeded, Result: "revision applied"}, nil
}

type fakeDevFastStartIssuer struct {
	calls int
}

func (f *fakeDevFastStartIssuer) Issue(ctx context.Context, certificateID string, commonName string, sanList []string, options *ACMEIssueOptions) (jobs.Job, error) {
	f.calls++
	return jobs.Job{ID: "issue-" + certificateID, Status: jobs.StatusSucceeded, Result: "certificate issued"}, nil
}

type fakeDevFastStartEasyProfiles struct {
	items       map[string]easysiteprofiles.EasySiteProfile
	upsertCalls int
}

func (f *fakeDevFastStartEasyProfiles) Get(siteID string) (easysiteprofiles.EasySiteProfile, bool, error) {
	if f.items == nil {
		return easysiteprofiles.EasySiteProfile{}, false, nil
	}
	item, ok := f.items[siteID]
	return item, ok, nil
}

func (f *fakeDevFastStartEasyProfiles) Upsert(ctx context.Context, profile easysiteprofiles.EasySiteProfile) (easysiteprofiles.EasySiteProfile, error) {
	if f.items == nil {
		f.items = make(map[string]easysiteprofiles.EasySiteProfile)
	}
	f.items[profile.SiteID] = profile
	f.upsertCalls++
	return profile, nil
}

type fakeDevFastStartRatePolicies struct {
	items []ratelimitpolicies.RateLimitPolicy
}

func (f *fakeDevFastStartRatePolicies) List() ([]ratelimitpolicies.RateLimitPolicy, error) {
	return append([]ratelimitpolicies.RateLimitPolicy(nil), f.items...), nil
}

func TestDevFastStartBootstrapperRunCreatesResourcesAndAppliesRevision(t *testing.T) {
	siteService := &fakeDevFastStartSites{}
	upstreamService := &fakeDevFastStartUpstreams{}
	certificateService := &fakeDevFastStartCertificates{}
	materials := &fakeDevFastStartMaterials{exists: false}
	tlsService := &fakeDevFastStartTLSConfigs{}
	revisionReader := &fakeDevFastStartRevisions{active: false}
	compileService := &fakeDevFastStartCompile{}
	applyService := &fakeDevFastStartApply{}
	issuer := &fakeDevFastStartIssuer{}
	easyProfiles := &fakeDevFastStartEasyProfiles{}

	bootstrapper := NewDevFastStartBootstrapper(
		config.DevFastStartConfig{
			Enabled:           true,
			Host:              "localhost",
			CertificateID:     "control-plane-localhost-tls",
			ManagementSiteID:  "control-plane-access",
			UpstreamHost:      "ui",
			UpstreamPort:      80,
			RetryDelaySeconds: 1,
			MaxAttempts:       2,
		},
		siteService,
		upstreamService,
		certificateService,
		materials,
		tlsService,
		revisionReader,
		compileService,
		applyService,
		issuer,
		easyProfiles,
		easyProfiles,
		nil,
	)

	if err := bootstrapper.Run(context.Background()); err != nil {
		t.Fatalf("dev fast start failed: %v", err)
	}
	if len(siteService.items) != 1 || siteService.items[0].ID != "control-plane-access" {
		t.Fatalf("expected management site to be created, got %+v", siteService.items)
	}
	if len(upstreamService.items) != 1 || upstreamService.items[0].ID != "control-plane-access-upstream" {
		t.Fatalf("expected management upstream to be created, got %+v", upstreamService.items)
	}
	if len(tlsService.items) != 1 || tlsService.items[0].CertificateID != "control-plane-localhost-tls" {
		t.Fatalf("expected tls config to be created, got %+v", tlsService.items)
	}
	if issuer.calls != 1 {
		t.Fatalf("expected certificate issue to run once, got %d", issuer.calls)
	}
	if compileService.calls != 1 {
		t.Fatalf("expected compile to run once, got %d", compileService.calls)
	}
	if applyService.calls != 1 {
		t.Fatalf("expected apply to run once, got %d", applyService.calls)
	}
	if easyProfiles.upsertCalls != 1 {
		t.Fatalf("expected easy profile to be initialized once, got %d", easyProfiles.upsertCalls)
	}
	profile, ok := easyProfiles.items["control-plane-access"]
	if !ok {
		t.Fatalf("expected easy profile for management site to be created")
	}
	if !profile.SecurityBehaviorAndLimits.UseLimitReq {
		t.Fatalf("expected management easy profile to keep limit_req enabled by default")
	}
	if !profile.SecurityBehaviorAndLimits.UseBadBehavior {
		t.Fatalf("expected management easy profile to keep bad-behavior enabled by default")
	}
	expectedRate := easysiteprofiles.DefaultProfile("control-plane-access").SecurityBehaviorAndLimits.LimitReqRate
	if profile.SecurityBehaviorAndLimits.LimitReqRate != expectedRate {
		t.Fatalf("expected default easy profile rate %s, got %s", expectedRate, profile.SecurityBehaviorAndLimits.LimitReqRate)
	}
	if len(profile.HTTPBehavior.AllowedMethods) < 7 {
		t.Fatalf("expected management easy profile to include API methods, got %+v", profile.HTTPBehavior.AllowedMethods)
	}
}

func TestDevFastStartBootstrapperRunRetriesApplyFailures(t *testing.T) {
	bootstrapper := NewDevFastStartBootstrapper(
		config.DevFastStartConfig{
			Enabled:           true,
			Host:              "localhost",
			CertificateID:     "control-plane-localhost-tls",
			ManagementSiteID:  "control-plane-access",
			UpstreamHost:      "ui",
			UpstreamPort:      80,
			RetryDelaySeconds: 1,
			MaxAttempts:       3,
		},
		&fakeDevFastStartSites{items: []sites.Site{{ID: "control-plane-access", PrimaryHost: "localhost", Enabled: true}}},
		&fakeDevFastStartUpstreams{items: []upstreams.Upstream{{ID: "control-plane-access-upstream", SiteID: "control-plane-access", Host: "ui", Port: 80, Scheme: "http"}}},
		&fakeDevFastStartCertificates{items: []certificates.Certificate{{ID: "control-plane-localhost-tls", CommonName: "localhost", Status: "active"}}},
		&fakeDevFastStartMaterials{exists: true},
		&fakeDevFastStartTLSConfigs{items: []tlsconfigs.TLSConfig{{SiteID: "control-plane-access", CertificateID: "control-plane-localhost-tls"}}},
		&fakeDevFastStartRevisions{active: false},
		&fakeDevFastStartCompile{},
		&fakeDevFastStartApply{failBefore: 1},
		&fakeDevFastStartIssuer{},
		&fakeDevFastStartEasyProfiles{items: map[string]easysiteprofiles.EasySiteProfile{
			"control-plane-access": easysiteprofiles.DefaultProfile("control-plane-access"),
		}},
		&fakeDevFastStartEasyProfiles{items: map[string]easysiteprofiles.EasySiteProfile{
			"control-plane-access": easysiteprofiles.DefaultProfile("control-plane-access"),
		}},
		nil,
	)

	if err := bootstrapper.Run(context.Background()); err != nil {
		t.Fatalf("expected retry to recover apply failure, got %v", err)
	}
}

func TestDevFastStartBootstrapperRunNoopsWhenActiveRevisionAlreadyExists(t *testing.T) {
	compileService := &fakeDevFastStartCompile{}
	applyService := &fakeDevFastStartApply{}
	issuer := &fakeDevFastStartIssuer{}
	easyProfiles := &fakeDevFastStartEasyProfiles{items: map[string]easysiteprofiles.EasySiteProfile{
		"control-plane-access": easysiteprofiles.DefaultProfile("control-plane-access"),
	}}

	bootstrapper := NewDevFastStartBootstrapper(
		config.DevFastStartConfig{
			Enabled:           true,
			Host:              "localhost",
			CertificateID:     "control-plane-localhost-tls",
			ManagementSiteID:  "control-plane-access",
			UpstreamHost:      "ui",
			UpstreamPort:      80,
			RetryDelaySeconds: 1,
			MaxAttempts:       1,
		},
		&fakeDevFastStartSites{items: []sites.Site{{ID: "control-plane-access", PrimaryHost: "localhost", Enabled: true}}},
		&fakeDevFastStartUpstreams{items: []upstreams.Upstream{{ID: "control-plane-access-upstream", SiteID: "control-plane-access", Host: "ui", Port: 80, Scheme: "http"}}},
		&fakeDevFastStartCertificates{items: []certificates.Certificate{{ID: "control-plane-localhost-tls", CommonName: "localhost", Status: "active"}}},
		&fakeDevFastStartMaterials{exists: true},
		&fakeDevFastStartTLSConfigs{items: []tlsconfigs.TLSConfig{{SiteID: "control-plane-access", CertificateID: "control-plane-localhost-tls"}}},
		&fakeDevFastStartRevisions{active: true},
		compileService,
		applyService,
		issuer,
		easyProfiles,
		easyProfiles,
		nil,
	)

	if err := bootstrapper.Run(context.Background()); err != nil {
		t.Fatalf("expected noop when active revision exists, got %v", err)
	}
	if compileService.calls != 0 || applyService.calls != 0 || issuer.calls != 0 {
		t.Fatalf("expected no compile/apply/issue, got compile=%d apply=%d issue=%d", compileService.calls, applyService.calls, issuer.calls)
	}
}

func TestDevFastStartBootstrapperRunKeepsManagementProtectionsEnabled(t *testing.T) {
	compileService := &fakeDevFastStartCompile{}
	applyService := &fakeDevFastStartApply{}
	easyProfiles := &fakeDevFastStartEasyProfiles{items: map[string]easysiteprofiles.EasySiteProfile{
		"control-plane-access": easysiteprofiles.DefaultProfile("control-plane-access"),
	}}
	ratePolicies := &fakeDevFastStartRatePolicies{items: []ratelimitpolicies.RateLimitPolicy{
		{ID: "easy-control-plane-access-rate", SiteID: "control-plane-access", Enabled: true},
	}}

	bootstrapper := NewDevFastStartBootstrapper(
		config.DevFastStartConfig{
			Enabled:           true,
			Host:              "localhost",
			CertificateID:     "control-plane-localhost-tls",
			ManagementSiteID:  "control-plane-access",
			UpstreamHost:      "ui",
			UpstreamPort:      80,
			RetryDelaySeconds: 1,
			MaxAttempts:       1,
		},
		&fakeDevFastStartSites{items: []sites.Site{{ID: "control-plane-access", PrimaryHost: "localhost", Enabled: true}}},
		&fakeDevFastStartUpstreams{items: []upstreams.Upstream{{ID: "control-plane-access-upstream", SiteID: "control-plane-access", Host: "ui", Port: 80, Scheme: "http"}}},
		&fakeDevFastStartCertificates{items: []certificates.Certificate{{ID: "control-plane-localhost-tls", CommonName: "localhost", Status: "active"}}},
		&fakeDevFastStartMaterials{exists: true},
		&fakeDevFastStartTLSConfigs{items: []tlsconfigs.TLSConfig{{SiteID: "control-plane-access", CertificateID: "control-plane-localhost-tls"}}},
		&fakeDevFastStartRevisions{active: true},
		compileService,
		applyService,
		&fakeDevFastStartIssuer{},
		easyProfiles,
		easyProfiles,
		ratePolicies,
	)

	if err := bootstrapper.Run(context.Background()); err != nil {
		t.Fatalf("expected bootstrap with management rate policy to succeed, got %v", err)
	}
	profile, ok := easyProfiles.items["control-plane-access"]
	if !ok {
		t.Fatalf("expected management easy profile to exist")
	}
	if !profile.SecurityBehaviorAndLimits.UseLimitReq || !profile.SecurityBehaviorAndLimits.UseBadBehavior {
		t.Fatalf("expected management protections to stay enabled, got %+v", profile.SecurityBehaviorAndLimits)
	}
	if easyProfiles.upsertCalls != 0 {
		t.Fatalf("expected no forced profile rewrite when protections already enabled, got %d", easyProfiles.upsertCalls)
	}
	if compileService.calls != 0 || applyService.calls != 0 {
		t.Fatalf("expected no compile/apply when nothing changed, got compile=%d apply=%d", compileService.calls, applyService.calls)
	}
}

func TestDevFastStartBootstrapperRunUpdatesExistingManagementRateDefaults(t *testing.T) {
	compileService := &fakeDevFastStartCompile{}
	applyService := &fakeDevFastStartApply{}
	profile := easysiteprofiles.DefaultProfile("control-plane-access")
	profile.SecurityBehaviorAndLimits.LimitReqRate = "25r/s"
	profile.SecurityBehaviorAndLimits.LimitConnMaxHTTP1 = 120
	profile.SecurityBehaviorAndLimits.LimitConnMaxHTTP2 = 240
	profile.SecurityBehaviorAndLimits.LimitConnMaxHTTP3 = 240
	profile.SecurityBehaviorAndLimits.BadBehaviorStatusCodes = []int{400, 401, 403, 404, 405, 429, 444}
	profile.SecurityBehaviorAndLimits.BadBehaviorBanTimeSeconds = 30

	easyProfiles := &fakeDevFastStartEasyProfiles{items: map[string]easysiteprofiles.EasySiteProfile{
		"control-plane-access": profile,
	}}

	bootstrapper := NewDevFastStartBootstrapper(
		config.DevFastStartConfig{
			Enabled:           true,
			Host:              "localhost",
			CertificateID:     "control-plane-localhost-tls",
			ManagementSiteID:  "control-plane-access",
			UpstreamHost:      "ui",
			UpstreamPort:      80,
			RetryDelaySeconds: 1,
			MaxAttempts:       1,
		},
		&fakeDevFastStartSites{items: []sites.Site{{ID: "control-plane-access", PrimaryHost: "localhost", Enabled: true}}},
		&fakeDevFastStartUpstreams{items: []upstreams.Upstream{{ID: "control-plane-access-upstream", SiteID: "control-plane-access", Host: "ui", Port: 80, Scheme: "http"}}},
		&fakeDevFastStartCertificates{items: []certificates.Certificate{{ID: "control-plane-localhost-tls", CommonName: "localhost", Status: "active"}}},
		&fakeDevFastStartMaterials{exists: true},
		&fakeDevFastStartTLSConfigs{items: []tlsconfigs.TLSConfig{{SiteID: "control-plane-access", CertificateID: "control-plane-localhost-tls"}}},
		&fakeDevFastStartRevisions{active: true},
		compileService,
		applyService,
		&fakeDevFastStartIssuer{},
		easyProfiles,
		easyProfiles,
		nil,
	)

	if err := bootstrapper.Run(context.Background()); err != nil {
		t.Fatalf("expected bootstrap update to succeed, got %v", err)
	}

	updated := easyProfiles.items["control-plane-access"]
	if updated.SecurityBehaviorAndLimits.LimitReqRate != "300r/s" {
		t.Fatalf("expected management rate limit update, got %s", updated.SecurityBehaviorAndLimits.LimitReqRate)
	}
	if updated.SecurityBehaviorAndLimits.LimitConnMaxHTTP1 < 200 {
		t.Fatalf("expected http/1 conn limit bump, got %d", updated.SecurityBehaviorAndLimits.LimitConnMaxHTTP1)
	}
	if sameIntSet(updated.SecurityBehaviorAndLimits.BadBehaviorStatusCodes, []int{400, 401, 403, 404, 405, 429, 444}) {
		t.Fatalf("expected management bad behavior codes to drop 429 escalation, got %+v", updated.SecurityBehaviorAndLimits.BadBehaviorStatusCodes)
	}
	if easyProfiles.upsertCalls != 1 {
		t.Fatalf("expected one management profile rewrite, got %d", easyProfiles.upsertCalls)
	}
	if compileService.calls != 1 || applyService.calls != 1 {
		t.Fatalf("expected compile/apply after management profile update, got compile=%d apply=%d", compileService.calls, applyService.calls)
	}
}
