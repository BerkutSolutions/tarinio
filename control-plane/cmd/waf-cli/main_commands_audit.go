package main

import (
	"flag"
	"net/http"
)

func cmdAudit(c *cli, args []string, noAuth bool) error {
	fs := flag.NewFlagSet("audit", flag.ExitOnError)
	action := fs.String("action", "", "audit action")
	siteID := fs.String("site-id", "", "site id")
	status := fs.String("status", "", "status")
	limit := fs.Int("limit", 50, "limit")
	offset := fs.Int("offset", 0, "offset")
	_ = fs.Parse(args)

	path := buildAuditPath(*action, *siteID, *status, *limit, *offset)
	value, err := c.requestJSON(http.MethodGet, path, nil, !noAuth)
	if err != nil {
		return err
	}
	if c.outputJSON {
		return printJSON(value)
	}
	root, ok := value.(map[string]any)
	if !ok {
		return printJSON(value)
	}
	items := asList(root["items"])
	total := stringify(root["total"])
	return renderAuditRows(items, total)
}
