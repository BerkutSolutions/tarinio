package services

import (
	"context"
	"fmt"

	"waf/control-plane/internal/audits"
	"waf/control-plane/internal/sites"
	"waf/control-plane/internal/upstreams"
)

type UpstreamStore interface {
	Create(item upstreams.Upstream) (upstreams.Upstream, error)
	List() ([]upstreams.Upstream, error)
	Update(item upstreams.Upstream) (upstreams.Upstream, error)
	Delete(id string) error
}

type SiteReader interface {
	List() ([]sites.Site, error)
}

type UpstreamService struct {
	store  UpstreamStore
	sites  SiteReader
	audits *AuditService
}

func NewUpstreamService(store UpstreamStore, sites SiteReader, audits *AuditService) *UpstreamService {
	return &UpstreamService{store: store, sites: sites, audits: audits}
}

func (s *UpstreamService) Create(ctx context.Context, item upstreams.Upstream) (created upstreams.Upstream, err error) {
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			Action:       "upstream.create",
			ResourceType: "upstream",
			ResourceID:   item.ID,
			SiteID:       item.SiteID,
			Status:       auditStatus(err),
			Summary:      "upstream create",
		})
	}()
	if err := s.ensureSiteExists(item.SiteID); err != nil {
		return upstreams.Upstream{}, err
	}
	created, err = s.store.Create(item)
	if err != nil {
		return created, err
	}
	if applyErr := runAutoApply(ctx); applyErr != nil {
		return upstreams.Upstream{}, applyErr
	}
	return created, nil
}

func (s *UpstreamService) List() ([]upstreams.Upstream, error) {
	return s.store.List()
}

func (s *UpstreamService) Update(ctx context.Context, item upstreams.Upstream) (updated upstreams.Upstream, err error) {
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			Action:       "upstream.update",
			ResourceType: "upstream",
			ResourceID:   item.ID,
			SiteID:       item.SiteID,
			Status:       auditStatus(err),
			Summary:      "upstream update",
		})
	}()
	if err := s.ensureSiteExists(item.SiteID); err != nil {
		return upstreams.Upstream{}, err
	}
	updated, err = s.store.Update(item)
	if err != nil {
		return updated, err
	}
	if applyErr := runAutoApply(ctx); applyErr != nil {
		return upstreams.Upstream{}, applyErr
	}
	return updated, nil
}

func (s *UpstreamService) Delete(ctx context.Context, id string) (err error) {
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			Action:       "upstream.delete",
			ResourceType: "upstream",
			ResourceID:   id,
			Status:       auditStatus(err),
			Summary:      "upstream delete",
		})
	}()
	if err := s.store.Delete(id); err != nil {
		return err
	}
	return runAutoApply(ctx)
}

func (s *UpstreamService) ensureSiteExists(siteID string) error {
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
