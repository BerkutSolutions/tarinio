package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

func cmdRevisions(c *cli, args []string, noAuth bool) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: waf-cli revisions compile | revisions apply <rev-id>")
	}
	switch args[0] {
	case "compile":
		value, err := c.requestJSON(http.MethodPost, "/api/revisions/compile", map[string]any{}, !noAuth)
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
		revision := asMap(root["revision"])
		job := asMap(root["job"])
		fmt.Printf("Compiled revision: %s\n", stringify(revision["id"]))
		fmt.Printf("Compile job: %s (%s)\n", stringify(job["id"]), stringify(job["status"]))
		return nil
	case "apply":
		if len(args) < 2 {
			return fmt.Errorf("usage: waf-cli revisions apply <rev-id>")
		}
		revID := strings.TrimSpace(args[1])
		if revID == "" {
			return fmt.Errorf("revision id is required")
		}
		value, err := c.requestJSON(http.MethodPost, "/api/revisions/"+url.PathEscape(revID)+"/apply", map[string]any{}, !noAuth)
		if err != nil {
			return err
		}
		if c.outputJSON {
			return printJSON(value)
		}
		job := asMap(value)
		fmt.Printf("Apply requested for revision %s\n", revID)
		fmt.Printf("Apply job: %s (%s)\n", stringify(job["id"]), stringify(job["status"]))
		return nil
	default:
		return fmt.Errorf("usage: waf-cli revisions compile | revisions apply <rev-id>")
	}
}
