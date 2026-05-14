package main

import (
	"fmt"
	"net/http"
)

func cmdHealth(c *cli) error {
	status, body, err := c.rawRequest(http.MethodGet, "/healthz", nil, false)
	if err != nil {
		return err
	}
	return renderHealthResponse(c.outputJSON, status, body)
}

func cmdSetup(c *cli, args []string, noAuth bool) error {
	if len(args) < 1 || args[0] != "status" {
		return fmt.Errorf("usage: waf-cli setup status")
	}
	value, err := c.requestJSON(http.MethodGet, "/api/setup/status", nil, !noAuth)
	if err != nil {
		return err
	}
	if c.outputJSON {
		return printJSON(value)
	}
	return renderSetupStatus(value)
}

func cmdAuthMe(c *cli, noAuth bool) error {
	value, err := c.requestJSON(http.MethodGet, "/api/auth/me", nil, !noAuth)
	if err != nil {
		return err
	}
	if c.outputJSON {
		return printJSON(value)
	}
	return renderAuthMe(value)
}
