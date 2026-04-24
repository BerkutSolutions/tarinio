package main

import (
	"bufio"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type threatIntelHit struct {
	Feed  string
	Score int
	Label string
}

type threatIntelCIDR struct {
	Net *net.IPNet
	Hit threatIntelHit
}

type threatIntelMatcher struct {
	feeds           []string
	refreshInterval time.Duration
	client          *http.Client

	mu      sync.RWMutex
	loaded  time.Time
	ipHits  map[string]threatIntelHit
	cidrHit []threatIntelCIDR
}

func newThreatIntelMatcherFromEnv() *threatIntelMatcher {
	rawFeeds := strings.TrimSpace(os.Getenv("WAF_THREAT_INTEL_FEEDS"))
	if rawFeeds == "" {
		return nil
	}
	feeds := splitThreatIntelFeeds(rawFeeds)
	if len(feeds) == 0 {
		return nil
	}
	refresh := 300
	if parsed, err := strconv.Atoi(strings.TrimSpace(os.Getenv("WAF_THREAT_INTEL_REFRESH_SECONDS"))); err == nil && parsed > 0 {
		refresh = parsed
	}
	return &threatIntelMatcher{
		feeds:           feeds,
		refreshInterval: time.Duration(refresh) * time.Second,
		client:          &http.Client{Timeout: 5 * time.Second},
		ipHits:          map[string]threatIntelHit{},
		cidrHit:         []threatIntelCIDR{},
	}
}

func splitThreatIntelFeeds(value string) []string {
	parts := strings.Split(strings.TrimSpace(value), ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func (m *threatIntelMatcher) Match(ip string) (threatIntelHit, bool) {
	if m == nil {
		return threatIntelHit{}, false
	}
	maybeNormalize := strings.TrimSpace(ip)
	if maybeNormalize == "" {
		return threatIntelHit{}, false
	}
	m.ensureFresh()

	m.mu.RLock()
	defer m.mu.RUnlock()
	if hit, ok := m.ipHits[maybeNormalize]; ok {
		return hit, true
	}
	parsed := net.ParseIP(maybeNormalize)
	if parsed == nil {
		return threatIntelHit{}, false
	}
	for _, item := range m.cidrHit {
		if item.Net != nil && item.Net.Contains(parsed) {
			return item.Hit, true
		}
	}
	return threatIntelHit{}, false
}

func (m *threatIntelMatcher) ensureFresh() {
	m.mu.RLock()
	loaded := m.loaded
	refresh := m.refreshInterval
	m.mu.RUnlock()
	if !loaded.IsZero() && refresh > 0 && time.Since(loaded) < refresh {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.loaded.IsZero() && refresh > 0 && time.Since(m.loaded) < refresh {
		return
	}
	ipHits := make(map[string]threatIntelHit)
	cidrHits := make([]threatIntelCIDR, 0, 128)
	for _, feed := range m.feeds {
		rows, err := loadThreatIntelFeedRows(feed, m.client)
		if err != nil {
			continue
		}
		for _, row := range rows {
			if parsedIP := net.ParseIP(row.Target); parsedIP != nil {
				key := parsedIP.String()
				if _, exists := ipHits[key]; !exists {
					ipHits[key] = threatIntelHit{Feed: row.Feed, Score: row.Score, Label: row.Label}
				}
				continue
			}
			if _, network, err := net.ParseCIDR(row.Target); err == nil && network != nil {
				cidrHits = append(cidrHits, threatIntelCIDR{
					Net: network,
					Hit: threatIntelHit{Feed: row.Feed, Score: row.Score, Label: row.Label},
				})
			}
		}
	}
	m.ipHits = ipHits
	m.cidrHit = cidrHits
	m.loaded = time.Now().UTC()
}

type threatIntelRow struct {
	Feed   string
	Target string
	Score  int
	Label  string
}

func loadThreatIntelFeedRows(feed string, client *http.Client) ([]threatIntelRow, error) {
	feed = strings.TrimSpace(feed)
	if feed == "" {
		return nil, os.ErrInvalid
	}
	var raw string
	if strings.HasPrefix(feed, "http://") || strings.HasPrefix(feed, "https://") {
		if client == nil {
			client = &http.Client{Timeout: 5 * time.Second}
		}
		resp, err := client.Get(feed)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, os.ErrInvalid
		}
		buf := new(strings.Builder)
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			buf.WriteString(scanner.Text())
			buf.WriteByte('\n')
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}
		raw = buf.String()
	} else {
		content, err := os.ReadFile(feed)
		if err != nil {
			return nil, err
		}
		raw = string(content)
	}
	rows := make([]threatIntelRow, 0, 128)
	scanner := bufio.NewScanner(strings.NewReader(raw))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		target, score, label := parseThreatIntelLine(line)
		if target == "" {
			continue
		}
		rows = append(rows, threatIntelRow{
			Feed:   feed,
			Target: target,
			Score:  score,
			Label:  label,
		})
	}
	return rows, scanner.Err()
}

func parseThreatIntelLine(line string) (target string, score int, label string) {
	fields := strings.FieldsFunc(line, func(r rune) bool {
		return r == ',' || r == ';' || r == '\t' || r == ' '
	})
	if len(fields) == 0 {
		return "", 0, ""
	}
	target = strings.TrimSpace(fields[0])
	if target == "" {
		return "", 0, ""
	}
	score = 100
	label = "malicious"
	if len(fields) > 1 {
		if parsed, err := strconv.Atoi(strings.TrimSpace(fields[1])); err == nil && parsed > 0 {
			score = parsed
		} else if fields[1] != "" {
			label = strings.TrimSpace(fields[1])
		}
	}
	if len(fields) > 2 {
		label = strings.TrimSpace(fields[2])
	}
	if label == "" {
		label = "malicious"
	}
	return target, score, label
}
