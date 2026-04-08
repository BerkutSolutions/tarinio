package services

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type RuntimeRequestCollector interface {
	Collect() ([]map[string]any, error)
}

type HTTPRuntimeRequestCollector struct {
	URL    string
	Client *http.Client
}

func NewHTTPRuntimeRequestCollector(healthURL string) *HTTPRuntimeRequestCollector {
	return &HTTPRuntimeRequestCollector{
		URL: deriveRuntimeRequestsURL(healthURL),
		Client: &http.Client{
			Timeout: 2 * time.Second,
		},
	}
}

func (c *HTTPRuntimeRequestCollector) Collect() ([]map[string]any, error) {
	return c.CollectWithOptions(nil)
}

func (c *HTTPRuntimeRequestCollector) CollectWithOptions(query url.Values) ([]map[string]any, error) {
	if c == nil || strings.TrimSpace(c.URL) == "" {
		return []map[string]any{}, nil
	}
	targetURL := c.URL
	if len(query) > 0 {
		if parsed, err := url.Parse(targetURL); err == nil {
			q := parsed.Query()
			for _, key := range []string{"limit", "offset", "since", "day", "retention_days"} {
				value := strings.TrimSpace(query.Get(key))
				if value == "" {
					continue
				}
				q.Set(key, value)
			}
			parsed.RawQuery = q.Encode()
			targetURL = parsed.String()
		}
	}
	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, err
	}
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return []map[string]any{}, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []map[string]any{}, nil
	}

	var list []map[string]any
	if err := json.Unmarshal(body, &list); err == nil {
		return list, nil
	}

	var wrapped struct {
		Requests []map[string]any `json:"requests"`
	}
	if err := json.Unmarshal(body, &wrapped); err == nil {
		return wrapped.Requests, nil
	}
	return []map[string]any{}, nil
}

func deriveRuntimeRequestsURL(healthURL string) string {
	raw := strings.TrimSpace(healthURL)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	parsed.Path = "/requests"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}
