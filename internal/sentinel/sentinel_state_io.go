package sentinel

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	sentinelsource "waf/internal/sentinel/source"
)

var accessLogPattern = regexp.MustCompile(`^(\S+) \S+ \S+ \[([^\]]+)\] "([A-Z]+) ([^"]*?) HTTP/[^"]*" (\d{3}) (\S+) "([^"]*)" "([^"]*)" "([^"]*)"$`)

func loadState(path string) State {
	content, err := os.ReadFile(path)
	if err != nil {
		return State{IPs: map[string]Record{}}
	}
	var out State
	if err := json.Unmarshal(content, &out); err != nil {
		return State{IPs: map[string]Record{}}
	}
	if out.IPs == nil {
		out.IPs = map[string]Record{}
	}
	return out
}

func saveState(path string, st State) error {
	raw, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func newSourceBackend(cfg Config) sentinelsource.Backend {
	switch strings.ToLower(strings.TrimSpace(cfg.SourceBackend)) {
	case "redis":
		return sentinelsource.NewFallbackBackend(
			sentinelsource.NewRedisBackend(),
			sentinelsource.NewFileBackend(cfg.LogPath),
		)
	case "file":
		return sentinelsource.NewFileBackend(cfg.LogPath)
	default:
		return sentinelsource.NewFallbackBackend(
			sentinelsource.NewFileBackend(cfg.LogPath),
			sentinelsource.NewRedisBackend(),
		)
	}
}

func readNewLines(path string, offset int64) ([]parsedAccess, int64, error) {
	return readNewEvents(sentinelsource.NewFileBackend(path), offset)
}

func readNewEvents(backend sentinelsource.Backend, offset int64) ([]parsedAccess, int64, error) {
	items, nextOffset, err := backend.Read(offset)
	if err != nil {
		return nil, offset, err
	}
	out := make([]parsedAccess, 0, len(items))
	for _, item := range items {
		out = append(out, parsedAccess{
			ip:          item.IP,
			site:        normalizeSiteID(item.Site),
			status:      item.Status,
			method:      item.Method,
			path:        item.Path,
			userAgent:   item.UserAgent,
			ja3:         item.JA3,
			antibotFail: item.AntibotFail,
			when:        item.When,
		})
	}
	return out, nextOffset, nil
}

func parseAccessLine(line string) (parsedAccess, bool) {
	if strings.HasPrefix(line, "{") {
		var item jsonAccess
		if err := json.Unmarshal([]byte(line), &item); err == nil {
			ip := strings.TrimSpace(item.ClientIP)
			if ip != "" && item.Status > 0 {
				when, err := time.Parse(time.RFC3339, strings.TrimSpace(item.Timestamp))
				if err == nil {
					return parsedAccess{
						ip:          ip,
						site:        normalizeSiteID(item.Site),
						status:      item.Status,
						method:      strings.TrimSpace(item.Method),
						path:        strings.TrimSpace(item.URI),
						userAgent:   strings.TrimSpace(item.UserAgent),
						ja3:         strings.TrimSpace(item.JA3),
						antibotFail: item.AntibotFail,
						when:        when.UTC(),
					}, true
				}
			}
		}
	}

	matches := accessLogPattern.FindStringSubmatch(line)
	if len(matches) != 10 {
		return parsedAccess{}, false
	}
	ip := strings.TrimSpace(matches[1])
	if ip == "" {
		return parsedAccess{}, false
	}
	status, err := strconv.Atoi(matches[5])
	if err != nil {
		return parsedAccess{}, false
	}
	when, err := time.Parse("02/Jan/2006:15:04:05 -0700", matches[2])
	if err != nil {
		return parsedAccess{}, false
	}
	return parsedAccess{
		ip:        ip,
		site:      normalizeSiteID(matches[9]),
		status:    status,
		method:    strings.TrimSpace(matches[3]),
		path:      strings.TrimSpace(matches[4]),
		userAgent: strings.TrimSpace(matches[8]),
		when:      when.UTC(),
	}, true
}
