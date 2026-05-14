package main

import (
	"errors"
	"fmt"
	"net/http"
)

func cmdSitesDelete(c *cli, args []string, noAuth bool) error {
	if len(args) < 2 {
		return errors.New(sitesDeleteUsage())
	}
	id := normalizeSiteID(args[1])
	if id == "" {
		return fmt.Errorf("site id is required")
	}
	if err := c.requestNoContent(http.MethodDelete, siteDeletePath(id), nil, !noAuth); err != nil {
		return err
	}
	fmt.Printf("Site deleted: %s\n", id)
	return nil
}
