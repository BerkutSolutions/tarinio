package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type RuntimeCRSStatus struct {
	ActiveVersion           string `json:"active_version"`
	ActivePath              string `json:"active_path"`
	SystemFallbackPath      string `json:"system_fallback_path"`
	LatestVersion           string `json:"latest_version"`
	LatestReleaseURL        string `json:"latest_release_url"`
	LastCheckedAt           string `json:"last_checked_at"`
	HasUpdate               bool   `json:"has_update"`
	HourlyAutoUpdateEnabled bool   `json:"hourly_auto_update_enabled"`
	FirstStartPending       bool   `json:"first_start_pending"`
	LastError               string `json:"last_error,omitempty"`
}

type RuntimeCRSService struct {
	BaseURL string
	Client  *http.Client
	Token   string
}

func NewRuntimeCRSService(baseURL string, token string) *RuntimeCRSService {
	return &RuntimeCRSService{
		BaseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		Client:  &http.Client{Timeout: 20 * time.Second},
		Token:   strings.TrimSpace(token),
	}
}

func RuntimeBaseURLFromHealthURL(healthURL string) string {
	healthURL = strings.TrimSpace(healthURL)
	if healthURL == "" {
		return "http://127.0.0.1:8081"
	}
	parsed, err := url.Parse(healthURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "http://127.0.0.1:8081"
	}
	return parsed.Scheme + "://" + parsed.Host
}

func (s *RuntimeCRSService) Status(ctx context.Context) (RuntimeCRSStatus, error) {
	var out RuntimeCRSStatus
	err := s.requestJSON(ctx, http.MethodGet, "/crs/status", nil, &out)
	return out, err
}

func (s *RuntimeCRSService) CheckUpdates(ctx context.Context, dryRun bool) (RuntimeCRSStatus, error) {
	var out RuntimeCRSStatus
	err := s.requestJSON(ctx, http.MethodPost, "/crs/check-updates", map[string]any{
		"dry_run": dryRun,
	}, &out)
	return out, err
}

func (s *RuntimeCRSService) Update(ctx context.Context) (RuntimeCRSStatus, error) {
	var out RuntimeCRSStatus
	err := s.requestJSON(ctx, http.MethodPost, "/crs/update", map[string]any{}, &out)
	return out, err
}

func (s *RuntimeCRSService) SetHourlyAutoUpdate(ctx context.Context, enabled bool) (RuntimeCRSStatus, error) {
	var out RuntimeCRSStatus
	err := s.requestJSON(ctx, http.MethodPost, "/crs/update", map[string]any{
		"enable_hourly_auto_update": enabled,
	}, &out)
	return out, err
}

func (s *RuntimeCRSService) requestJSON(ctx context.Context, method, path string, payload any, out any) error {
	base := strings.TrimRight(strings.TrimSpace(s.BaseURL), "/")
	if base == "" {
		base = "http://127.0.0.1:8081"
	}
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, base+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	setRuntimeAuthHeader(req, s.Token)
	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		msg := strings.TrimSpace(string(raw))
		if msg == "" {
			msg = fmt.Sprintf("runtime CRS endpoint returned status %d", resp.StatusCode)
		}
		return fmt.Errorf("%s", msg)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(out)
}
