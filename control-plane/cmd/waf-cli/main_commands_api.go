package main

import (
	"flag"
	"fmt"
)

func cmdAPI(c *cli, args []string, noAuth bool) error {
	fs := flag.NewFlagSet("api", flag.ExitOnError)
	file := fs.String("file", "", "json body file")
	_ = fs.Parse(args)
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: waf-cli api <GET|POST|PUT|DELETE> <path> [--file body.json]")
	}
	method, path := normalizeAPITarget(fs.Arg(0), fs.Arg(1))
	payload, err := loadAPIPayload(*file)
	if err != nil {
		return err
	}
	value, err := c.requestJSON(method, path, payload, !noAuth)
	if err != nil {
		if handleAPINoContentError(err, method, path) {
			return nil
		}
		return err
	}
	return printJSON(value)
}
