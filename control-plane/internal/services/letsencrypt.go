package services

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"waf/control-plane/internal/audits"
	"waf/control-plane/internal/certificates"
	"waf/control-plane/internal/jobs"
)

type LetsEncryptClient interface {
	Issue(commonName string, sanList []string) (IssuedMaterial, error)
	Renew(commonName string, sanList []string) (IssuedMaterial, error)
}

type IssuedMaterial struct {
	CertificatePEM []byte
	PrivateKeyPEM  []byte
	NotBefore      string
	NotAfter       string
}

type LetsEncryptService struct {
	client       LetsEncryptClient
	jobs         JobStore
	certificates CertificateStore
	materials    CertificateMaterialStore
	audits       *AuditService
}

func NewLetsEncryptService(client LetsEncryptClient, jobs JobStore, certificates CertificateStore, materials CertificateMaterialStore, audits *AuditService) *LetsEncryptService {
	return &LetsEncryptService{
		client:       client,
		jobs:         jobs,
		certificates: certificates,
		materials:    materials,
		audits:       audits,
	}
}

func (s *LetsEncryptService) Issue(ctx context.Context, certificateID string, commonName string, sanList []string) (job jobs.Job, err error) {
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			Action:       "certificate.acme_issue",
			ResourceType: "certificate",
			ResourceID:   strings.ToLower(strings.TrimSpace(certificateID)),
			RelatedJobID: job.ID,
			Status:       auditStatus(err),
			Summary:      "acme issue",
		})
	}()
	jobID := newJobID("issue", certificateID)
	job, err = s.jobs.Create(jobs.Job{
		ID:                  jobID,
		Type:                jobs.TypeCertificateIssue,
		TargetCertificateID: certificateID,
	})
	if err != nil {
		return jobs.Job{}, err
	}

	if _, err := s.jobs.MarkRunning(job.ID); err != nil {
		return jobs.Job{}, err
	}

	issued, err := s.client.Issue(strings.TrimSpace(commonName), sanList)
	if err != nil {
		return s.jobs.MarkFailed(job.ID, err.Error())
	}

	certificate, err := s.ensureCertificate(certificateID, commonName, sanList)
	if err != nil {
		return s.jobs.MarkFailed(job.ID, err.Error())
	}

	certificate.NotBefore = issued.NotBefore
	certificate.NotAfter = issued.NotAfter
	certificate.Status = "active"
	if _, err := s.certificates.Update(certificate); err != nil {
		return s.jobs.MarkFailed(job.ID, err.Error())
	}

	if _, err := s.materials.Put(certificate.ID, issued.CertificatePEM, issued.PrivateKeyPEM); err != nil {
		return s.jobs.MarkFailed(job.ID, err.Error())
	}

	return s.jobs.MarkSucceeded(job.ID, "certificate issued")
}

func (s *LetsEncryptService) Renew(ctx context.Context, certificateID string) (job jobs.Job, err error) {
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			Action:       "certificate.acme_renew",
			ResourceType: "certificate",
			ResourceID:   strings.ToLower(strings.TrimSpace(certificateID)),
			RelatedJobID: job.ID,
			Status:       auditStatus(err),
			Summary:      "acme renew",
		})
	}()
	jobID := newJobID("renew", certificateID)
	job, err = s.jobs.Create(jobs.Job{
		ID:                  jobID,
		Type:                jobs.TypeCertificateRenew,
		TargetCertificateID: certificateID,
	})
	if err != nil {
		return jobs.Job{}, err
	}

	if _, err := s.jobs.MarkRunning(job.ID); err != nil {
		return jobs.Job{}, err
	}

	certificate, err := s.findCertificate(certificateID)
	if err != nil {
		return s.jobs.MarkFailed(job.ID, err.Error())
	}

	issued, err := s.client.Renew(certificate.CommonName, certificate.SANList)
	if err != nil {
		return s.jobs.MarkFailed(job.ID, err.Error())
	}

	certificate.NotBefore = issued.NotBefore
	certificate.NotAfter = issued.NotAfter
	certificate.Status = "active"
	if _, err := s.certificates.Update(certificate); err != nil {
		return s.jobs.MarkFailed(job.ID, err.Error())
	}

	if _, err := s.materials.Put(certificate.ID, issued.CertificatePEM, issued.PrivateKeyPEM); err != nil {
		return s.jobs.MarkFailed(job.ID, err.Error())
	}

	return s.jobs.MarkSucceeded(job.ID, "certificate renewed")
}

func (s *LetsEncryptService) ensureCertificate(certificateID string, commonName string, sanList []string) (certificates.Certificate, error) {
	certificate, err := s.findCertificate(certificateID)
	if err == nil {
		return certificate, nil
	}
	if strings.TrimSpace(commonName) == "" {
		return certificates.Certificate{}, err
	}
	return s.certificates.Create(certificates.Certificate{
		ID:         strings.ToLower(strings.TrimSpace(certificateID)),
		CommonName: strings.TrimSpace(commonName),
		SANList:    sanList,
		Status:     "active",
	})
}

func (s *LetsEncryptService) findCertificate(certificateID string) (certificates.Certificate, error) {
	items, err := s.certificates.List()
	if err != nil {
		return certificates.Certificate{}, err
	}
	certificateID = strings.ToLower(strings.TrimSpace(certificateID))
	for _, item := range items {
		if item.ID == certificateID {
			return item, nil
		}
	}
	return certificates.Certificate{}, fmt.Errorf("certificate %s not found", certificateID)
}

func newJobID(prefix string, certificateID string) string {
	sum := sha256.Sum256([]byte(prefix + ":" + strings.ToLower(strings.TrimSpace(certificateID)) + ":" + time.Now().UTC().Format(time.RFC3339Nano)))
	return fmt.Sprintf("%s-%x", prefix, sum[:6])
}
