package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"waf/control-plane/internal/accesspolicies"
	"waf/control-plane/internal/audits"
)

type ManualBanService struct {
	store  AccessPolicyStore
	sites  SiteReader
	audits *AuditService
}

func NewManualBanService(store AccessPolicyStore, sites SiteReader, audits *AuditService) *ManualBanService {
	return &ManualBanService{
		store:  store,
		sites:  sites,
		audits: audits,
	}
}

func (s *ManualBanService) Ban(ctx context.Context, siteID string, address string) (policy accesspolicies.AccessPolicy, err error) {
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			Action:       "accesspolicy.ban",
			ResourceType: "accesspolicy",
			ResourceID:   siteID,
			SiteID:       siteID,
			Status:       auditStatus(err),
			Summary:      "manual ban",
			Details:      map[string]any{"ip": strings.TrimSpace(address)},
		})
	}()
	siteID = strings.ToLower(strings.TrimSpace(siteID))
	address = strings.TrimSpace(address)
	canonicalSiteID, err := s.resolveSiteID(siteID)
	if err != nil {
		return accesspolicies.AccessPolicy{}, err
	}
	siteID = canonicalSiteID

	policy, found, err := s.findBySite(siteID)
	if err != nil {
		return accesspolicies.AccessPolicy{}, err
	}
	if !found {
		created, createErr := s.store.Create(accesspolicies.AccessPolicy{
			ID:        defaultAccessPolicyID(siteID),
			SiteID:    siteID,
			Enabled:   true,
			AllowList: nil,
			DenyList:  []string{address},
		})
		if createErr != nil {
			return accesspolicies.AccessPolicy{}, createErr
		}
		if applyErr := runAutoApply(ctx); applyErr != nil {
			return accesspolicies.AccessPolicy{}, applyErr
		}
		return created, nil
	}

	if containsString(policy.DenyList, address) {
		return policy, nil
	}
	policy.Enabled = true
	policy.DenyList = append(policy.DenyList, address)
	updated, updateErr := s.store.Update(policy)
	if updateErr != nil {
		return accesspolicies.AccessPolicy{}, updateErr
	}
	if applyErr := runAutoApply(ctx); applyErr != nil {
		return accesspolicies.AccessPolicy{}, applyErr
	}
	return updated, nil
}

func (s *ManualBanService) Unban(ctx context.Context, siteID string, address string) (policy accesspolicies.AccessPolicy, err error) {
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			Action:       "accesspolicy.unban",
			ResourceType: "accesspolicy",
			ResourceID:   siteID,
			SiteID:       siteID,
			Status:       auditStatus(err),
			Summary:      "manual unban",
			Details:      map[string]any{"ip": strings.TrimSpace(address)},
		})
	}()
	siteID = strings.ToLower(strings.TrimSpace(siteID))
	address = strings.TrimSpace(address)
	canonicalSiteID, err := s.resolveSiteID(siteID)
	if err != nil {
		return accesspolicies.AccessPolicy{}, err
	}
	siteID = canonicalSiteID

	policy, found, err := s.findBySite(siteID)
	if err != nil {
		return accesspolicies.AccessPolicy{}, err
	}
	if !found {
		return accesspolicies.AccessPolicy{
			ID:        defaultAccessPolicyID(siteID),
			SiteID:    siteID,
			Enabled:   false,
			AllowList: nil,
			DenyList:  nil,
		}, nil
	}

	if !containsString(policy.DenyList, address) {
		return policy, nil
	}

	filtered := make([]string, 0, len(policy.DenyList))
	for _, item := range policy.DenyList {
		if strings.TrimSpace(item) == address {
			continue
		}
		filtered = append(filtered, item)
	}
	policy.DenyList = filtered
	updated, updateErr := s.store.Update(policy)
	if updateErr != nil {
		return accesspolicies.AccessPolicy{}, updateErr
	}
	if applyErr := runAutoApply(ctx); applyErr != nil {
		return accesspolicies.AccessPolicy{}, applyErr
	}
	return updated, nil
}

func (s *ManualBanService) findBySite(siteID string) (accesspolicies.AccessPolicy, bool, error) {
	items, err := s.store.List()
	if err != nil {
		return accesspolicies.AccessPolicy{}, false, err
	}
	for _, item := range items {
		if item.SiteID == siteID {
			return deepCopyPolicy(item), true, nil
		}
	}
	return accesspolicies.AccessPolicy{}, false, nil
}

func (s *ManualBanService) resolveSiteID(siteID string) (string, error) {
	siteID = strings.ToLower(strings.TrimSpace(siteID))
	if siteID == "" {
		return "", fmt.Errorf("site %s not found", siteID)
	}
	items, err := s.sites.List()
	if err != nil {
		return "", err
	}
	aliasToken := normalizeSiteAliasToken(siteID)
	for _, item := range items {
		id := strings.ToLower(strings.TrimSpace(item.ID))
		host := strings.ToLower(strings.TrimSpace(item.PrimaryHost))
		if id == siteID {
			return item.ID, nil
		}
		if normalizeSiteAliasToken(id) == aliasToken {
			return item.ID, nil
		}
		if host == siteID {
			return item.ID, nil
		}
		if normalizeSiteAliasToken(host) == aliasToken {
			return item.ID, nil
		}
	}
	return "", fmt.Errorf("site %s not found", siteID)
}

func normalizeSiteAliasToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	normalized := strings.Trim(b.String(), "-")
	return normalized
}

func defaultAccessPolicyID(siteID string) string {
	return siteID + "-access"
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if strings.TrimSpace(item) == target {
			return true
		}
	}
	return false
}

func deepCopyPolicy(item accesspolicies.AccessPolicy) accesspolicies.AccessPolicy {
	content, err := json.Marshal(item)
	if err != nil {
		return item
	}
	var copied accesspolicies.AccessPolicy
	if err := json.Unmarshal(content, &copied); err != nil {
		return item
	}
	return copied
}
