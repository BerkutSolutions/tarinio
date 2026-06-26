package services

import (
	"strconv"
	"strings"
	"time"

	"waf/control-plane/internal/events"
)

func summarizeRequests(items []map[string]any, cutoff, now time.Time) (int, int, []DashboardKeyCount, []DashboardKeyCount, []DashboardTimeCount, []DashboardTimeCount, int, []DashboardKeyCount) {
	var total int
	series := map[time.Time]int{}
	blockedSeries := map[time.Time]int{}
	var blockedTotal int
	errorCounts := map[string]int{}
	siteCounts := map[string]int{}
	urlCounts := map[string]int{}
	uniqueIPs := map[string]struct{}{}

	for _, row := range items {
		entry, ok := row["entry"].(map[string]any)
		if !ok {
			continue
		}
		uri := strings.TrimSpace(asString(entry["uri"]))
		siteID := strings.TrimSpace(asString(entry["site"]))
		host := strings.TrimSpace(asString(entry["host"]))
		if shouldSkipInternalRequest(uri, siteID, host) {
			continue
		}
		when := parseAnyTime(entry["timestamp"])
		if when.IsZero() {
			when = parseAnyTime(row["ingested_at"])
		}
		if when.IsZero() || when.Before(cutoff) {
			continue
		}
		total++
		if siteID != "" {
			siteCounts[siteID]++
		} else if host != "" {
			siteCounts[host]++
		} else {
			siteCounts["-"]++
		}
		if uri != "" {
			urlCounts[uri]++
		}
		clientIP := strings.TrimSpace(asString(entry["client_ip"]))
		if clientIP != "" {
			uniqueIPs[clientIP] = struct{}{}
		}
		bucket := when.UTC().Truncate(time.Hour)
		series[bucket]++
		statusCode := parseAnyInt(entry["status"])
		if statusCode == 403 || statusCode == 429 || statusCode == 444 {
			blockedTotal++
			blockedSeries[bucket]++
		}
		if statusCode >= 400 && statusCode <= 599 {
			errorCounts[strconv.Itoa(statusCode)]++
		}
	}

	seriesOut := buildHourlySeries(series, now)
	blockedOut := buildHourlySeries(blockedSeries, now)
	return total, len(uniqueIPs), topCounts(siteCounts, 20), topCounts(urlCounts, 20), seriesOut, blockedOut, blockedTotal, topCounts(errorCounts, 7)
}

func summarizeAttackEvents(items []events.Event, cutoff, now time.Time) (int, int, int, []DashboardKeyCount, []DashboardKeyCount, []DashboardKeyCount, []DashboardTimeCount, []DashboardKeyCount) {
	var attacks int
	var blocked int
	ipCounts := map[string]int{}
	countryCounts := map[string]int{}
	urlCounts := map[string]int{}
	errorCounts := map[string]int{}
	uniqueIPs := map[string]struct{}{}
	blockedByHour := map[time.Time]int{}

	for _, item := range items {
		if item.Type != events.TypeSecurityAccess && item.Type != events.TypeSecurityRateLimit && item.Type != events.TypeSecurityWAF {
			continue
		}
		when := parseAnyTime(item.OccurredAt)
		if when.IsZero() || when.Before(cutoff) {
			continue
		}
		host := strings.TrimSpace(asString(item.Details["host"]))
		path := strings.TrimSpace(asString(item.Details["path"]))
		if path == "" {
			path = strings.TrimSpace(asString(item.Details["uri"]))
		}
		if shouldSkipInternalRequest(path, item.SiteID, host) {
			continue
		}
		// Not every security event should be counted as an "attack" in the dashboard.
		// Runtime may emit warning signals like "burst detected (not blocked)" during normal admin UI activity.
		// If backend explicitly marks the event as not blocked, treat it as an alert and skip it from attack metrics.
		if raw, ok := item.Details["blocked"]; ok {
			if flag, typeOK := raw.(bool); typeOK && !flag {
				continue
			}
		}

		attacks++
		ip := strings.TrimSpace(asString(item.Details["client_ip"]))
		if ip == "" {
			ip = strings.TrimSpace(asString(item.Details["ip"]))
		}
		if ip != "" {
			uniqueIPs[ip] = struct{}{}
			ipCounts[ip]++
		}

		urlPath := strings.TrimSpace(asString(item.Details["path"]))
		if urlPath == "" {
			urlPath = strings.TrimSpace(asString(item.Details["uri"]))
		}
		if urlPath != "" {
			urlCounts[urlPath]++
		}

		country := strings.ToUpper(strings.TrimSpace(asString(item.Details["country"])))
		if country == "" {
			country = strings.ToUpper(strings.TrimSpace(asString(item.Details["client_country"])))
		}
		if country == "" {
			country = "UNKNOWN"
		}
		countryCounts[country]++
		statusCode := parseAnyInt(item.Details["status"])
		if statusCode >= 400 && statusCode <= 599 {
			errorCounts[strconv.Itoa(statusCode)]++
		}

		if isBlockedEvent(item) {
			blocked++
			blockedByHour[when.UTC().Truncate(time.Hour)]++
		}
	}

	return attacks, blocked, len(uniqueIPs), topCounts(ipCounts, 10), topCounts(countryCounts, 10), topCounts(urlCounts, 10), buildHourlySeries(blockedByHour, now), topCounts(errorCounts, 7)
}

func summarizeRequestAttacks(items []map[string]any, cutoff time.Time) requestAttackSummary {
	ipCounts := map[string]int{}
	countryCounts := map[string]int{}
	urlCounts := map[string]int{}
	uniqueIPs := map[string]struct{}{}

	var out requestAttackSummary
	for _, row := range items {
		entry, ok := row["entry"].(map[string]any)
		if !ok {
			continue
		}
		when := parseAnyTime(entry["timestamp"])
		if when.IsZero() {
			when = parseAnyTime(row["ingested_at"])
		}
		if when.IsZero() || when.Before(cutoff) {
			continue
		}
		uri := strings.TrimSpace(asString(entry["uri"]))
		siteID := strings.TrimSpace(asString(entry["site"]))
		host := strings.TrimSpace(asString(entry["host"]))
		if shouldSkipInternalRequest(uri, siteID, host) {
			continue
		}
		statusCode := parseAnyInt(entry["status"])
		if statusCode != 403 && statusCode != 429 && statusCode != 444 {
			continue
		}

		out.AttacksDay++
		out.BlockedAttacksDay++
		if uri != "" {
			urlCounts[uri]++
		}

		ip := strings.TrimSpace(asString(entry["client_ip"]))
		if ip != "" {
			uniqueIPs[ip] = struct{}{}
			ipCounts[ip]++
		}

		country := strings.ToUpper(strings.TrimSpace(asString(entry["country"])))
		if country == "" {
			country = strings.ToUpper(strings.TrimSpace(asString(entry["client_country"])))
		}
		if country == "" {
			country = "UNKNOWN"
		}
		countryCounts[country]++
	}

	out.UniqueIPsDay = len(uniqueIPs)
	out.TopIPs = topCounts(ipCounts, 10)
	out.TopCountries = topCounts(countryCounts, 10)
	out.TopURLs = topCounts(urlCounts, 10)
	return out
}

func buildHourlySeries(values map[time.Time]int, now time.Time) []DashboardTimeCount {
	end := now.UTC().Truncate(time.Hour)
	start := end.Add(-23 * time.Hour)
	out := make([]DashboardTimeCount, 0, 24)
	for i := 0; i < 24; i++ {
		ts := start.Add(time.Duration(i) * time.Hour)
		out = append(out, DashboardTimeCount{
			Label:     ts.Format("15:04"),
			Timestamp: ts.Format(time.RFC3339),
			Count:     values[ts],
		})
	}
	return out
}
