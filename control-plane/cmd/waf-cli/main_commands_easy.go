package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
)

func cmdEasy(c *cli, args []string, noAuth bool) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: waf-cli easy get <site-id> | easy upsert <site-id> --file profile.json")
	}
	switch args[0] {
	case "get":
		if len(args) < 2 {
			return fmt.Errorf("usage: waf-cli easy get <site-id>")
		}
		siteID := strings.TrimSpace(args[1])
		value, err := c.requestJSON(http.MethodGet, "/api/easy-site-profiles/"+url.PathEscape(siteID), nil, !noAuth)
		if err != nil {
			return err
		}
		return printJSON(value)
	case "upsert":
		fs := flag.NewFlagSet("easy upsert", flag.ExitOnError)
		file := fs.String("file", "", "profile json file")
		_ = fs.Parse(args[1:])
		if fs.NArg() < 1 {
			return fmt.Errorf("usage: waf-cli easy upsert <site-id> --file profile.json")
		}
		siteID := strings.TrimSpace(fs.Arg(0))
		if strings.TrimSpace(*file) == "" {
			return fmt.Errorf("--file is required")
		}
		payload, err := readJSONFile(*file)
		if err != nil {
			return err
		}
		value, err := c.requestJSON(http.MethodPut, "/api/easy-site-profiles/"+url.PathEscape(siteID), payload, !noAuth)
		if err != nil {
			return err
		}
		if c.outputJSON {
			return printJSON(value)
		}
		fmt.Printf("Easy profile upserted for site %s from %s\n", siteID, filepath.Clean(*file))
		return nil
	default:
		return fmt.Errorf("usage: waf-cli easy get <site-id> | easy upsert <site-id> --file profile.json")
	}
}
