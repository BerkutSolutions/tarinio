package services

import (
	"context"
	"fmt"
	"strings"

	"waf/control-plane/internal/accesspolicies"
	"waf/control-plane/internal/certificatematerials"
	"waf/control-plane/internal/certificates"
	"waf/control-plane/internal/easysiteprofiles"
	"waf/control-plane/internal/managementhosts"
	"waf/control-plane/internal/ratelimitpolicies"
	"waf/control-plane/internal/sites"
	"waf/control-plane/internal/tlsconfigs"
	"waf/control-plane/internal/upstreams"
	"waf/control-plane/internal/virtualpatches"
	"waf/control-plane/internal/wafpolicies"
)

// SiteMigration moves every site-scoped configuration record before the old
// identity is removed. It deliberately uses the stores directly so one rename
// produces one final runtime revision instead of many intermediate applies.
type SiteMigration struct {
	sites *sites.Store
	upstreams *upstreams.Store
	certificates *certificates.Store
	materials *certificatematerials.Store
	tls *tlsconfigs.Store
	waf *wafpolicies.Store
	access *accesspolicies.Store
	rate *ratelimitpolicies.Store
	easy *easysiteprofiles.Store
	patches *virtualpatches.Store
	management *managementhosts.Store
}

func NewSiteMigration(sites *sites.Store, upstreams *upstreams.Store, certificates *certificates.Store, materials *certificatematerials.Store, tls *tlsconfigs.Store, waf *wafpolicies.Store, access *accesspolicies.Store, rate *ratelimitpolicies.Store, easy *easysiteprofiles.Store, patches *virtualpatches.Store, management *managementhosts.Store) *SiteMigration {
	return &SiteMigration{sites: sites, upstreams: upstreams, certificates: certificates, materials: materials, tls: tls, waf: waf, access: access, rate: rate, easy: easy, patches: patches, management: management}
}

func (m *SiteMigration) Rename(ctx context.Context, oldID string, next sites.Site) (sites.Site, error) {
	_ = ctx
	oldID, next.ID = strings.ToLower(strings.TrimSpace(oldID)), strings.ToLower(strings.TrimSpace(next.ID))
	if oldID == "" || next.ID == "" || oldID == next.ID { return sites.Site{}, fmt.Errorf("site rename requires distinct ids") }
	items, err := m.sites.List(); if err != nil { return sites.Site{}, err }
	var old sites.Site; found := false
	for _, item := range items { if item.ID == oldID { old, found = item, true }; if item.ID == next.ID { return sites.Site{}, fmt.Errorf("site %s already exists", next.ID) } }
	if !found { return sites.Site{}, fmt.Errorf("site %s not found", oldID) }
	if strings.TrimSpace(next.PrimaryHost) == "" { return sites.Site{}, fmt.Errorf("site primary_host is required") }

	upstreamItems, err := m.upstreams.List(); if err != nil { return sites.Site{}, err }
	tlsItems, err := m.tls.List(); if err != nil { return sites.Site{}, err }
	wafItems, err := m.waf.List(); if err != nil { return sites.Site{}, err }
	accessItems, err := m.access.List(); if err != nil { return sites.Site{}, err }
	rateItems, err := m.rate.List(); if err != nil { return sites.Site{}, err }
	easyItems, err := m.easy.List(); if err != nil { return sites.Site{}, err }
	patchItems, err := m.patches.List(oldID); if err != nil { return sites.Site{}, err }

	var oldTLS *tlsconfigs.TLSConfig
	for i := range tlsItems { if tlsItems[i].SiteID == oldID { oldTLS = &tlsItems[i]; break } }
	if oldTLS != nil {
		newCertID := oldTLS.CertificateID
		if strings.EqualFold(newCertID, oldID+"-tls") { newCertID = next.ID + "-tls" }
		if newCertID != oldTLS.CertificateID {
			if err := m.renameCertificate(oldTLS.CertificateID, newCertID); err != nil { return sites.Site{}, err }
		}
		if _, err := m.tls.Create(tlsconfigs.TLSConfig{SiteID: next.ID, CertificateID: newCertID}); err != nil { return sites.Site{}, err }
	}
	for _, item := range upstreamItems { if item.SiteID == oldID { item.SiteID = next.ID; if _, err := m.upstreams.Update(item); err != nil { return sites.Site{}, err } } }
	for _, item := range wafItems { if item.SiteID == oldID { item.SiteID = next.ID; if _, err := m.waf.Update(item); err != nil { return sites.Site{}, err } } }
	for _, item := range accessItems { if item.SiteID == oldID { item.SiteID = next.ID; if _, err := m.access.Update(item); err != nil { return sites.Site{}, err } } }
	for _, item := range rateItems { if item.SiteID == oldID { item.SiteID = next.ID; if _, err := m.rate.Update(item); err != nil { return sites.Site{}, err } } }
	for _, item := range easyItems { if item.SiteID == oldID { item.SiteID = next.ID; if _, err := m.easy.Create(item); err != nil { return sites.Site{}, err }; if err := m.easy.Delete(oldID); err != nil { return sites.Site{}, err } } }
	for _, item := range patchItems { item.SiteID = next.ID; if err := m.patches.Delete(item.ID); err != nil { return sites.Site{}, err }; if _, err := m.patches.Create(item); err != nil { return sites.Site{}, err } }
	created, err := m.sites.Create(next); if err != nil { return sites.Site{}, err }
	if oldTLS != nil { if err := m.tls.Delete(oldID); err != nil { return sites.Site{}, err } }
	if err := m.sites.Delete(oldID); err != nil { return sites.Site{}, err }
	if err := m.moveManagementHost(old.PrimaryHost, created.PrimaryHost); err != nil { return sites.Site{}, err }
	return created, nil
}

func (m *SiteMigration) renameCertificate(oldID, newID string) error {
	items, err := m.certificates.List(); if err != nil { return err }
	for _, item := range items {
		if item.ID != oldID { continue }
		item.ID = newID
		if _, err := m.certificates.Create(item); err != nil { return err }
		if _, pem, key, readErr := m.materials.Read(oldID); readErr == nil { if _, err := m.materials.Put(newID, pem, key); err != nil { return err }; _ = m.materials.Delete(oldID) }
		return m.certificates.Delete(oldID)
	}
	return nil
}

func (m *SiteMigration) moveManagementHost(oldHost, newHost string) error {
	settings, err := m.management.Get(); if err != nil || !settings.Migrated { return err }
	changed := false
	for i, host := range settings.Hosts { if strings.EqualFold(host, oldHost) { settings.Hosts[i], changed = newHost, true } }
	if !changed { return nil }
	_, err = m.management.Update(settings.Hosts, settings.Version)
	return err
}
