package main

import (
	"fmt"
	"net/http"
)

func cmdBanLike(c *cli, action string, args []string, noAuth bool) error {
	defaultSite := envOrDefault("WAF_CLI_DEFAULT_SITE", envOrDefault("CONTROL_PLANE_DEV_FAST_START_MANAGEMENT_SITE_ID", "control-plane-access"))
	siteID, positional, err := extractSiteFlag(args, defaultSite)
	if err != nil {
		return err
	}
	path, body, ip, err := buildBanMutationRequest(action, positional, siteID, defaultSite)
	if err != nil {
		return err
	}
	value, err := c.requestJSON(http.MethodPost, path, body, !noAuth)
	if err != nil {
		return err
	}
	if c.outputJSON {
		return printJSON(value)
	}
	fmt.Printf("IP %s %sed for site %s\n", ip, action, siteID)
	if item, ok := value.(map[string]any); ok {
		deny := asStringSlice(item["denylist"])
		fmt.Printf("Active denylist size: %d\n", len(deny))
	}
	return nil
}
