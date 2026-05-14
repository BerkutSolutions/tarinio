package main

import (
	"fmt"
	"net/url"
	"strings"
)

func buildAuditPath(action, siteID, status string, limit, offset int) string {
	query := url.Values{}
	if strings.TrimSpace(action) != "" {
		query.Set("action", strings.TrimSpace(action))
	}
	if strings.TrimSpace(siteID) != "" {
		query.Set("site_id", strings.TrimSpace(siteID))
	}
	if strings.TrimSpace(status) != "" {
		query.Set("status", strings.TrimSpace(status))
	}
	query.Set("limit", fmt.Sprintf("%d", limit))
	query.Set("offset", fmt.Sprintf("%d", offset))
	path := "/api/audit"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	return path
}
