package main

import (
	"bufio"
	"encoding/json"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var accessLogPattern = regexp.MustCompile(`^(\S+) \S+ \S+ \[([^\]]+)\] "([A-Z]+) ([^"]*?) HTTP/[^"]*" (\d{3}) (\S+) "([^"]*)" "([^"]*)" "([^"]*)"$`)

type securityEvent struct {
	Type            string         `json:"type"`
	Severity        string         `json:"severity"`
	SiteID          string         `json:"site_id,omitempty"`
	SourceComponent string         `json:"source_component"`
	OccurredAt      string         `json:"occurred_at"`
	Summary         string         `json:"summary"`
	Details         map[string]any `json:"details,omitempty"`
}

type securityEventSource struct {
	mu     sync.Mutex
	path   string
	offset int64
}

func newSecurityEventSource(path string) *securityEventSource {
	return &securityEventSource{path: path}
}

func normalizeSiteID(value string) string {
	site := strings.ToLower(strings.TrimSpace(value))
	site = strings.ReplaceAll(site, "_", "-")
	return site
}

func shouldSkipInternalSite(siteID string) bool {
	switch normalizeSiteID(siteID) {
	case "control-plane-access", "control-plane", "ui":
		return true
	default:
		return false
	}
}

type requestStreamSource struct {
	mu       sync.Mutex
	path     string
	maxItems int
}

func newRequestStreamSource(path string, maxItems int) *requestStreamSource {
	if maxItems <= 0 {
		maxItems = 500
	}
	return &requestStreamSource{path: path, maxItems: maxItems}
}

func (s *requestStreamSource) latest() ([]map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []map[string]any{}, nil
		}
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	out := make([]map[string]any, 0, s.maxItems)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		item, ok := parseAccessLine(line)
		if !ok {
			continue
		}
		row := map[string]any{
			"stream":      "runtime",
			"ingested_at": item.when.UTC().Format(time.RFC3339Nano),
			"entry": map[string]any{
				"timestamp":     item.when.UTC().Format(time.RFC3339Nano),
				"request_id":    item.requestID,
				"client_ip":     item.ip,
				"method":        item.method,
				"uri":           item.path,
				"status":        item.status,
				"site":          item.siteID,
				"upstream_addr": item.upstreamAddr,
				"referer":       item.referer,
				"user_agent":    item.userAgent,
			},
		}
		out = append(out, row)
		if len(out) > s.maxItems {
			out = out[len(out)-s.maxItems:]
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

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
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		item, ok := parseAccessLine(line)
		if !ok {
			continue
		}
		if shouldSkipInternalSite(item.siteID) {
			continue
		}

		second := item.when.UTC().Format("2006-01-02T15:04:05Z")
		burstKey := item.ip + "|" + second
		burstBySecond[burstKey]++
		burstPathKey := item.ip + "|" + item.path + "|" + second
		burstBySecondAndPath[burstPathKey]++
		if _, exists := burstMeta[burstKey]; !exists {
			burstMeta[burstKey] = map[string]string{
				"ip":      item.ip,
				"ts":      second,
				"site_id": item.siteID,
			}
		}
		if _, exists := burstMeta[burstPathKey]; !exists {
			burstMeta[burstPathKey] = map[string]string{
				"ip":      item.ip,
				"path":    item.path,
				"ts":      second,
				"site_id": item.siteID,
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
		if count < 8 {
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
				"blocked":         false,
			},
		})
	}
	for key, count := range burstBySecondAndPath {
		if count < 4 {
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
				"blocked":           false,
			},
		})
	}

	return out, nil
}

type parsedAccess struct {
	requestID    string
	ip           string
	siteID       string
	method       string
	path         string
	status       int
	upstreamAddr string
	referer      string
	userAgent    string
	when         time.Time
}

func parseAccessLine(line string) (parsedAccess, bool) {
	if strings.HasPrefix(line, "{") {
		var item struct {
			Timestamp    string `json:"timestamp"`
			RequestID    string `json:"request_id"`
			ClientIP     string `json:"client_ip"`
			Method       string `json:"method"`
			URI          string `json:"uri"`
			Status       int    `json:"status"`
			Referer      string `json:"referer"`
			UserAgent    string `json:"user_agent"`
			Site         string `json:"site"`
			UpstreamAddr string `json:"upstream_addr"`
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
				requestID:    strings.TrimSpace(item.RequestID),
				ip:           ip,
				siteID:       sanitizeSiteID(item.Site),
				method:       strings.TrimSpace(item.Method),
				path:         strings.TrimSpace(item.URI),
				status:       item.Status,
				upstreamAddr: strings.TrimSpace(item.UpstreamAddr),
				referer:      strings.TrimSpace(item.Referer),
				userAgent:    strings.TrimSpace(item.UserAgent),
				when:         when.UTC(),
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
