package main

import (
	"fmt"
	"sort"
)

func renderBansRows(c *cli, rows []map[string]any) error {
	if c.outputJSON {
		return printJSON(rows)
	}
	fmt.Printf("Bans: %d item(s)\n", len(rows))
	if len(rows) == 0 {
		return nil
	}
	sort.Slice(rows, func(i, j int) bool {
		return stringify(rows[i]["site_id"])+stringify(rows[i]["ip"]) < stringify(rows[j]["site_id"])+stringify(rows[j]["ip"])
	})
	return printTableFromMaps(rows, []string{"site_id", "ip", "policy_id", "updated_at"})
}
