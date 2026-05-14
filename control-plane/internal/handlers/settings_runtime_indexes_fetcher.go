package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func runtimeIndexesFromQuery(values url.Values) map[string]any {
	if strings.TrimSpace(values.Get("storage_indexes_limit")) == "" && strings.TrimSpace(values.Get("storage_indexes_offset")) == "" {
		return nil
	}
	stream := normalizeStorageIndexStream(values.Get("stream"))
	limit := 10
	offset := 0
	if raw := strings.TrimSpace(values.Get("storage_indexes_limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if raw := strings.TrimSpace(values.Get("storage_indexes_offset")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed >= 0 {
			offset = parsed
		}
	}
	indexes, err := fetchStorageIndexes(stream, limit, offset)
	if err != nil {
		return map[string]any{
			"stream": stream,
			"items":  []map[string]any{},
			"total":  0,
			"limit":  limit,
			"offset": offset,
			"error":  err.Error(),
		}
	}
	return indexes
}

func normalizeStorageIndexStream(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "events":
		return "events"
	case "activity":
		return "activity"
	default:
		return "requests"
	}
}

func deriveRuntimeIndexesURL(healthURL string) string {
	raw := strings.TrimSpace(healthURL)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	parsed.Path = "/requests/indexes"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func (f *runtimeIndexFetcher) Fetch(stream string, limit int, offset int) (map[string]any, error) {
	if f == nil || strings.TrimSpace(f.url) == "" {
		return map[string]any{
			"stream": stream,
			"items":  []map[string]any{},
			"total":  0,
			"limit":  limit,
			"offset": offset,
		}, nil
	}
	target, err := url.Parse(f.url)
	if err != nil {
		return nil, err
	}
	q := target.Query()
	q.Set("limit", strconv.Itoa(limit))
	q.Set("offset", strconv.Itoa(offset))
	if strings.TrimSpace(stream) != "" {
		q.Set("stream", stream)
	}
	target.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, err
	}
	client := f.client
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("runtime indexes endpoint returned %d", resp.StatusCode)
	}
	var payload map[string]any
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (f *runtimeIndexFetcher) Delete(stream string, date string) error {
	if f == nil || strings.TrimSpace(f.url) == "" {
		return nil
	}
	day := strings.TrimSpace(date)
	if day == "" {
		return fmt.Errorf("date is required")
	}
	if _, err := time.Parse("2006-01-02", day); err != nil {
		return fmt.Errorf("date must be in YYYY-MM-DD format")
	}

	target, err := url.Parse(f.url)
	if err != nil {
		return err
	}
	q := target.Query()
	q.Set("date", day)
	if strings.TrimSpace(stream) != "" {
		q.Set("stream", stream)
	}
	target.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodDelete, target.String(), nil)
	if err != nil {
		return err
	}
	client := f.client
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("runtime indexes delete endpoint returned %d", resp.StatusCode)
	}
	return nil
}

func fetchStorageIndexes(stream string, limit int, offset int) (map[string]any, error) {
	stream = normalizeStorageIndexStream(stream)
	if stream == "requests" {
		if runtimeRequestIndexes == nil {
			return map[string]any{
				"stream": stream,
				"items":  []map[string]any{},
				"total":  0,
				"limit":  limit,
				"offset": offset,
			}, nil
		}
		return runtimeRequestIndexes.Fetch(stream, limit, offset)
	}
	runtimeSettingsState.mu.RLock()
	logging := runtimeSettingsState.logging
	pepper := runtimeSettingsState.pepper
	runtimeSettingsState.mu.RUnlock()
	return fetchOpenSearchStorageIndexes(stream, logging, pepper, limit, offset)
}

func deleteStorageIndexes(stream string, day string) error {
	stream = normalizeStorageIndexStream(stream)
	if stream == "requests" {
		if runtimeRequestIndexes == nil {
			return fmt.Errorf("runtime indexes fetcher unavailable")
		}
		return runtimeRequestIndexes.Delete(stream, day)
	}
	runtimeSettingsState.mu.RLock()
	logging := runtimeSettingsState.logging
	pepper := runtimeSettingsState.pepper
	runtimeSettingsState.mu.RUnlock()
	return deleteOpenSearchStorageIndex(stream, day, logging, pepper)
}
