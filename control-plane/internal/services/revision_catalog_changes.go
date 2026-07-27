package services

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"

	"waf/control-plane/internal/revisions"
	"waf/control-plane/internal/revisionsnapshots"
)

type RevisionCatalogChange struct {
	SiteID string   `json:"site_id"`
	Action string   `json:"action"`
	Fields []string `json:"fields,omitempty"`
}

func buildRevisionChanges(items []revisions.Revision, snapshots map[string]revisionsnapshots.Snapshot) map[string][]RevisionCatalogChange {
	result := make(map[string][]RevisionCatalogChange, len(items))
	previous := map[string]any{}
	for _, revision := range items {
		snapshot, ok := snapshots[revision.ID]
		if !ok {
			continue
		}
		current := snapshotServiceValues(snapshot)
		siteIDs := make(map[string]struct{}, len(previous)+len(current))
		for siteID := range previous {
			siteIDs[siteID] = struct{}{}
		}
		for siteID := range current {
			siteIDs[siteID] = struct{}{}
		}
		changes := make([]RevisionCatalogChange, 0, len(siteIDs))
		for siteID := range siteIDs {
			before, hadBefore := previous[siteID]
			after, hasAfter := current[siteID]
			switch {
			case !hadBefore && hasAfter:
				changes = append(changes, RevisionCatalogChange{SiteID: siteID, Action: "created", Fields: []string{"service"}})
			case hadBefore && !hasAfter:
				changes = append(changes, RevisionCatalogChange{SiteID: siteID, Action: "deleted", Fields: []string{"service"}})
			case !reflect.DeepEqual(before, after):
				fields := make([]string, 0, 16)
				collectChangedFieldPaths("", before, after, &fields)
				sort.Strings(fields)
				changes = append(changes, RevisionCatalogChange{SiteID: siteID, Action: "updated", Fields: fields})
			}
		}
		sort.Slice(changes, func(i, j int) bool { return changes[i].SiteID < changes[j].SiteID })
		result[revision.ID] = changes
		previous = current
	}
	return result
}

func snapshotServiceValues(snapshot revisionsnapshots.Snapshot) map[string]any {
	result := make(map[string]any, len(snapshot.Sites))
	for _, site := range snapshot.Sites {
		content, err := json.Marshal(siteRevisionFingerprint{
			Site:              findSite(snapshot.Sites, site.ID),
			EasyProfile:       findEasyProfile(snapshot.EasySiteProfiles, site.ID),
			Upstreams:         filterUpstreams(snapshot.Upstreams, site.ID),
			TLSConfigs:        filterTLSConfigs(snapshot.TLSConfigs, site.ID),
			Certificates:      filterCertificates(snapshot.Certificates, snapshot.TLSConfigs, site.ID),
			WAFPolicies:       filterWAFPolicies(snapshot.WAFPolicies, site.ID),
			AccessPolicies:    filterAccessPolicies(snapshot.AccessPolicies, site.ID),
			RateLimitPolicies: filterRateLimitPolicies(snapshot.RateLimitPolicies, site.ID),
		})
		if err != nil {
			continue
		}
		var value any
		if json.Unmarshal(content, &value) == nil {
			result[site.ID] = value
		}
	}
	return result
}

func collectChangedFieldPaths(prefix string, before, after any, fields *[]string) {
	beforeMap, beforeOK := before.(map[string]any)
	afterMap, afterOK := after.(map[string]any)
	if beforeOK && afterOK {
		keys := make(map[string]struct{}, len(beforeMap)+len(afterMap))
		for key := range beforeMap {
			keys[key] = struct{}{}
		}
		for key := range afterMap {
			keys[key] = struct{}{}
		}
		for key := range keys {
			path := key
			if prefix != "" {
				path = prefix + "." + key
			}
			if isRevisionMetadataField(path) {
				continue
			}
			collectChangedFieldPaths(path, beforeMap[key], afterMap[key], fields)
		}
		return
	}
	if reflect.DeepEqual(before, after) {
		return
	}
	if prefix == "" {
		prefix = "service"
	}
	*fields = append(*fields, prefix)
}

func isRevisionMetadataField(path string) bool {
	name := path
	if index := strings.LastIndex(path, "."); index >= 0 {
		name = path[index+1:]
	}
	return name == "created_at" || name == "updated_at" || name == "last_login_at" || name == "last_used_at"
}

func filterRevisionChanges(changes []RevisionCatalogChange, targetSiteIDs []string) []RevisionCatalogChange {
	if len(targetSiteIDs) == 0 {
		return append([]RevisionCatalogChange(nil), changes...)
	}
	allowed := make(map[string]struct{}, len(targetSiteIDs))
	for _, siteID := range targetSiteIDs {
		allowed[strings.ToLower(strings.TrimSpace(siteID))] = struct{}{}
	}
	filtered := make([]RevisionCatalogChange, 0, len(changes))
	for _, change := range changes {
		if _, ok := allowed[strings.ToLower(change.SiteID)]; ok {
			filtered = append(filtered, change)
		}
	}
	return filtered
}
