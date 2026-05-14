package main

import (
	"fmt"
	"strings"
)

func extractSiteFlag(args []string, defaultSite string) (string, []string, error) {
	siteID := defaultSite
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		current := strings.TrimSpace(args[i])
		if current == "" {
			continue
		}
		if strings.HasPrefix(current, "--site=") {
			value := strings.TrimSpace(strings.TrimPrefix(current, "--site="))
			if value == "" {
				return "", nil, fmt.Errorf("--site value is required")
			}
			siteID = value
			continue
		}
		if current == "--site" {
			if i+1 >= len(args) {
				return "", nil, fmt.Errorf("--site value is required")
			}
			value := strings.TrimSpace(args[i+1])
			if value == "" || strings.HasPrefix(value, "--") {
				return "", nil, fmt.Errorf("--site value is required")
			}
			siteID = value
			i += 1
			continue
		}
		out = append(out, args[i])
	}
	return siteID, out, nil
}
