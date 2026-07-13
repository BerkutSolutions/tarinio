package services

import (
	"strings"

	"waf/control-plane/internal/sites"
	"waf/control-plane/internal/upstreams"
)

// legacyManagementUIProxySiteIDs keeps existing installations manageable while
// they migrate to explicit management-host settings. The compose UI is an
// internal upstream, so an enabled site that is the sole ui:80 proxy is an
// unambiguous former management endpoint. Once an operator saves explicit
// hosts, this compatibility path is disabled permanently.
func legacyManagementUIProxySiteIDs(siteItems []sites.Site, upstreamItems []upstreams.Upstream, managementConfigured bool) map[string]struct{} {
	if managementConfigured {
		return nil
	}
	defaultUpstreamBySite := make(map[string]upstreams.Upstream, len(upstreamItems))
	for _, upstream := range upstreamItems {
		if _, exists := defaultUpstreamBySite[upstream.SiteID]; !exists {
			defaultUpstreamBySite[upstream.SiteID] = upstream
		}
	}
	candidates := make([]string, 0, 1)
	for _, site := range siteItems {
		if !site.Enabled {
			continue
		}
		upstream, ok := defaultUpstreamBySite[site.ID]
		if !ok || !isLegacyManagementUIProxy(upstream) {
			continue
		}
		candidates = append(candidates, site.ID)
	}
	if len(candidates) != 1 {
		return nil
	}
	return map[string]struct{}{candidates[0]: {}}
}

func isLegacyManagementUIProxy(upstream upstreams.Upstream) bool {
	return strings.EqualFold(strings.TrimSpace(upstream.Scheme), "http") &&
		strings.EqualFold(strings.TrimSpace(upstream.Host), "ui") &&
		upstream.Port == 80
}
