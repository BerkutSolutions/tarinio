package main

import "strings"

func buildBansRows(policies []map[string]any, siteID string) []map[string]any {
	rows := make([]map[string]any, 0)
	for _, item := range policies {
		if siteID != "" && !strings.EqualFold(stringify(item["site_id"]), siteID) {
			continue
		}
		deny := asStringSlice(item["denylist"])
		for _, ip := range deny {
			rows = append(rows, map[string]any{
				"site_id":    stringify(item["site_id"]),
				"policy_id":  stringify(item["id"]),
				"ip":         ip,
				"updated_at": stringify(item["updated_at"]),
			})
		}
	}
	return rows
}
