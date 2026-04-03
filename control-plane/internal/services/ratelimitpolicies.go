package services

import (
	"context"
	"fmt"

	"waf/control-plane/internal/audits"
	"waf/control-plane/internal/ratelimitpolicies"
)

type RateLimitPolicyStore interface {
	Create(item ratelimitpolicies.RateLimitPolicy) (ratelimitpolicies.RateLimitPolicy, error)
	List() ([]ratelimitpolicies.RateLimitPolicy, error)
	Update(item ratelimitpolicies.RateLimitPolicy) (ratelimitpolicies.RateLimitPolicy, error)
	Delete(id string) error
}

type RateLimitPolicyService struct {
	store  RateLimitPolicyStore
	sites  SiteReader
	audits *AuditService
}

func NewRateLimitPolicyService(store RateLimitPolicyStore, sites SiteReader, audits *AuditService) *RateLimitPolicyService {
	return &RateLimitPolicyService{store: store, sites: sites, audits: audits}
}

func (s *RateLimitPolicyService) Create(ctx context.Context, item ratelimitpolicies.RateLimitPolicy) (created ratelimitpolicies.RateLimitPolicy, err error) {
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			Action:       "ratelimitpolicy.create",
			ResourceType: "ratelimitpolicy",
			ResourceID:   item.ID,
			SiteID:       item.SiteID,
			Status:       auditStatus(err),
			Summary:      "rate limit policy create",
		})
	}()
	if err := s.ensureSiteExists(item.SiteID); err != nil {
		return ratelimitpolicies.RateLimitPolicy{}, err
	}
	created, err = s.store.Create(item)
	if err != nil {
		return created, err
	}
	if applyErr := runAutoApply(ctx); applyErr != nil {
		return ratelimitpolicies.RateLimitPolicy{}, applyErr
	}
	return created, nil
}

func (s *RateLimitPolicyService) List() ([]ratelimitpolicies.RateLimitPolicy, error) {
	return s.store.List()
}

func (s *RateLimitPolicyService) Update(ctx context.Context, item ratelimitpolicies.RateLimitPolicy) (updated ratelimitpolicies.RateLimitPolicy, err error) {
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			Action:       "ratelimitpolicy.update",
			ResourceType: "ratelimitpolicy",
			ResourceID:   item.ID,
			SiteID:       item.SiteID,
			Status:       auditStatus(err),
			Summary:      "rate limit policy update",
		})
	}()
	if err := s.ensureSiteExists(item.SiteID); err != nil {
		return ratelimitpolicies.RateLimitPolicy{}, err
	}
	updated, err = s.store.Update(item)
	if err != nil {
		return updated, err
	}
	if applyErr := runAutoApply(ctx); applyErr != nil {
		return ratelimitpolicies.RateLimitPolicy{}, applyErr
	}
	return updated, nil
}

func (s *RateLimitPolicyService) Delete(ctx context.Context, id string) (err error) {
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			Action:       "ratelimitpolicy.delete",
			ResourceType: "ratelimitpolicy",
			ResourceID:   id,
			Status:       auditStatus(err),
			Summary:      "rate limit policy delete",
		})
	}()
	if err := s.store.Delete(id); err != nil {
		return err
	}
	return runAutoApply(ctx)
}

func (s *RateLimitPolicyService) ensureSiteExists(siteID string) error {
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
