package services

import (
	"context"
	"fmt"

	"waf/control-plane/internal/audits"
	"waf/control-plane/internal/wafpolicies"
)

type WAFPolicyStore interface {
	Create(item wafpolicies.WAFPolicy) (wafpolicies.WAFPolicy, error)
	List() ([]wafpolicies.WAFPolicy, error)
	Update(item wafpolicies.WAFPolicy) (wafpolicies.WAFPolicy, error)
	Delete(id string) error
}

type WAFPolicyService struct {
	store  WAFPolicyStore
	sites  SiteReader
	audits *AuditService
}

func NewWAFPolicyService(store WAFPolicyStore, sites SiteReader, audits *AuditService) *WAFPolicyService {
	return &WAFPolicyService{store: store, sites: sites, audits: audits}
}

func (s *WAFPolicyService) Create(ctx context.Context, item wafpolicies.WAFPolicy) (created wafpolicies.WAFPolicy, err error) {
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			Action:       "wafpolicy.create",
			ResourceType: "wafpolicy",
			ResourceID:   item.ID,
			SiteID:       item.SiteID,
			Status:       auditStatus(err),
			Summary:      "waf policy create",
		})
	}()
	if err := s.ensureSiteExists(item.SiteID); err != nil {
		return wafpolicies.WAFPolicy{}, err
	}
	created, err = s.store.Create(item)
	if err != nil {
		return created, err
	}
	if applyErr := runAutoApply(ctx); applyErr != nil {
		return wafpolicies.WAFPolicy{}, applyErr
	}
	return created, nil
}

func (s *WAFPolicyService) List() ([]wafpolicies.WAFPolicy, error) {
	return s.store.List()
}

func (s *WAFPolicyService) Update(ctx context.Context, item wafpolicies.WAFPolicy) (updated wafpolicies.WAFPolicy, err error) {
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			Action:       "wafpolicy.update",
			ResourceType: "wafpolicy",
			ResourceID:   item.ID,
			SiteID:       item.SiteID,
			Status:       auditStatus(err),
			Summary:      "waf policy update",
		})
	}()
	if err := s.ensureSiteExists(item.SiteID); err != nil {
		return wafpolicies.WAFPolicy{}, err
	}
	updated, err = s.store.Update(item)
	if err != nil {
		return updated, err
	}
	if applyErr := runAutoApply(ctx); applyErr != nil {
		return wafpolicies.WAFPolicy{}, applyErr
	}
	return updated, nil
}

func (s *WAFPolicyService) Delete(ctx context.Context, id string) (err error) {
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			Action:       "wafpolicy.delete",
			ResourceType: "wafpolicy",
			ResourceID:   id,
			Status:       auditStatus(err),
			Summary:      "waf policy delete",
		})
	}()
	if err := s.store.Delete(id); err != nil {
		return err
	}
	return runAutoApply(ctx)
}

func (s *WAFPolicyService) ensureSiteExists(siteID string) error {
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
