package services

import (
	"context"

	"waf/control-plane/internal/audits"
	"waf/control-plane/internal/sites"
)

type SiteStore interface {
	Create(site sites.Site) (sites.Site, error)
	List() ([]sites.Site, error)
	Update(site sites.Site) (sites.Site, error)
	Delete(id string) error
}

type SiteService struct {
	store  SiteStore
	audits *AuditService
}

func NewSiteService(store SiteStore, audits *AuditService) *SiteService {
	return &SiteService{store: store, audits: audits}
}

func (s *SiteService) Create(ctx context.Context, site sites.Site) (created sites.Site, err error) {
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			Action:       "site.create",
			ResourceType: "site",
			ResourceID:   site.ID,
			SiteID:       site.ID,
			Status:       auditStatus(err),
			Summary:      "site create",
		})
	}()
	created, err = s.store.Create(site)
	if err == nil {
		created.ID = created.ID
		if applyErr := runAutoApply(ctx); applyErr != nil {
			return sites.Site{}, applyErr
		}
	}
	return created, err
}

func (s *SiteService) List() ([]sites.Site, error) {
	return s.store.List()
}

func (s *SiteService) Update(ctx context.Context, site sites.Site) (updated sites.Site, err error) {
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			Action:       "site.update",
			ResourceType: "site",
			ResourceID:   site.ID,
			SiteID:       site.ID,
			Status:       auditStatus(err),
			Summary:      "site update",
		})
	}()
	updated, err = s.store.Update(site)
	if err != nil {
		return updated, err
	}
	if applyErr := runAutoApply(ctx); applyErr != nil {
		return sites.Site{}, applyErr
	}
	return updated, nil
}

func (s *SiteService) Delete(ctx context.Context, id string) (err error) {
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			Action:       "site.delete",
			ResourceType: "site",
			ResourceID:   id,
			SiteID:       id,
			Status:       auditStatus(err),
			Summary:      "site delete",
		})
	}()
	if err := s.store.Delete(id); err != nil {
		return err
	}
	return runAutoApply(ctx)
}
