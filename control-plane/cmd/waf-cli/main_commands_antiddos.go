package main

import (
	"flag"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
)

func cmdAntiDDoS(c *cli, args []string, noAuth bool) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: waf-cli antiddos get | antiddos upsert --file antiddos.json")
	}
	switch args[0] {
	case "get":
		value, err := c.requestJSON(http.MethodGet, "/api/anti-ddos/settings", nil, !noAuth)
		if err != nil {
			return err
		}
		return printJSON(value)
	case "upsert":
		fs := flag.NewFlagSet("antiddos upsert", flag.ExitOnError)
		file := fs.String("file", "", "settings json file")
		_ = fs.Parse(args[1:])
		if strings.TrimSpace(*file) == "" {
			return fmt.Errorf("--file is required")
		}
		payload, err := readJSONFile(*file)
		if err != nil {
			return err
		}
		value, err := c.requestJSON(http.MethodPut, "/api/anti-ddos/settings", payload, !noAuth)
		if err != nil {
			return err
		}
		if c.outputJSON {
			return printJSON(value)
		}
		fmt.Printf("Anti-DDoS settings upserted from %s\n", filepath.Clean(*file))
		return nil
	default:
		return fmt.Errorf("usage: waf-cli antiddos get | antiddos upsert --file antiddos.json")
	}
}
