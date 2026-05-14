package main

import (
	"fmt"
	"net/http"
)

func cmdReports(c *cli, args []string, noAuth bool) error {
	if len(args) < 1 || args[0] != "revisions" {
		return fmt.Errorf("usage: waf-cli reports revisions")
	}
	value, err := c.requestJSON(http.MethodGet, "/api/reports/revisions", nil, !noAuth)
	if err != nil {
		return err
	}
	if c.outputJSON {
		return printJSON(value)
	}
	root := asMap(value)
	fmt.Println("Revision report summary:")
	keys := sortedMapKeys(root)
	for _, key := range keys {
		fmt.Printf("  %s: %s\n", key, stringify(root[key]))
	}
	return nil
}
