package main

import (
	"flag"
	"fmt"
	"net/http"
)

func cmdBans(c *cli, args []string, noAuth bool) error {
	if len(args) < 1 || args[0] != "list" {
		return fmt.Errorf("usage: waf-cli bans list [--site <id>]")
	}
	siteID := ""
	fs := flag.NewFlagSet("bans list", flag.ExitOnError)
	fs.StringVar(&siteID, "site", "", "site id")
	_ = fs.Parse(args[1:])

	value, err := c.requestJSON(http.MethodGet, "/api/access-policies", nil, !noAuth)
	if err != nil {
		return err
	}
	policies := asList(value)
	rows := buildBansRows(policies, siteID)
	return renderBansRows(c, rows)
}
