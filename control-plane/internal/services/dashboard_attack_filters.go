package services

import (
	"sort"
	"strings"

	"waf/control-plane/internal/events"
)

func shouldSkipInternalRequest(uri string, siteID string, host string) bool {
	path := strings.ToLower(strings.TrimSpace(uri))
	site := strings.ToLower(strings.TrimSpace(siteID))
	site = strings.ReplaceAll(site, "_", "-")
	if site == "control-plane-access" || site == "control-plane" || site == "ui" {
		return true
	}
	if path == "" {
		return false
	}
	if isTarinioAdminAppPath(path) {
		return true
	}
	if !isInternalManagementPath(path) {
		return false
	}
	host = strings.ToLower(strings.TrimSpace(host))
	return host == "" || host == "localhost" || host == "127.0.0.1" || host == "::1" || host == "control-plane" || host == "ui" || site == ""
}

func isTarinioAdminAppPath(path string) bool {
	canonical := strings.ToLower(strings.TrimSpace(path))
	if canonical == "" {
		return false
	}
	for _, exact := range tarinioAdminExactPaths {
		if canonical == exact {
			return true
		}
	}
	for _, prefix := range tarinioAdminPrefixPaths {
		if strings.HasPrefix(canonical, prefix) {
			return true
		}
	}
	for _, prefix := range tarinioAdminSegmentPrefixes {
		if canonical == prefix || strings.HasPrefix(canonical, prefix+"/") {
			return true
		}
	}
	return false
}

func isInternalManagementPath(path string) bool {
	return strings.HasPrefix(path, "/api/") ||
		strings.HasPrefix(path, "/static/") ||
		strings.HasPrefix(path, "/dashboard") ||
		strings.HasPrefix(path, "/healthz") ||
		strings.HasPrefix(path, "/readyz") ||
		strings.HasPrefix(path, "/login") ||
		strings.HasPrefix(path, "/logout") ||
		strings.HasPrefix(path, "/setup") ||
		strings.HasPrefix(path, "/onboarding") ||
		strings.HasPrefix(path, "/favicon") ||
		strings.HasPrefix(path, "/manifest") ||
		strings.HasPrefix(path, "/site.webmanifest")
}

func mergeSeriesMax(primary []DashboardTimeCount, secondary []DashboardTimeCount) []DashboardTimeCount {
	if len(primary) == 0 {
		return append([]DashboardTimeCount(nil), secondary...)
	}
	secMap := make(map[string]DashboardTimeCount, len(secondary))
	for _, item := range secondary {
		secMap[item.Timestamp] = item
	}
	out := make([]DashboardTimeCount, 0, len(primary))
	for _, item := range primary {
		other, ok := secMap[item.Timestamp]
		if ok && other.Count > item.Count {
			item.Count = other.Count
		}
		out = append(out, item)
	}
	return out
}

func mergeKeyCountsSum(primary []DashboardKeyCount, secondary []DashboardKeyCount, limit int) []DashboardKeyCount {
	sums := map[string]int{}
	for _, item := range primary {
		key := strings.TrimSpace(item.Key)
		if key == "" {
			continue
		}
		sums[key] += item.Count
	}
	for _, item := range secondary {
		key := strings.TrimSpace(item.Key)
		if key == "" {
			continue
		}
		sums[key] += item.Count
	}
	return topCounts(sums, limit)
}

func isBlockedEvent(item events.Event) bool {
	if blocked, ok := item.Details["blocked"].(bool); ok {
		return blocked
	}
	status := parseAnyInt(item.Details["status"])
	return status == 403 || status == 429 || status == 444
}

func topCounts(values map[string]int, limit int) []DashboardKeyCount {
	out := make([]DashboardKeyCount, 0, len(values))
	for key, count := range values {
		if strings.TrimSpace(key) == "" || count <= 0 {
			continue
		}
		out = append(out, DashboardKeyCount{Key: key, Count: count})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].Key < out[j].Key
		}
		return out[i].Count > out[j].Count
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}
