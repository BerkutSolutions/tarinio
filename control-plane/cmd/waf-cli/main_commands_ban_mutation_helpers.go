package main

import (
	"fmt"
	"net/url"
	"strings"
)

func buildBanMutationRequest(action string, positional []string, siteID string, usageSite string) (string, map[string]string, string, error) {
	if len(positional) < 1 {
		return "", nil, "", fmt.Errorf("usage: waf-cli %s <ip> [--site %s]", action, usageSite)
	}
	ip := strings.TrimSpace(positional[0])
	if ip == "" {
		return "", nil, "", fmt.Errorf("ip is required")
	}
	path := "/api/sites/" + url.PathEscape(siteID) + "/" + action
	body := map[string]string{"ip": ip}
	return path, body, ip, nil
}
