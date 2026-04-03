package services

import (
	"context"
	"testing"

	"waf/control-plane/internal/accesspolicies"
	"waf/control-plane/internal/antiddos"
	"waf/control-plane/internal/certificatematerials"
	"waf/control-plane/internal/certificates"
	"waf/control-plane/internal/easysiteprofiles"
	"waf/control-plane/internal/jobs"
	"waf/control-plane/internal/ratelimitpolicies"
	"waf/control-plane/internal/revisions"
	"waf/control-plane/internal/revisionsnapshots"
	"waf/control-plane/internal/sites"
	"waf/control-plane/internal/tlsconfigs"
	"waf/control-plane/internal/upstreams"
	"waf/control-plane/internal/wafpolicies"
)

type fakeRevisionMetadataStore struct {
	items []revisions.Revision
}

func (f *fakeRevisionMetadataStore) SavePending(revision revisions.Revision) error {
	f.items = append(f.items, revision)
	return nil
}

func (f *fakeRevisionMetadataStore) List() ([]revisions.Revision, error) {
	return append([]revisions.Revision(nil), f.items...), nil
}

type fakeRevisionSnapshotStore struct {
	path      string
	checksum  string
	snapshot  revisionsnapshots.Snapshot
	materials []revisionsnapshots.MaterialContent
}

func (f *fakeRevisionSnapshotStore) Save(revisionID string, snapshot revisionsnapshots.Snapshot, materials []revisionsnapshots.MaterialContent) (string, string, error) {
	f.snapshot = snapshot
	f.materials = append([]revisionsnapshots.MaterialContent(nil), materials...)
	return f.path, f.checksum, nil
}

type fakeSiteStateReader struct{ items []sites.Site }

func (f *fakeSiteStateReader) List() ([]sites.Site, error) {
	return append([]sites.Site(nil), f.items...), nil
}

type fakeUpstreamStateReader struct{ items []upstreams.Upstream }

func (f *fakeUpstreamStateReader) List() ([]upstreams.Upstream, error) {
	return append([]upstreams.Upstream(nil), f.items...), nil
}

type fakeCertificateStateReader struct{ items []certificates.Certificate }

func (f *fakeCertificateStateReader) List() ([]certificates.Certificate, error) {
	return append([]certificates.Certificate(nil), f.items...), nil
}

type fakeTLSConfigStateReader struct{ items []tlsconfigs.TLSConfig }

func (f *fakeTLSConfigStateReader) List() ([]tlsconfigs.TLSConfig, error) {
	return append([]tlsconfigs.TLSConfig(nil), f.items...), nil
}

type fakeWAFPolicyStateReader struct{ items []wafpolicies.WAFPolicy }

func (f *fakeWAFPolicyStateReader) List() ([]wafpolicies.WAFPolicy, error) {
	return append([]wafpolicies.WAFPolicy(nil), f.items...), nil
}

type fakeAccessPolicyStateReader struct{ items []accesspolicies.AccessPolicy }

func (f *fakeAccessPolicyStateReader) List() ([]accesspolicies.AccessPolicy, error) {
	return append([]accesspolicies.AccessPolicy(nil), f.items...), nil
}

type fakeRateLimitPolicyStateReader struct {
	items []ratelimitpolicies.RateLimitPolicy
}

func (f *fakeRateLimitPolicyStateReader) List() ([]ratelimitpolicies.RateLimitPolicy, error) {
	return append([]ratelimitpolicies.RateLimitPolicy(nil), f.items...), nil
}

type fakeEasySiteProfileStateReader struct {
	items []easysiteprofiles.EasySiteProfile
}

func (f *fakeEasySiteProfileStateReader) List() ([]easysiteprofiles.EasySiteProfile, error) {
	return append([]easysiteprofiles.EasySiteProfile(nil), f.items...), nil
}

type fakeAntiDDoSSettingsReader struct {
	item antiddos.Settings
}

func (f *fakeAntiDDoSSettingsReader) Get() (antiddos.Settings, error) {
	if f.item.ConnLimit == 0 {
		f.item = antiddos.DefaultSettings()
	}
	return f.item, nil
}

type fakeCertificateMaterialReader struct{}

func (f *fakeCertificateMaterialReader) Read(certificateID string) (certificatematerials.MaterialRecord, []byte, []byte, error) {
	return certificatematerials.MaterialRecord{CertificateID: certificateID}, []byte("CERT"), []byte("KEY"), nil
}

func TestRevisionCompileService_CreateBuildsPendingRevisionAndCompileJob(t *testing.T) {
	revisionStore := &fakeRevisionMetadataStore{items: []revisions.Revision{{ID: "rev-000001", Version: 1}}}
	snapshotStore := &fakeRevisionSnapshotStore{path: "snapshots/rev-000002.json", checksum: "abc123"}
	jobStore, err := jobs.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new job store: %v", err)
	}

	service := NewRevisionCompileService(
		revisionStore,
		snapshotStore,
		jobStore,
		&fakeSiteStateReader{items: []sites.Site{{ID: "site-a"}}},
		&fakeUpstreamStateReader{items: []upstreams.Upstream{{ID: "upstream-a", SiteID: "site-a"}}},
		&fakeCertificateStateReader{items: []certificates.Certificate{{ID: "cert-a", CommonName: "a.example.com", Status: "active"}}},
		&fakeTLSConfigStateReader{items: []tlsconfigs.TLSConfig{{SiteID: "site-a", CertificateID: "cert-a"}}},
		&fakeWAFPolicyStateReader{items: []wafpolicies.WAFPolicy{{ID: "waf-a", SiteID: "site-a", Enabled: true, Mode: wafpolicies.ModeDetection}}},
		&fakeAccessPolicyStateReader{items: []accesspolicies.AccessPolicy{{ID: "access-a", SiteID: "site-a"}}},
		&fakeRateLimitPolicyStateReader{items: []ratelimitpolicies.RateLimitPolicy{{ID: "rate-a", SiteID: "site-a", Enabled: true, Limits: ratelimitpolicies.Limits{RequestsPerSecond: 10, Burst: 5}}}},
		&fakeEasySiteProfileStateReader{items: []easysiteprofiles.EasySiteProfile{easysiteprofiles.DefaultProfile("site-a")}},
		&fakeAntiDDoSSettingsReader{item: antiddos.Settings{UseL4Guard: true, ConnLimit: 350, RatePerSecond: 175, RateBurst: 350, Ports: []int{80, 443}, ChainMode: antiddos.ChainModeAuto, Target: antiddos.TargetDrop, L7StatusCode: 429}},
		&fakeCertificateMaterialReader{},
		nil,
	)

	result, err := service.Create(context.Background())
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if result.Revision.ID != "rev-000002" || result.Revision.Version != 2 || result.Revision.Status != revisions.StatusPending {
		t.Fatalf("unexpected revision: %+v", result.Revision)
	}
	if result.Revision.BundlePath != "snapshots/rev-000002.json" || result.Revision.Checksum != "abc123" {
		t.Fatalf("unexpected snapshot metadata: %+v", result.Revision)
	}
	if result.Job.Type != jobs.TypeCompile || result.Job.TargetRevisionID != "rev-000002" || result.Job.Status != jobs.StatusPending {
		t.Fatalf("unexpected job: %+v", result.Job)
	}
	if len(snapshotStore.snapshot.Sites) != 1 || len(snapshotStore.snapshot.RateLimitPolicies) != 1 || len(snapshotStore.snapshot.EasySiteProfiles) != 1 {
		t.Fatalf("unexpected snapshot contents: %+v", snapshotStore.snapshot)
	}
	if snapshotStore.snapshot.AntiDDoSSettings.ConnLimit != 350 {
		t.Fatalf("expected anti-ddos settings in snapshot, got %+v", snapshotStore.snapshot.AntiDDoSSettings)
	}
	if len(snapshotStore.materials) != 1 {
		t.Fatalf("expected frozen materials, got %+v", snapshotStore.materials)
	}
}
