package main

import (
	"bufio"
	"os"
	"strings"
	"time"
)

func (s *securityEventSource) next() ([]securityEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if s.offset > stat.Size() {
		s.offset = 0
	}
	if _, err := file.Seek(s.offset, 0); err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	out := make([]securityEvent, 0, 32)
	burstBySecond := make(map[string]int)
	burstBySecondAndPath := make(map[string]int)
	burstMeta := make(map[string]map[string]string)
	threatSeenByTick := make(map[string]struct{})
	burstThreshold := burstThresholdFromEnv("WAF_SECURITY_EVENT_BURST_RPS_THRESHOLD", defaultBurstRequestsPerSecondThreshold)
	burstPathThreshold := burstThresholdFromEnv("WAF_SECURITY_EVENT_BURST_PATH_RPS_THRESHOLD", defaultBurstPathRequestsPerSecondThreshold)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		item, ok := parseAccessLine(line)
		if !ok {
			continue
		}
		if shouldSkipInternalManagementRequest(item) {
			continue
		}
		if s.threatIntel != nil {
			if hit, ok := s.threatIntel.Match(item.ip); ok {
				seenKey := item.ip + "\x00" + hit.Feed
				if _, exists := threatSeenByTick[seenKey]; !exists {
					threatSeenByTick[seenKey] = struct{}{}
					out = append(out, securityEvent{
						Type:            "security_threat_intel",
						Severity:        "warning",
						SiteID:          item.siteID,
						SourceComponent: "runtime-threat-intel",
						OccurredAt:      item.when.UTC().Format(time.RFC3339Nano),
						Summary:         "threat intel reputation match",
						Details: map[string]any{
							"client_ip":   item.ip,
							"country":     item.country,
							"city":        item.city,
							"host":        item.host,
							"path":        item.path,
							"feed":        hit.Feed,
							"intel_score": hit.Score,
							"intel_label": hit.Label,
						},
					})
				}
			}
		}
		if shouldTrackRequestBurst(item) {
			second := item.when.UTC().Format("2006-01-02T15:04:05Z")
			scopeKey := burstScopeKey(item)
			burstKey := item.ip + "|" + scopeKey + "|" + second
			burstBySecond[burstKey]++
			burstPath := normalizeBurstPath(item.path)
			burstPathKey := item.ip + "|" + scopeKey + "|" + burstPath + "|" + second
			burstBySecondAndPath[burstPathKey]++
			if _, exists := burstMeta[burstKey]; !exists {
				burstMeta[burstKey] = map[string]string{
					"ip":      item.ip,
					"ts":      second,
					"site_id": item.siteID,
					"host":    item.host,
					"country": item.country,
					"city":    item.city,
				}
			}
			if _, exists := burstMeta[burstPathKey]; !exists {
				burstMeta[burstPathKey] = map[string]string{
					"ip":      item.ip,
					"path":    burstPath,
					"ts":      second,
					"site_id": item.siteID,
					"host":    item.host,
					"country": item.country,
					"city":    item.city,
				}
			}
		}

		switch item.status {
		case 429:
			out = append(out, securityEvent{
				Type:            "security_rate_limit",
				Severity:        "warning",
				SiteID:          item.siteID,
				SourceComponent: "runtime-nginx",
				OccurredAt:      item.when.UTC().Format(time.RFC3339Nano),
				Summary:         "rate limit triggered",
				Details: map[string]any{
					"status":     item.status,
					"method":     item.method,
					"path":       item.path,
					"client_ip":  item.ip,
					"country":    item.country,
					"city":       item.city,
					"host":       item.host,
					"referer":    item.referer,
					"user_agent": item.userAgent,
				},
			})
		case 403, 444:
			out = append(out, securityEvent{
				Type:            "security_access",
				Severity:        "warning",
				SiteID:          item.siteID,
				SourceComponent: "runtime-nginx",
				OccurredAt:      item.when.UTC().Format(time.RFC3339Nano),
				Summary:         "access blocked",
				Details: map[string]any{
					"status":     item.status,
					"method":     item.method,
					"path":       item.path,
					"client_ip":  item.ip,
					"country":    item.country,
					"city":       item.city,
					"host":       item.host,
					"referer":    item.referer,
					"user_agent": item.userAgent,
				},
			})
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if pos, err := file.Seek(0, 1); err == nil {
		s.offset = pos
	}

	for key, count := range burstBySecond {
		if count < burstThreshold {
			continue
		}
		meta := burstMeta[key]
		out = append(out, securityEvent{
			Type:            "security_waf",
			Severity:        "warning",
			SiteID:          sanitizeSiteID(meta["site_id"]),
			SourceComponent: "runtime-nginx",
			OccurredAt:      meta["ts"],
			Summary:         "request burst detected (not blocked)",
			Details: map[string]any{
				"client_ip":       meta["ip"],
				"requests_second": count,
				"host":            meta["host"],
				"country":         meta["country"],
				"city":            meta["city"],
				"blocked":         false,
			},
		})
	}
	for key, count := range burstBySecondAndPath {
		if count < burstPathThreshold {
			continue
		}
		meta := burstMeta[key]
		out = append(out, securityEvent{
			Type:            "security_waf",
			Severity:        "warning",
			SiteID:          sanitizeSiteID(meta["site_id"]),
			SourceComponent: "runtime-nginx",
			OccurredAt:      meta["ts"],
			Summary:         "request burst detected on path (not blocked)",
			Details: map[string]any{
				"client_ip":         meta["ip"],
				"path":              meta["path"],
				"path_requests_sec": count,
				"host":              meta["host"],
				"country":           meta["country"],
				"city":              meta["city"],
				"blocked":           false,
			},
		})
	}

	return out, nil
}
