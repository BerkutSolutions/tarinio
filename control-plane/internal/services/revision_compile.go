package services

import (
	"context"
	"fmt"
	"time"

	"waf/control-plane/internal/accesspolicies"
	"waf/control-plane/internal/antiddos"
	"waf/control-plane/internal/audits"
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

type SiteStateReader interface {
	List() ([]sites.Site, error)
}

type UpstreamStateReader interface {
	List() ([]upstreams.Upstream, error)
}

type CertificateStateReader interface {
	List() ([]certificates.Certificate, error)
}

type TLSConfigStateReader interface {
	List() ([]tlsconfigs.TLSConfig, error)
}

type WAFPolicyStateReader interface {
	List() ([]wafpolicies.WAFPolicy, error)
}

type AccessPolicyStateReader interface {
	List() ([]accesspolicies.AccessPolicy, error)
}

type RateLimitPolicyStateReader interface {
	List() ([]ratelimitpolicies.RateLimitPolicy, error)
}

type EasySiteProfileStateReader interface {
	List() ([]easysiteprofiles.EasySiteProfile, error)
}

type AntiDDoSSettingsReader interface {
	Get() (antiddos.Settings, error)
}

type RevisionMetadataStore interface {
	SavePending(revision revisions.Revision) error
	List() ([]revisions.Revision, error)
}

type RevisionSnapshotStore interface {
	Save(revisionID string, snapshot revisionsnapshots.Snapshot, materials []revisionsnapshots.MaterialContent) (string, string, error)
}

type CompileRequestResult struct {
	Revision revisions.Revision `json:"revision"`
	Job      jobs.Job           `json:"job"`
}

type RevisionCompileService struct {
	revisions         RevisionMetadataStore
	snapshots         RevisionSnapshotStore
	jobs              JobStore
	sites             SiteStateReader
	upstreams         UpstreamStateReader
	certificates      CertificateStateReader
	tlsConfigs        TLSConfigStateReader
	wafPolicies       WAFPolicyStateReader
	accessPolicies    AccessPolicyStateReader
	rateLimitPolicies RateLimitPolicyStateReader
	easySiteProfiles  EasySiteProfileStateReader
	antiDDoSSettings  AntiDDoSSettingsReader
	materials         CertificateMaterialReader
	audits            *AuditService
}

type CertificateMaterialReader interface {
	Read(certificateID string) (certificatematerials.MaterialRecord, []byte, []byte, error)
}

func NewRevisionCompileService(
	revisions RevisionMetadataStore,
	snapshots RevisionSnapshotStore,
	jobs JobStore,
	sites SiteStateReader,
	upstreams UpstreamStateReader,
	certificates CertificateStateReader,
	tlsConfigs TLSConfigStateReader,
	wafPolicies WAFPolicyStateReader,
	accessPolicies AccessPolicyStateReader,
	rateLimitPolicies RateLimitPolicyStateReader,
	easySiteProfiles EasySiteProfileStateReader,
	antiDDoSSettings AntiDDoSSettingsReader,
	materials CertificateMaterialReader,
	audits *AuditService,
) *RevisionCompileService {
	return &RevisionCompileService{
		revisions:         revisions,
		snapshots:         snapshots,
		jobs:              jobs,
		sites:             sites,
		upstreams:         upstreams,
		certificates:      certificates,
		tlsConfigs:        tlsConfigs,
		wafPolicies:       wafPolicies,
		accessPolicies:    accessPolicies,
		rateLimitPolicies: rateLimitPolicies,
		easySiteProfiles:  easySiteProfiles,
		antiDDoSSettings:  antiDDoSSettings,
		materials:         materials,
		audits:            audits,
	}
}

func (s *RevisionCompileService) Create(ctx context.Context) (result CompileRequestResult, err error) {
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			Action:            "revision.compile_request",
			ResourceType:      "revision",
			ResourceID:        result.Revision.ID,
			RelatedRevisionID: result.Revision.ID,
			RelatedJobID:      result.Job.ID,
			Status:            auditStatus(err),
			Summary:           "revision compile request",
		})
	}()
	existing, err := s.revisions.List()
	if err != nil {
		return CompileRequestResult{}, err
	}
	version := nextRevisionVersion(existing)
	revisionID := fmt.Sprintf("rev-%06d", version)
	createdAt := time.Now().UTC().Format(time.RFC3339Nano)

	snapshot, materials, err := s.buildSnapshot()
	if err != nil {
		return CompileRequestResult{}, err
	}
	snapshotPath, checksum, err := s.snapshots.Save(revisionID, snapshot, materials)
	if err != nil {
		return CompileRequestResult{}, err
	}

	revision := revisions.Revision{
		ID:         revisionID,
		Version:    version,
		CreatedAt:  createdAt,
		Checksum:   checksum,
		BundlePath: snapshotPath,
		Status:     revisions.StatusPending,
	}
	if err := s.revisions.SavePending(revision); err != nil {
		return CompileRequestResult{}, err
	}

	job, err := s.jobs.Create(jobs.Job{
		ID:               "compile-" + revisionID,
		Type:             jobs.TypeCompile,
		TargetRevisionID: revisionID,
	})
	if err != nil {
		return CompileRequestResult{}, err
	}

	return CompileRequestResult{
		Revision: revision,
		Job:      job,
	}, nil
}

func (s *RevisionCompileService) buildSnapshot() (revisionsnapshots.Snapshot, []revisionsnapshots.MaterialContent, error) {
	siteItems, err := s.sites.List()
	if err != nil {
		return revisionsnapshots.Snapshot{}, nil, err
	}
	upstreamItems, err := s.upstreams.List()
	if err != nil {
		return revisionsnapshots.Snapshot{}, nil, err
	}
	certificateItems, err := s.certificates.List()
	if err != nil {
		return revisionsnapshots.Snapshot{}, nil, err
	}
	tlsConfigItems, err := s.tlsConfigs.List()
	if err != nil {
		return revisionsnapshots.Snapshot{}, nil, err
	}
	wafPolicyItems, err := s.wafPolicies.List()
	if err != nil {
		return revisionsnapshots.Snapshot{}, nil, err
	}
	accessPolicyItems, err := s.accessPolicies.List()
	if err != nil {
		return revisionsnapshots.Snapshot{}, nil, err
	}
	rateLimitPolicyItems, err := s.rateLimitPolicies.List()
	if err != nil {
		return revisionsnapshots.Snapshot{}, nil, err
	}
	easySiteProfileItems, err := s.easySiteProfiles.List()
	if err != nil {
		return revisionsnapshots.Snapshot{}, nil, err
	}
	antiDDoSSettings := antiddos.DefaultSettings()
	if s.antiDDoSSettings != nil {
		item, err := s.antiDDoSSettings.Get()
		if err != nil {
			return revisionsnapshots.Snapshot{}, nil, err
		}
		antiDDoSSettings = item
	}
	materialItems := make([]revisionsnapshots.MaterialContent, 0, len(certificateItems))
	for _, certificate := range certificateItems {
		record, certificatePEM, privateKeyPEM, err := s.materials.Read(certificate.ID)
		if err != nil {
			continue
		}
		materialItems = append(materialItems, revisionsnapshots.MaterialContent{
			CertificateID:  record.CertificateID,
			CertificatePEM: certificatePEM,
			PrivateKeyPEM:  privateKeyPEM,
		})
	}

	return revisionsnapshots.Snapshot{
		Sites:             siteItems,
		Upstreams:         upstreamItems,
		Certificates:      certificateItems,
		TLSConfigs:        tlsConfigItems,
		WAFPolicies:       wafPolicyItems,
		AccessPolicies:    accessPolicyItems,
		RateLimitPolicies: rateLimitPolicyItems,
		EasySiteProfiles:  easySiteProfileItems,
		AntiDDoSSettings:  antiDDoSSettings,
	}, materialItems, nil
}

func nextRevisionVersion(items []revisions.Revision) int {
	maxVersion := 0
	for _, item := range items {
		if item.Version > maxVersion {
			maxVersion = item.Version
		}
	}
	return maxVersion + 1
}
