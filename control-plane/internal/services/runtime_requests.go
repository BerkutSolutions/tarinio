package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const runtimeRequestsHTTPTimeout = 8 * time.Second
const runtimeRequestsCollectAttemptTimeout = 2 * time.Second
const runtimeRequestsProbeAttemptTimeout = 1200 * time.Millisecond

type RuntimeRequestCollector interface {
	Collect() ([]map[string]any, error)
}

type RuntimeRequestProber interface {
	Probe(query url.Values) error
}

type HTTPRuntimeRequestCollector struct {
	URL    string
	Client *http.Client
	Token  string
}

func NewHTTPRuntimeRequestCollector(healthURL string, token string) *HTTPRuntimeRequestCollector {
	return &HTTPRuntimeRequestCollector{
		URL: deriveRuntimeRequestsURL(healthURL),
		Client: &http.Client{
			Timeout: runtimeRequestsHTTPTimeout,
		},
		Token: strings.TrimSpace(token),
	}
}

func (c *HTTPRuntimeRequestCollector) Collect() ([]map[string]any, error) {
	return c.CollectWithOptions(nil)
}

func (c *HTTPRuntimeRequestCollector) CollectWithOptions(query url.Values) ([]map[string]any, error) {
	if c == nil || strings.TrimSpace(c.URL) == "" {
		return []map[string]any{}, nil
	}
	var lastErr error
	for _, baseURL := range runtimeEndpointCandidates(c.URL, "http://127.0.0.1:8081/requests") {
		targetURL := baseURL
		if len(query) > 0 {
			if parsed, err := url.Parse(targetURL); err == nil {
				q := parsed.Query()
				for _, key := range []string{"limit", "offset", "since", "day", "retention_days", "probe"} {
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
		reqCtx, cancel := context.WithTimeout(context.Background(), runtimeRequestsCollectAttemptTimeout)
		req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, targetURL, nil)
		if err != nil {
			cancel()
			lastErr = err
			continue
		}
		setRuntimeAuthHeader(req, c.Token)
		client := c.Client
		if client == nil {
			client = &http.Client{Timeout: runtimeRequestsHTTPTimeout}
		}
		resp, err := client.Do(req)
		if err != nil {
			cancel()
			lastErr = err
			continue
		}
		items, err := func() ([]map[string]any, error) {
			defer cancel()
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return nil, fmt.Errorf("runtime requests endpoint returned %d", resp.StatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, err
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
			return nil, fmt.Errorf("decode runtime requests payload")
		}()
		if err == nil {
			return items, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func (c *HTTPRuntimeRequestCollector) Probe(query url.Values) error {
	if c == nil || strings.TrimSpace(c.URL) == "" {
		return nil
	}
	var lastErr error
	for _, baseURL := range runtimeEndpointCandidates(deriveRuntimeRequestsProbeURL(c.URL), "http://127.0.0.1:8081/requests/probe") {
		targetURL := baseURL
		if len(query) > 0 {
			if parsed, err := url.Parse(targetURL); err == nil {
				q := parsed.Query()
				for _, key := range []string{"retention_days", "day"} {
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
		reqCtx, cancel := context.WithTimeout(context.Background(), runtimeRequestsProbeAttemptTimeout)
		req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, targetURL, nil)
		if err != nil {
			cancel()
			lastErr = err
			continue
		}
		setRuntimeAuthHeader(req, c.Token)
		client := c.Client
		if client == nil {
			client = &http.Client{Timeout: runtimeRequestsHTTPTimeout}
		}
		resp, err := client.Do(req)
		if err != nil {
			cancel()
			lastErr = err
			continue
		}
		func() {
			defer cancel()
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				lastErr = fmt.Errorf("runtime requests probe endpoint returned %d", resp.StatusCode)
				return
			}
			lastErr = nil
		}()
		if lastErr == nil {
			return nil
		}
	}
	return lastErr
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

func deriveRuntimeRequestsProbeURL(targetURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(targetURL))
	if err != nil {
		return strings.TrimSpace(targetURL)
	}
	parsed.Path = "/requests/probe"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}
