package main

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type parsedAccess struct {
	requestID      string
	ip             string
	siteID         string
	management     bool
	host           string
	country        string
	city           string
	method         string
	path           string
	status         int
	upstreamAddr   string
	referer        string
	userAgent      string
	securityReason string
	when           time.Time
}

func parseAccessLine(line string) (parsedAccess, bool) {
	if strings.HasPrefix(line, "{") {
		var item struct {
			Timestamp      string `json:"timestamp"`
			RequestID      string `json:"request_id"`
			ClientIP       string `json:"client_ip"`
			Country        string `json:"country"`
			City           string `json:"city"`
			Method         string `json:"method"`
			URI            string `json:"uri"`
			Status         int    `json:"status"`
			Referer        string `json:"referer"`
			UserAgent      string `json:"user_agent"`
			Site           string `json:"site"`
			Management     int    `json:"management"`
			Host           string `json:"host"`
			UpstreamAddr   string `json:"upstream_addr"`
			SecurityReason string `json:"security_reason"`
		}
		if err := json.Unmarshal([]byte(line), &item); err == nil {
			ip := strings.TrimSpace(item.ClientIP)
			if ip == "" || item.Status <= 0 {
				return parsedAccess{}, false
			}
			when, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(item.Timestamp))
			if err != nil {
				when, err = time.Parse(time.RFC3339, strings.TrimSpace(item.Timestamp))
				if err != nil {
					return parsedAccess{}, false
				}
			}
			return parsedAccess{
				requestID:      strings.TrimSpace(item.RequestID),
				ip:             ip,
				siteID:         sanitizeSiteID(item.Site),
				management:     item.Management == 1,
				host:           strings.ToLower(strings.TrimSpace(item.Host)),
				country:        strings.ToUpper(strings.TrimSpace(item.Country)),
				city:           strings.TrimSpace(item.City),
				method:         strings.TrimSpace(item.Method),
				path:           strings.TrimSpace(item.URI),
				status:         item.Status,
				upstreamAddr:   strings.TrimSpace(item.UpstreamAddr),
				referer:        strings.TrimSpace(item.Referer),
				userAgent:      strings.TrimSpace(item.UserAgent),
				securityReason: strings.TrimSpace(item.SecurityReason),
				when:           when.UTC(),
			}, true
		}
	}

	matches := accessLogPattern.FindStringSubmatch(line)
	if len(matches) != 10 {
		return parsedAccess{}, false
	}
	when, err := time.Parse("02/Jan/2006:15:04:05 -0700", matches[2])
	if err != nil {
		return parsedAccess{}, false
	}
	status, err := strconv.Atoi(matches[5])
	if err != nil {
		return parsedAccess{}, false
	}
	return parsedAccess{
		ip:           matches[1],
		siteID:       sanitizeSiteID(matches[9]),
		host:         "",
		country:      "",
		city:         "",
		method:       matches[3],
		path:         matches[4],
		status:       status,
		referer:      matches[7],
		userAgent:    matches[8],
		upstreamAddr: "",
		when:         when,
	}, true
}

func sanitizeSiteID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" || value == "-" {
		return ""
	}
	return value
}

func burstScopeKey(item parsedAccess) string {
	if siteID := sanitizeSiteID(item.siteID); siteID != "" {
		return siteID
	}
	if host := strings.ToLower(strings.TrimSpace(item.host)); host != "" {
		return host
	}
	return "unknown"
}

func normalizeBurstPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" || trimmed == "-" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err == nil && strings.TrimSpace(parsed.Path) != "" {
		trimmed = parsed.Path
	}
	return strings.ToLower(strings.TrimSpace(trimmed))
}
