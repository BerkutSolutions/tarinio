package main

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type requestDashboardKeyCount struct {
	Key     string `json:"key"`
	Count   int    `json:"count"`
	Country string `json:"country,omitempty"`
}

type requestDashboardTimeCount struct {
	Label     string `json:"label"`
	Timestamp string `json:"timestamp"`
	Count     int    `json:"count"`
}

type requestDashboardSummary struct {
	RequestsDay       int                         `json:"requests_day"`
	UniqueIPsDay      int                         `json:"unique_ips_day"`
	TopSites          []requestDashboardKeyCount  `json:"top_sites"`
	TopURLs           []requestDashboardKeyCount  `json:"top_urls"`
	RequestsSeries    []requestDashboardTimeCount `json:"requests_series"`
	BlockedDay        int                         `json:"blocked_day"`
	BlockedSeries     []requestDashboardTimeCount `json:"blocked_series"`
	PopularErrors     []requestDashboardKeyCount  `json:"popular_errors"`
	AttacksDay        int                         `json:"attacks_day"`
	UniqueAttackerIPs int                         `json:"unique_attacker_ips_day"`
	TopAttackerIPs    []requestDashboardKeyCount  `json:"top_attacker_ips"`
	TopCountries      []requestDashboardKeyCount  `json:"top_attacker_countries"`
	MostAttackedURLs  []requestDashboardKeyCount  `json:"most_attacked_urls"`
}

func (s *requestStreamSource) dashboardSummary(query map[string][]string) (requestDashboardSummary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	options := parseRequestQueryOptions(query, s.maxItems, s.defaultRetention)
	if err := s.ensureArchiveRootLocked(); err != nil {
		return requestDashboardSummary{}, err
	}
	if err := s.ingestArchiveLocked(options.RetentionDays); err != nil {
		return requestDashboardSummary{}, err
	}
	return s.dashboardSummaryArchiveLocked(options), nil
}

func (s *requestStreamSource) dashboardSummaryArchiveLocked(options requestQueryOptions) requestDashboardSummary {
	requestHours, blockedHours := map[time.Time]int{}, map[time.Time]int{}
	sites, urls, errorsByStatus := map[string]int{}, map[string]int{}, map[string]int{}
	attackIPs, countries, attackURLs := map[string]int{}, map[string]int{}, map[string]int{}
	attackIPCountries := map[string]map[string]int{}
	requestIPs, uniqueAttackIPs := map[string]struct{}{}, map[string]struct{}{}
	out := requestDashboardSummary{}
	for _, day := range requestDayArchiveKeys(options) {
		content, err := os.ReadFile(filepath.Join(s.archiveRoot, day+".jsonl"))
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(content), "\n") {
			record, ok := loadRequestLogRecordFromArchiveLine(line)
			if !ok || !requestRecordMatchesDashboardOptions(record, options) || shouldSkipDashboardRequest(record) {
				continue
			}
			out.RequestsDay++
			bucket, _ := parseRequestTimestamp(record.Timestamp)
			bucket = bucket.Truncate(time.Hour)
			requestHours[bucket]++
			sites[requestDashboardSiteKey(record)]++
			if record.URI != "" {
				urls[record.URI]++
			}
			if record.ClientIP != "" {
				requestIPs[record.ClientIP] = struct{}{}
			}
			if record.Status >= 400 && record.Status <= 599 {
				errorsByStatus[strconv.Itoa(record.Status)]++
			}
			if record.Status != 403 && record.Status != 429 && record.Status != 444 {
				continue
			}
			out.BlockedDay++
			out.AttacksDay++
			blockedHours[bucket]++
			if record.ClientIP != "" {
				attackIPs[record.ClientIP]++
				uniqueAttackIPs[record.ClientIP] = struct{}{}
				country := strings.ToUpper(strings.TrimSpace(record.Country))
				if country != "" && country != "UNKNOWN" {
					if attackIPCountries[record.ClientIP] == nil {
						attackIPCountries[record.ClientIP] = map[string]int{}
					}
					attackIPCountries[record.ClientIP][country]++
				}
			}
			country := strings.ToUpper(strings.TrimSpace(record.Country))
			if country != "" && country != "UNKNOWN" {
				countries[country]++
			}
			if record.URI != "" {
				attackURLs[record.URI]++
			}
		}
	}
	out.UniqueIPsDay, out.UniqueAttackerIPs = len(requestIPs), len(uniqueAttackIPs)
	out.TopSites, out.TopURLs = requestDashboardTopCounts(sites, 20), requestDashboardTopCounts(urls, 20)
	out.PopularErrors = requestDashboardTopCounts(errorsByStatus, 7)
	out.TopAttackerIPs, out.TopCountries = requestDashboardIPCounts(attackIPs, attackIPCountries, 10), requestDashboardTopCounts(countries, 10)
	out.MostAttackedURLs = requestDashboardTopCounts(attackURLs, 10)
	out.RequestsSeries, out.BlockedSeries = requestDashboardHourlySeries(requestHours), requestDashboardHourlySeries(blockedHours)
	return out
}

func requestRecordMatchesDashboardOptions(record requestLogRecord, options requestQueryOptions) bool {
	return requestTimestampMatchesOptions(record.Timestamp, options)
}

func shouldSkipDashboardRequest(record requestLogRecord) bool {
	if shouldSkipInternalSite(record.Site) {
		return true
	}
	host := strings.ToLower(strings.TrimSpace(record.Host))
	if isTarinioAdminAppPath(record.URI) && (host == "" || isInternalManagementHost(host) || strings.TrimSpace(record.Site) == "") {
		return true
	}
	if !isInternalManagementPath(record.URI) {
		return false
	}
	return host == "" || isInternalManagementHost(host) || strings.TrimSpace(record.Site) == ""
}

func requestDashboardSiteKey(record requestLogRecord) string {
	if value := strings.TrimSpace(record.Site); value != "" {
		return value
	}
	if value := strings.TrimSpace(record.Host); value != "" {
		return value
	}
	return "-"
}

func requestDashboardTopCounts(values map[string]int, limit int) []requestDashboardKeyCount {
	out := make([]requestDashboardKeyCount, 0, len(values))
	for key, count := range values {
		if strings.TrimSpace(key) != "" && count > 0 {
			out = append(out, requestDashboardKeyCount{Key: key, Count: count})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].Key < out[j].Key
		}
		return out[i].Count > out[j].Count
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func requestDashboardIPCounts(values map[string]int, countries map[string]map[string]int, limit int) []requestDashboardKeyCount {
	out := requestDashboardTopCounts(values, limit)
	for index := range out {
		country := requestDashboardTopCounts(countries[out[index].Key], 1)
		if len(country) > 0 {
			out[index].Country = country[0].Key
		}
	}
	return out
}

func requestDashboardHourlySeries(values map[time.Time]int) []requestDashboardTimeCount {
	end, start := time.Now().UTC().Truncate(time.Hour), time.Now().UTC().Truncate(time.Hour).Add(-23*time.Hour)
	out := make([]requestDashboardTimeCount, 0, 24)
	for at := start; !at.After(end); at = at.Add(time.Hour) {
		out = append(out, requestDashboardTimeCount{Label: at.Format("15:04"), Timestamp: at.Format(time.RFC3339), Count: values[at]})
	}
	return out
}
