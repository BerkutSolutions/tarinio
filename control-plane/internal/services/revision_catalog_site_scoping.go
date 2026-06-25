package services

import (
	"encoding/json"
	"sort"

	"waf/control-plane/internal/accesspolicies"
	"waf/control-plane/internal/certificates"
	"waf/control-plane/internal/easysiteprofiles"
	"waf/control-plane/internal/ratelimitpolicies"
	"waf/control-plane/internal/revisions"
	"waf/control-plane/internal/revisionsnapshots"
	"waf/control-plane/internal/sites"
	"waf/control-plane/internal/tlsconfigs"
	"waf/control-plane/internal/upstreams"
	"waf/control-plane/internal/wafpolicies"
)

type revisionSiteScope struct {
	ChangedSites  []RevisionCatalogSite
	ChangedSiteID map[string]struct{}
	ActiveSiteIDs []string
}

type siteRevisionFingerprint struct {
	Site              *sites.Site                        `json:"site,omitempty"`
	EasyProfile       *easysiteprofiles.EasySiteProfile  `json:"easy_profile,omitempty"`
	Upstreams         []upstreams.Upstream               `json:"upstreams,omitempty"`
	TLSConfigs        []tlsconfigs.TLSConfig             `json:"tls_configs,omitempty"`
	Certificates      []certificates.Certificate         `json:"certificates,omitempty"`
	WAFPolicies       []wafpolicies.WAFPolicy            `json:"waf_policies,omitempty"`
	AccessPolicies    []accesspolicies.AccessPolicy      `json:"access_policies,omitempty"`
	RateLimitPolicies []ratelimitpolicies.RateLimitPolicy `json:"rate_limit_policies,omitempty"`
}

func buildRevisionSiteScopes(
	revisionsAsc []revisions.Revision,
	snapshotsByRevision map[string]revisionsnapshots.Snapshot,
	activeRevisionID string,
	siteRefByID map[string]RevisionCatalogSite,
) map[string]revisionSiteScope {
	scopes := make(map[string]revisionSiteScope, len(revisionsAsc))
	activeFingerprints := map[string]string{}
	if activeSnapshot, ok := snapshotsByRevision[activeRevisionID]; ok {
		activeFingerprints = buildSiteFingerprintMap(activeSnapshot)
	}

	previousFingerprints := make(map[string]string)
	activeRevisionBySite := make(map[string]string)

	for _, revision := range revisionsAsc {
		snapshot, ok := snapshotsByRevision[revision.ID]
		if !ok {
			continue
		}
		fingerprints := buildSiteFingerprintMap(snapshot)
		changedSites := make([]RevisionCatalogSite, 0, len(fingerprints))
		changedSiteID := make(map[string]struct{}, len(fingerprints))
		for siteID, fingerprint := range fingerprints {
			if previousFingerprints[siteID] == fingerprint {
				continue
			}
			changedSiteID[siteID] = struct{}{}
			changedSites = append(changedSites, resolveRevisionSite(siteID, snapshot.Sites, siteRefByID))
			if activeFingerprints[siteID] != "" && activeFingerprints[siteID] == fingerprint {
				activeRevisionBySite[siteID] = revision.ID
			}
			previousFingerprints[siteID] = fingerprint
		}
		sort.Slice(changedSites, func(i, j int) bool {
			return changedSites[i].SiteID < changedSites[j].SiteID
		})
		scopes[revision.ID] = revisionSiteScope{
			ChangedSites:  changedSites,
			ChangedSiteID: changedSiteID,
		}
	}

	for revisionID, scope := range scopes {
		activeSiteIDs := make([]string, 0, len(scope.ChangedSiteID))
		for _, site := range scope.ChangedSites {
			if activeRevisionBySite[site.SiteID] == revisionID {
				activeSiteIDs = append(activeSiteIDs, site.SiteID)
			}
		}
		sort.Strings(activeSiteIDs)
		scope.ActiveSiteIDs = activeSiteIDs
		scopes[revisionID] = scope
	}

	return scopes
}

func buildSiteFingerprintMap(snapshot revisionsnapshots.Snapshot) map[string]string {
	fingerprints := make(map[string]string, len(snapshot.Sites))
	for _, site := range snapshot.Sites {
		fingerprints[site.ID] = buildSiteFingerprint(site.ID, snapshot)
	}
	return fingerprints
}

func buildSiteFingerprint(siteID string, snapshot revisionsnapshots.Snapshot) string {
	payload := siteRevisionFingerprint{
		Site:              findSite(snapshot.Sites, siteID),
		EasyProfile:       findEasyProfile(snapshot.EasySiteProfiles, siteID),
		Upstreams:         filterUpstreams(snapshot.Upstreams, siteID),
		TLSConfigs:        filterTLSConfigs(snapshot.TLSConfigs, siteID),
		Certificates:      filterCertificates(snapshot.Certificates, snapshot.TLSConfigs, siteID),
		WAFPolicies:       filterWAFPolicies(snapshot.WAFPolicies, siteID),
		AccessPolicies:    filterAccessPolicies(snapshot.AccessPolicies, siteID),
		RateLimitPolicies: filterRateLimitPolicies(snapshot.RateLimitPolicies, siteID),
	}
	content, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return string(content)
}

func resolveRevisionSite(siteID string, snapshotSites []sites.Site, siteRefByID map[string]RevisionCatalogSite) RevisionCatalogSite {
	if ref, ok := siteRefByID[siteID]; ok {
		return ref
	}
	if site := findSite(snapshotSites, siteID); site != nil {
		return RevisionCatalogSite{
			SiteID:      site.ID,
			PrimaryHost: site.PrimaryHost,
			Enabled:     site.Enabled,
		}
	}
	return RevisionCatalogSite{SiteID: siteID}
}

func findSite(items []sites.Site, siteID string) *sites.Site {
	for i := range items {
		if items[i].ID == siteID {
			item := items[i]
			return &item
		}
	}
	return nil
}

func findEasyProfile(items []easysiteprofiles.EasySiteProfile, siteID string) *easysiteprofiles.EasySiteProfile {
	for i := range items {
		if items[i].SiteID == siteID {
			item := items[i]
			return &item
		}
	}
	return nil
}

func filterUpstreams(items []upstreams.Upstream, siteID string) []upstreams.Upstream {
	filtered := make([]upstreams.Upstream, 0, 1)
	for _, item := range items {
		if item.SiteID == siteID {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterTLSConfigs(items []tlsconfigs.TLSConfig, siteID string) []tlsconfigs.TLSConfig {
	filtered := make([]tlsconfigs.TLSConfig, 0, 1)
	for _, item := range items {
		if item.SiteID == siteID {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterCertificates(items []certificates.Certificate, tlsItems []tlsconfigs.TLSConfig, siteID string) []certificates.Certificate {
	certificateIDs := make(map[string]struct{}, 1)
	for _, item := range tlsItems {
		if item.SiteID == siteID {
			certificateIDs[item.CertificateID] = struct{}{}
		}
	}
	filtered := make([]certificates.Certificate, 0, len(certificateIDs))
	for _, item := range items {
		if _, ok := certificateIDs[item.ID]; ok {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterWAFPolicies(items []wafpolicies.WAFPolicy, siteID string) []wafpolicies.WAFPolicy {
	filtered := make([]wafpolicies.WAFPolicy, 0, 1)
	for _, item := range items {
		if item.SiteID == siteID {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterAccessPolicies(items []accesspolicies.AccessPolicy, siteID string) []accesspolicies.AccessPolicy {
	filtered := make([]accesspolicies.AccessPolicy, 0, 1)
	for _, item := range items {
		if item.SiteID == siteID {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterRateLimitPolicies(items []ratelimitpolicies.RateLimitPolicy, siteID string) []ratelimitpolicies.RateLimitPolicy {
	filtered := make([]ratelimitpolicies.RateLimitPolicy, 0, 1)
	for _, item := range items {
		if item.SiteID == siteID {
			filtered = append(filtered, item)
		}
	}
	return filtered
}
