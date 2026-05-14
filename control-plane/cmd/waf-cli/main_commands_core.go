package main

import (
	"errors"
	"fmt"
	"net/http"
)

func cmdSites(c *cli, args []string, noAuth bool) error {
	if len(args) < 1 {
		return errors.New(sitesCommandUsage())
	}
	switch args[0] {
	case "list":
		return cmdSitesList(c, noAuth)
	case "delete":
		return cmdSitesDelete(c, args, noAuth)
	default:
		return errors.New(sitesCommandUsage())
	}
}

func cmdList(c *cli, title, path string, args []string, noAuth bool) error {
	if len(args) < 1 || args[0] != "list" {
		return fmt.Errorf("usage: waf-cli %s list", listUsageCommand(title))
	}
	value, err := c.requestJSON(http.MethodGet, path, nil, !noAuth)
	if err != nil {
		return err
	}
	return renderGenericList(c, listRenderTitle(title), value)
}
