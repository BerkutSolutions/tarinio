package services

import (
	"context"

	"waf/control-plane/internal/audits"
	"waf/control-plane/internal/certificates"
)

type CertificateStore interface {
	Create(item certificates.Certificate) (certificates.Certificate, error)
	List() ([]certificates.Certificate, error)
	Update(item certificates.Certificate) (certificates.Certificate, error)
	Delete(id string) error
}

type CertificateService struct {
	store  CertificateStore
	audits *AuditService
}

func NewCertificateService(store CertificateStore, audits *AuditService) *CertificateService {
	return &CertificateService{store: store, audits: audits}
}

func (s *CertificateService) Create(ctx context.Context, item certificates.Certificate) (created certificates.Certificate, err error) {
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			Action:       "certificate.create",
			ResourceType: "certificate",
			ResourceID:   item.ID,
			Status:       auditStatus(err),
			Summary:      "certificate create",
		})
	}()
	return s.store.Create(item)
}

func (s *CertificateService) List() ([]certificates.Certificate, error) {
	return s.store.List()
}

func (s *CertificateService) Update(ctx context.Context, item certificates.Certificate) (updated certificates.Certificate, err error) {
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			Action:       "certificate.update",
			ResourceType: "certificate",
			ResourceID:   item.ID,
			Status:       auditStatus(err),
			Summary:      "certificate update",
		})
	}()
	return s.store.Update(item)
}

func (s *CertificateService) Delete(ctx context.Context, id string) (err error) {
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			Action:       "certificate.delete",
			ResourceType: "certificate",
			ResourceID:   id,
			Status:       auditStatus(err),
			Summary:      "certificate delete",
		})
	}()
	return s.store.Delete(id)
}
