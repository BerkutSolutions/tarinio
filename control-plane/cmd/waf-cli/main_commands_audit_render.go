package main

import "fmt"

func renderAuditRows(items []map[string]any, total string) error {
	fmt.Printf("Audit: %d item(s), total=%s\n", len(items), total)
	if len(items) == 0 {
		return nil
	}
	return printTableFromMaps(items, []string{"occurred_at", "action", "site_id", "status", "actor_ip"})
}
