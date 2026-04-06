package services

import (
	"context"
	"fmt"
	"strings"

	"waf/control-plane/internal/accesspolicies"
	"waf/control-plane/internal/audits"
)

type AccessPolicyStore interface {
	Create(item accesspolicies.AccessPolicy) (accesspolicies.AccessPolicy, error)
	List() ([]accesspolicies.AccessPolicy, error)
	Update(item accesspolicies.AccessPolicy) (accesspolicies.AccessPolicy, error)
	Delete(id string) error
}

type AccessPolicyService struct {
	store  AccessPolicyStore
	sites  SiteReader
	audits *AuditService
}

func NewAccessPolicyService(store AccessPolicyStore, sites SiteReader, audits *AuditService) *AccessPolicyService {
	return &AccessPolicyService{store: store, sites: sites, audits: audits}
}

func (s *AccessPolicyService) Create(ctx context.Context, item accesspolicies.AccessPolicy) (created accesspolicies.AccessPolicy, err error) {
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			Action:       "accesspolicy.create",
			ResourceType: "accesspolicy",
			ResourceID:   item.ID,
			SiteID:       item.SiteID,
			Status:       auditStatus(err),
			Summary:      "access policy create",
		})
	}()
	if err := s.ensureSiteExists(item.SiteID); err != nil {
		return accesspolicies.AccessPolicy{}, err
	}
	created, err = s.store.Create(item)
	if err != nil {
		return created, err
	}
	if applyErr := runAutoApply(ctx); applyErr != nil {
		return accesspolicies.AccessPolicy{}, applyErr
	}
	return created, nil
}

func (s *AccessPolicyService) List() ([]accesspolicies.AccessPolicy, error) {
	return s.store.List()
}

func (s *AccessPolicyService) Update(ctx context.Context, item accesspolicies.AccessPolicy) (updated accesspolicies.AccessPolicy, err error) {
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			Action:       "accesspolicy.update",
			ResourceType: "accesspolicy",
			ResourceID:   item.ID,
			SiteID:       item.SiteID,
			Status:       auditStatus(err),
			Summary:      "access policy update",
		})
	}()
	if err := s.ensureSiteExists(item.SiteID); err != nil {
		return accesspolicies.AccessPolicy{}, err
	}
	updated, err = s.store.Update(item)
	if err != nil {
		return updated, err
	}
	if applyErr := runAutoApply(ctx); applyErr != nil {
		return accesspolicies.AccessPolicy{}, applyErr
	}
	return updated, nil
}

func (s *AccessPolicyService) Upsert(ctx context.Context, item accesspolicies.AccessPolicy) (out accesspolicies.AccessPolicy, err error) {
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			Action:       "accesspolicy.upsert",
			ResourceType: "accesspolicy",
			ResourceID:   item.ID,
			SiteID:       item.SiteID,
			Status:       auditStatus(err),
			Summary:      "access policy upsert",
		})
	}()
	if err := s.ensureSiteExists(item.SiteID); err != nil {
		return accesspolicies.AccessPolicy{}, err
	}
	existing, findErr := s.findBySiteID(item.SiteID)
	if findErr != nil {
		return accesspolicies.AccessPolicy{}, findErr
	}
	if existing != nil {
		item.ID = existing.ID
		out, err = s.store.Update(item)
	} else {
		if strings.TrimSpace(item.ID) == "" {
			item.ID = item.SiteID + "-access"
		}
		out, err = s.store.Create(item)
	}
	if err != nil {
		return out, err
	}
	if applyErr := runAutoApply(ctx); applyErr != nil {
		return accesspolicies.AccessPolicy{}, applyErr
	}
	return out, nil
}

func (s *AccessPolicyService) Delete(ctx context.Context, id string) (err error) {
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			Action:       "accesspolicy.delete",
			ResourceType: "accesspolicy",
			ResourceID:   id,
			Status:       auditStatus(err),
			Summary:      "access policy delete",
		})
	}()
	if err := s.store.Delete(id); err != nil {
		return err
	}
	return runAutoApply(ctx)
}

func (s *AccessPolicyService) findBySiteID(siteID string) (*accesspolicies.AccessPolicy, error) {
	items, err := s.store.List()
	if err != nil {
		return nil, err
	}
	needle := strings.ToLower(strings.TrimSpace(siteID))
	for _, item := range items {
		if strings.ToLower(strings.TrimSpace(item.SiteID)) == needle {
			copyItem := item
			return &copyItem, nil
		}
	}
	return nil, nil
}

func (s *AccessPolicyService) ensureSiteExists(siteID string) error {
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
