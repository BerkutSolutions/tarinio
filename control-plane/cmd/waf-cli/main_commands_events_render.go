package main

import "fmt"

func renderEvents(c *cli, filtered []map[string]any) error {
	if c.outputJSON {
		return printJSON(filtered)
	}
	fmt.Printf("Events: %d item(s)\n", len(filtered))
	if len(filtered) == 0 {
		return nil
	}
	return printTableFromMaps(filtered, []string{"occurred_at", "type", "severity", "site_id", "summary"})
}
