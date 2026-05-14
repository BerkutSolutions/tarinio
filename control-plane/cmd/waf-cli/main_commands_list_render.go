package main

import "fmt"

func renderGenericList(c *cli, title string, value any) error {
	if c.outputJSON {
		return printJSON(value)
	}
	items := asList(value)
	fmt.Printf("%s: %d item(s)\n", title, len(items))
	if len(items) == 0 {
		return nil
	}
	return printTableFromMaps(items, []string{"id", "site_id", "enabled", "updated_at"})
}
