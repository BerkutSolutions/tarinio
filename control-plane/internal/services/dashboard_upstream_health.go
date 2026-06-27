package services

import (
	"strings"
	"time"
)

const (
	upstreamHealthOK       = "ok"
	upstreamHealthWarning  = "warning"
	upstreamHealthCritical = "critical"

	upstreamWarnThreshold     = 0.05 // 5%
	upstreamCriticalThreshold = 0.15 // 15%
	upstreamWindowMinutes     = 5
)

// DashboardUpstreamHealth is the per-site upstream 5xx anomaly status.
type DashboardUpstreamHealth struct {
	SiteID        string  `json:"site_id"`
	Status        string  `json:"status"`
	ErrorRate     float64 `json:"error_rate"`
	WindowMinutes int     `json:"window_minutes"`
}

// summarizeUpstreamHealth calculates per-site 5xx error rates over the last
// upstreamWindowMinutes and returns a health status for each site that had
// at least one request in that window.
func summarizeUpstreamHealth(items []map[string]any, now time.Time) []DashboardUpstreamHealth {
	cutoff := now.Add(-time.Duration(upstreamWindowMinutes) * time.Minute)

	type siteStat struct {
		total  int
		errors int
	}
	stats := map[string]*siteStat{}

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
		siteID := strings.TrimSpace(asString(entry["site"]))
		if siteID == "" {
			host := strings.TrimSpace(asString(entry["host"]))
			if host == "" {
				continue
			}
			siteID = host
		}
		uri := strings.TrimSpace(asString(entry["uri"]))
		if shouldSkipInternalRequest(uri, siteID, strings.TrimSpace(asString(entry["host"]))) {
			continue
		}
		st := stats[siteID]
		if st == nil {
			st = &siteStat{}
			stats[siteID] = st
		}
		st.total++
		statusCode := parseAnyInt(entry["status"])
		if statusCode >= 500 && statusCode <= 599 {
			st.errors++
		}
	}

	out := make([]DashboardUpstreamHealth, 0, len(stats))
	for siteID, st := range stats {
		if st.total == 0 {
			continue
		}
		rate := float64(st.errors) / float64(st.total)
		status := upstreamHealthOK
		if rate > upstreamCriticalThreshold {
			status = upstreamHealthCritical
		} else if rate > upstreamWarnThreshold {
			status = upstreamHealthWarning
		}
		out = append(out, DashboardUpstreamHealth{
			SiteID:        siteID,
			Status:        status,
			ErrorRate:     rate,
			WindowMinutes: upstreamWindowMinutes,
		})
	}
	return out
}
