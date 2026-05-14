package main

import "strings"

func filterEvents(items []map[string]any, query eventsQuery) []map[string]any {
	filtered := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if query.eventType != "" && !strings.EqualFold(stringify(item["type"]), query.eventType) {
			continue
		}
		if query.siteID != "" && !strings.EqualFold(stringify(item["site_id"]), query.siteID) {
			continue
		}
		if query.severity != "" && !strings.EqualFold(stringify(item["severity"]), query.severity) {
			continue
		}
		filtered = append(filtered, item)
	}
	if query.limit > 0 && len(filtered) > query.limit {
		filtered = filtered[:query.limit]
	}
	return filtered
}
