package services

import (
	"context"
	"fmt"
	"strings"

	"waf/control-plane/internal/audits"
	"waf/control-plane/internal/managementhosts"
	"waf/control-plane/internal/sites"
)

type ManagementHostSiteReader interface {
	List() ([]sites.Site, error)
}

type ManagementHostsService struct {
	store  *managementhosts.Store
	sites  ManagementHostSiteReader
	audits *AuditService
}

func NewManagementHostsService(store *managementhosts.Store, sites ManagementHostSiteReader, audits *AuditService) *ManagementHostsService {
	return &ManagementHostsService{store: store, sites: sites, audits: audits}
}
func (s *ManagementHostsService) Get() (managementhosts.Settings, error) { return s.store.Get() }
func (s *ManagementHostsService) UpdateDirectIPAccess(ctx context.Context, block bool) (item managementhosts.Settings, err error) {
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{Action: "settings.direct_ip_access.update", ResourceType: "system_setting", ResourceID: "direct_ip_access", Status: auditStatus(err), Summary: "update direct IP access policy"})
	}()
	return s.store.UpdateDirectIPAccess(block)
}
func (s *ManagementHostsService) Update(ctx context.Context, hosts []string, version int64) (item managementhosts.Settings, err error) {
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{Action: "settings.management_hosts.update", ResourceType: "system_setting", ResourceID: "management_hosts", Status: auditStatus(err), Summary: "update management hosts"})
	}()
	canonical, normalizeErr := managementhosts.NormalizeHosts(hosts)
	if normalizeErr != nil {
		return managementhosts.Settings{}, normalizeErr
	}
	current, getErr := s.store.Get()
	if getErr != nil {
		return managementhosts.Settings{}, getErr
	}
	if version != current.Version {
		return managementhosts.Settings{}, managementhosts.ErrVersionConflict
	}
	if err := s.ensureEnabledManagementSites(canonical); err != nil {
		return managementhosts.Settings{}, err
	}
	return s.store.Update(canonical, version)
}

// ensureEnabledManagementSites prevents persisting an ownership mapping that
// compiler cannot render into a management safeguard. The store is updated
// only after this precondition succeeds, so the previous working mapping stays
// intact on an operator typo or a half-completed service migration.
func (s *ManagementHostsService) ensureEnabledManagementSites(hosts []string) error {
	if len(hosts) == 0 {
		return fmt.Errorf("at least one management host is required")
	}
	if s.sites == nil {
		return fmt.Errorf("management host site reader is required")
	}
	items, err := s.sites.List()
	if err != nil {
		return fmt.Errorf("list sites for management-host validation: %w", err)
	}
	enabled := make(map[string]bool, len(items))
	for _, site := range items {
		if site.Enabled {
			enabled[strings.ToLower(strings.TrimSuffix(strings.TrimSpace(site.PrimaryHost), "."))] = true
		}
	}
	for _, host := range hosts {
		if !enabled[host] {
			return fmt.Errorf("management host %q must match an enabled site's primary_host before it can be saved", host)
		}
	}
	return nil
}
