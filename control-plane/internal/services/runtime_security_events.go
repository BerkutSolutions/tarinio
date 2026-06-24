package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"waf/control-plane/internal/events"
)

const runtimeSecurityEventsAttemptTimeout = 1200 * time.Millisecond

type RuntimeSecurityEventCollector interface {
	Collect() ([]events.Event, error)
}

type RuntimeSecurityEventProber interface {
	Probe() error
}

type HTTPRuntimeSecurityEventCollector struct {
	URL    string
	Client *http.Client
	Token  string
}

func NewHTTPRuntimeSecurityEventCollector(healthURL string, token string) *HTTPRuntimeSecurityEventCollector {
	return &HTTPRuntimeSecurityEventCollector{
		URL: deriveRuntimeSecurityEventsURL(healthURL),
		Client: &http.Client{
			Timeout: 2 * time.Second,
		},
		Token: strings.TrimSpace(token),
	}
}

func (c *HTTPRuntimeSecurityEventCollector) Collect() ([]events.Event, error) {
	if c == nil || strings.TrimSpace(c.URL) == "" {
		return nil, nil
	}
	var lastErr error
	for _, candidate := range runtimeEndpointCandidates(c.URL, "http://127.0.0.1:8081/security-events") {
		reqCtx, cancel := context.WithTimeout(context.Background(), runtimeSecurityEventsAttemptTimeout)
		req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, candidate, nil)
		if err != nil {
			cancel()
			lastErr = err
			continue
		}
		setRuntimeAuthHeader(req, c.Token)
		client := c.Client
		if client == nil {
			client = &http.Client{Timeout: 2 * time.Second}
		}
		resp, err := client.Do(req)
		if err != nil {
			cancel()
			lastErr = err
			continue
		}
		items, err := func() ([]events.Event, error) {
			defer cancel()
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return nil, fmt.Errorf("runtime security events endpoint returned %d", resp.StatusCode)
			}

			var payload struct {
				Events []struct {
					ID              string         `json:"id"`
					Type            string         `json:"type"`
					Severity        string         `json:"severity"`
					SiteID          string         `json:"site_id"`
					SourceComponent string         `json:"source_component"`
					OccurredAt      string         `json:"occurred_at"`
					Summary         string         `json:"summary"`
					Details         map[string]any `json:"details"`
				} `json:"events"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
				return nil, err
			}

			out := make([]events.Event, 0, len(payload.Events))
			for _, item := range payload.Events {
				siteID := strings.TrimSpace(item.SiteID)
				siteToken := strings.ToLower(strings.ReplaceAll(siteID, "_", "-"))
				if siteToken == "control-plane-access" || siteToken == "control-plane" || siteToken == "ui" {
					continue
				}
				eventType := events.Type(strings.TrimSpace(item.Type))
				switch eventType {
				case events.TypeSecurityWAF, events.TypeSecurityRateLimit, events.TypeSecurityAccess:
				default:
					eventType = events.TypeSecurityWAF
				}
				severity := events.Severity(strings.TrimSpace(item.Severity))
				switch severity {
				case events.SeverityInfo, events.SeverityWarning, events.SeverityError:
				default:
					severity = events.SeverityWarning
				}
				occurredAt := strings.TrimSpace(item.OccurredAt)
				if occurredAt == "" {
					occurredAt = time.Now().UTC().Format(time.RFC3339Nano)
				}
				out = append(out, events.Event{
					ID:              strings.TrimSpace(item.ID),
					Type:            eventType,
					Severity:        severity,
					SiteID:          siteID,
					SourceComponent: firstNonEmpty(strings.TrimSpace(item.SourceComponent), "runtime-nginx"),
					OccurredAt:      occurredAt,
					Summary:         firstNonEmpty(strings.TrimSpace(item.Summary), "runtime security event"),
					Details:         item.Details,
				})
			}
			return out, nil
		}()
		if err == nil {
			return items, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func (c *HTTPRuntimeSecurityEventCollector) Probe() error {
	if c == nil || strings.TrimSpace(c.URL) == "" {
		return nil
	}
	var lastErr error
	for _, candidate := range runtimeEndpointCandidates(deriveRuntimeSecurityEventsProbeURL(c.URL), "http://127.0.0.1:8081/security-events/probe") {
		reqCtx, cancel := context.WithTimeout(context.Background(), runtimeSecurityEventsAttemptTimeout)
		req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, candidate, nil)
		if err != nil {
			cancel()
			lastErr = err
			continue
		}
		setRuntimeAuthHeader(req, c.Token)
		client := c.Client
		if client == nil {
			client = &http.Client{Timeout: 2 * time.Second}
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
				lastErr = fmt.Errorf("runtime security events probe endpoint returned %d", resp.StatusCode)
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

func deriveRuntimeSecurityEventsURL(healthURL string) string {
	raw := strings.TrimSpace(healthURL)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	parsed.Path = "/security-events"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func deriveRuntimeSecurityEventsProbeURL(targetURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(targetURL))
	if err != nil {
		return strings.TrimSpace(targetURL)
	}
	parsed.Path = "/security-events/probe"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
