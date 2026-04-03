package services

import (
	"context"
	"fmt"

	"waf/control-plane/internal/audits"
	"waf/control-plane/internal/certificates"
	"waf/control-plane/internal/tlsconfigs"
)

type TLSConfigStore interface {
	Create(item tlsconfigs.TLSConfig) (tlsconfigs.TLSConfig, error)
	List() ([]tlsconfigs.TLSConfig, error)
	Update(item tlsconfigs.TLSConfig) (tlsconfigs.TLSConfig, error)
	Delete(siteID string) error
}

type CertificateReader interface {
	List() ([]certificates.Certificate, error)
}

type TLSConfigService struct {
	store        TLSConfigStore
	sites        SiteReader
	certificates CertificateReader
	audits       *AuditService
}

func NewTLSConfigService(store TLSConfigStore, sites SiteReader, certificates CertificateReader, audits *AuditService) *TLSConfigService {
	return &TLSConfigService{
		store:        store,
		sites:        sites,
		certificates: certificates,
		audits:       audits,
	}
}

func (s *TLSConfigService) Create(ctx context.Context, item tlsconfigs.TLSConfig) (created tlsconfigs.TLSConfig, err error) {
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			Action:       "tlsconfig.create",
			ResourceType: "tlsconfig",
			ResourceID:   item.SiteID,
			SiteID:       item.SiteID,
			Status:       auditStatus(err),
			Summary:      "tls config create",
		})
	}()
	if err := s.ensureSiteExists(item.SiteID); err != nil {
		return tlsconfigs.TLSConfig{}, err
	}
	if err := s.ensureCertificateExists(item.CertificateID); err != nil {
		return tlsconfigs.TLSConfig{}, err
	}
	created, err = s.store.Create(item)
	if err != nil {
		return created, err
	}
	if applyErr := runAutoApply(ctx); applyErr != nil {
		return tlsconfigs.TLSConfig{}, applyErr
	}
	return created, nil
}

func (s *TLSConfigService) List() ([]tlsconfigs.TLSConfig, error) {
	return s.store.List()
}

func (s *TLSConfigService) Update(ctx context.Context, item tlsconfigs.TLSConfig) (updated tlsconfigs.TLSConfig, err error) {
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			Action:       "tlsconfig.update",
			ResourceType: "tlsconfig",
			ResourceID:   item.SiteID,
			SiteID:       item.SiteID,
			Status:       auditStatus(err),
			Summary:      "tls config update",
		})
	}()
	if err := s.ensureSiteExists(item.SiteID); err != nil {
		return tlsconfigs.TLSConfig{}, err
	}
	if err := s.ensureCertificateExists(item.CertificateID); err != nil {
		return tlsconfigs.TLSConfig{}, err
	}
	updated, err = s.store.Update(item)
	if err != nil {
		return updated, err
	}
	if applyErr := runAutoApply(ctx); applyErr != nil {
		return tlsconfigs.TLSConfig{}, applyErr
	}
	return updated, nil
}

func (s *TLSConfigService) Delete(ctx context.Context, siteID string) (err error) {
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			Action:       "tlsconfig.delete",
			ResourceType: "tlsconfig",
			ResourceID:   siteID,
			SiteID:       siteID,
			Status:       auditStatus(err),
			Summary:      "tls config delete",
		})
	}()
	if err := s.store.Delete(siteID); err != nil {
		return err
	}
	return runAutoApply(ctx)
}

func (s *TLSConfigService) ensureSiteExists(siteID string) error {
	items, err := s.sites.List()
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.ID == siteID {
			return nil
		}
	}
	return fmt.Errorf("site %s not found", siteID)
}

func (s *TLSConfigService) ensureCertificateExists(certificateID string) error {
	items, err := s.certificates.List()
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.ID == certificateID {
			return nil
		}
	}
	return fmt.Errorf("certificate %s not found", certificateID)
}
