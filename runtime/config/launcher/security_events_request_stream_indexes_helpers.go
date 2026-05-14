package main

import (
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (s *requestStreamSource) indexesFromBackendsLocked(options requestQueryOptions) (map[string]any, bool) {
	mergedItems := make([]map[string]any, 0)
	hadBackend := false
	seen := map[string]struct{}{}
	total := 0

	if s.opensearch != nil {
		payload, err := s.opensearch.indexes(requestQueryOptions{Limit: 500, Offset: 0}, s.archiveRoot)
		if err == nil {
			hadBackend = true
			total += parseMapInt(payload, "total")
			for _, item := range parseIndexItems(payload) {
				key := indexItemKey(item)
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				mergedItems = append(mergedItems, item)
			}
		} else if !errors.Is(err, errOpenSearchDisabled) {
			s.lastIngestError = err.Error()
		}
	}
	if s.clickhouse != nil {
		payload, err := s.clickhouse.indexes(requestQueryOptions{Limit: 500, Offset: 0}, s.archiveRoot)
		if err == nil {
			hadBackend = true
			total += parseMapInt(payload, "total")
			for _, item := range parseIndexItems(payload) {
				key := indexItemKey(item)
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				mergedItems = append(mergedItems, item)
			}
		} else if !errors.Is(err, errClickHouseDisabled) {
			s.lastIngestError = err.Error()
		}
	}
	if !hadBackend {
		return nil, false
	}
	sort.Slice(mergedItems, func(i, j int) bool {
		return strings.TrimSpace(asString(mergedItems[i]["date"])) > strings.TrimSpace(asString(mergedItems[j]["date"]))
	})
	start := maxInt(options.Offset, 0)
	if start > len(mergedItems) {
		start = len(mergedItems)
	}
	end := len(mergedItems)
	if options.Limit > 0 && start+options.Limit < end {
		end = start + options.Limit
	}
	return map[string]any{
		"items":             mergedItems[start:end],
		"total":             len(mergedItems),
		"limit":             options.Limit,
		"offset":            options.Offset,
		"archive_root":      s.archiveRoot,
		"storage_type":      "tiered",
		"last_ingest_error": s.lastIngestError,
	}, true
}

func reverseRequestRows(items []map[string]any) {
	for left, right := 0, len(items)-1; left < right; left, right = left+1, right-1 {
		items[left], items[right] = items[right], items[left]
	}
}

func appendRequestRowsDedup(target *[]map[string]any, seen map[string]struct{}, items []map[string]any) {
	for _, item := range items {
		key := requestRowKey(item)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		*target = append(*target, item)
	}
}

func requestRowKey(item map[string]any) string {
	entry, _ := item["entry"].(map[string]any)
	return strings.Join([]string{
		strings.TrimSpace(asString(item["stream"])),
		strings.TrimSpace(asString(item["ingested_at"])),
		strings.TrimSpace(asString(entry["timestamp"])),
		strings.TrimSpace(asString(entry["request_id"])),
		strings.TrimSpace(asString(entry["client_ip"])),
		strings.TrimSpace(asString(entry["uri"])),
		strconv.Itoa(parseIntValue(entry["status"])),
	}, "|")
}

func requestRowTimestamp(item map[string]any) time.Time {
	entry, _ := item["entry"].(map[string]any)
	for _, raw := range []string{
		strings.TrimSpace(asString(entry["timestamp"])),
		strings.TrimSpace(asString(item["ingested_at"])),
	} {
		if raw == "" {
			continue
		}
		if parsed, err := time.Parse(time.RFC3339Nano, raw); err == nil {
			return parsed.UTC()
		}
		if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
			return parsed.UTC()
		}
	}
	return time.Time{}
}

func parseIndexItems(payload map[string]any) []map[string]any {
	raw, _ := payload["items"].([]map[string]any)
	if raw != nil {
		return raw
	}
	itemsRaw, _ := payload["items"].([]any)
	out := make([]map[string]any, 0, len(itemsRaw))
	for _, item := range itemsRaw {
		typed, _ := item.(map[string]any)
		if typed != nil {
			out = append(out, typed)
		}
	}
	return out
}

func parseMapInt(payload map[string]any, key string) int {
	if payload == nil {
		return 0
	}
	return parseIntValue(payload[key])
}

func indexItemKey(item map[string]any) string {
	return strings.Join([]string{
		strings.TrimSpace(asString(item["date"])),
		strings.TrimSpace(asString(item["storage_type"])),
		strings.TrimSpace(asString(item["file_name"])),
	}, "|")
}

func asString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []byte:
		return string(typed)
	default:
		return ""
	}
}
