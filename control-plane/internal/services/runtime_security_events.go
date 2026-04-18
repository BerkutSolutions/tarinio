package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"waf/control-plane/internal/events"
)

type RuntimeSecurityEventCollector interface {
	Collect() ([]events.Event, error)
}

type RuntimeSecurityEventProber interface {
	Probe() error
}

type HTTPRuntimeSecurityEventCollector struct {
	URL    string
	Client *http.Client
}

func NewHTTPRuntimeSecurityEventCollector(healthURL string) *HTTPRuntimeSecurityEventCollector {
	return &HTTPRuntimeSecurityEventCollector{
		URL: deriveRuntimeSecurityEventsURL(healthURL),
		Client: &http.Client{
			Timeout: 2 * time.Second,
		},
	}
}

func (c *HTTPRuntimeSecurityEventCollector) Collect() ([]events.Event, error) {
	if c == nil || strings.TrimSpace(c.URL) == "" {
		return nil, nil
	}
	req, err := http.NewRequest(http.MethodGet, c.URL, nil)
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
}

func (c *HTTPRuntimeSecurityEventCollector) Probe() error {
	if c == nil || strings.TrimSpace(c.URL) == "" {
		return nil
	}
	targetURL := deriveRuntimeSecurityEventsProbeURL(c.URL)
	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return err
	}
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("runtime security events probe endpoint returned %d", resp.StatusCode)
	}
	return nil
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
