package main

import "net/http"

func cmdSitesList(c *cli, noAuth bool) error {
	value, err := c.requestJSON(http.MethodGet, "/api/sites", nil, !noAuth)
	if err != nil {
		return err
	}
	return printSites(c, value)
}
