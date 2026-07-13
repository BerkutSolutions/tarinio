package services

import (
	"context"
	"fmt"
	"strings"

	"waf/control-plane/internal/audits"
	"waf/control-plane/internal/certificates"
	"waf/control-plane/internal/references"
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
	if err := s.ensureCertificateMatchesSite(item.SiteID, item.CertificateID); err != nil {
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
	action := "tlsconfig.update"
	details := map[string]any(nil)
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			Action:       action,
			ResourceType: "tlsconfig",
			ResourceID:   item.SiteID,
			SiteID:       item.SiteID,
			Status:       auditStatus(err),
			Summary:      action,
			Details:      details,
		})
	}()
	if err := s.ensureSiteExists(item.SiteID); err != nil {
		return tlsconfigs.TLSConfig{}, err
	}
	if err := s.ensureCertificateMatchesSite(item.SiteID, item.CertificateID); err != nil {
		return tlsconfigs.TLSConfig{}, err
	}
	current, listErr := s.store.List()
	if listErr != nil {
		return tlsconfigs.TLSConfig{}, listErr
	}
	for _, previous := range current {
		if previous.SiteID == item.SiteID && previous.CertificateID != item.CertificateID {
			action = "tlsconfig.rebind"
			details = map[string]any{"previous_certificate_id": previous.CertificateID, "certificate_id": item.CertificateID}
			break
		}
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

func (s *TLSConfigService) ensureCertificateMatchesSite(siteID, certificateID string) error {
	sitesList, err := s.sites.List()
	if err != nil {
		return err
	}
	certificatesList, err := s.certificates.List()
	if err != nil {
		return err
	}
	var host string
	foundSite := false
	for _, site := range sitesList {
		if site.ID == siteID {
			foundSite = true
			host = strings.ToLower(strings.TrimSpace(site.PrimaryHost))
			break
		}
	}
	if !foundSite {
		return references.NewError(references.CodeMissing, "site_id", siteID)
	}
	for _, certificate := range certificatesList {
		if certificate.ID != certificateID {
			continue
		}
		if host == "" {
			return nil
		}
		names := append([]string{certificate.CommonName}, certificate.SANList...)
		for _, name := range names {
			if certificateHostMatches(host, name) {
				return nil
			}
		}
		if strings.TrimSpace(certificate.CommonName) == "" && len(certificate.SANList) == 0 {
			return nil
		}
		return references.NewError(references.CodeCertificateHostMismatch, "certificate_id", certificateID)
	}
	return references.NewError(references.CodeMissing, "certificate_id", certificateID)
}

func certificateHostMatches(host, pattern string) bool {
	host, pattern = strings.ToLower(strings.TrimSpace(host)), strings.ToLower(strings.TrimSpace(pattern))
	if host == pattern {
		return true
	}
	if !strings.HasPrefix(pattern, "*.") {
		return false
	}
	suffix := strings.TrimPrefix(pattern, "*")
	return strings.HasSuffix(host, suffix) && strings.Count(host, ".") == strings.Count(strings.TrimPrefix(suffix, "."), ".")+1
}
