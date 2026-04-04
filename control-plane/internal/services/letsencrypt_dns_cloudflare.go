package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/challenge/dns01"
)

type cloudflareDNSProvider struct {
	apiToken string
	client   *http.Client
}

var _ challenge.Provider = (*cloudflareDNSProvider)(nil)

func newCloudflareDNSProvider(apiToken string) (*cloudflareDNSProvider, error) {
	apiToken = strings.TrimSpace(apiToken)
	if apiToken == "" {
		return nil, fmt.Errorf("cloudflare api token is required")
	}
	return &cloudflareDNSProvider{
		apiToken: apiToken,
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
	}, nil
}

func (p *cloudflareDNSProvider) Present(domain, _, keyAuth string) error {
	info := dns01.GetChallengeInfo(domain, keyAuth)
	zoneName, zoneID, err := p.findZone(info.EffectiveFQDN)
	if err != nil {
		return err
	}
	recordName := strings.TrimSuffix(strings.TrimSuffix(info.EffectiveFQDN, "."), "."+zoneName)
	if recordName == "" {
		recordName = "@"
	}
	payload := map[string]any{
		"type":    "TXT",
		"name":    strings.TrimSuffix(info.EffectiveFQDN, "."),
		"content": info.Value,
		"ttl":     120,
	}
	if err := p.requestJSON(http.MethodPost, "/zones/"+zoneID+"/dns_records", payload, nil); err != nil {
		return fmt.Errorf("cloudflare create txt record %s for %s: %w", recordName, zoneName, err)
	}
	return nil
}

func (p *cloudflareDNSProvider) CleanUp(domain, _, keyAuth string) error {
	info := dns01.GetChallengeInfo(domain, keyAuth)
	_, zoneID, err := p.findZone(info.EffectiveFQDN)
	if err != nil {
		return err
	}
	params := url.Values{}
	params.Set("type", "TXT")
	params.Set("name", strings.TrimSuffix(info.EffectiveFQDN, "."))
	params.Set("content", info.Value)
	var listResp struct {
		Result []struct {
			ID string `json:"id"`
		} `json:"result"`
	}
	if err := p.requestJSON(http.MethodGet, "/zones/"+zoneID+"/dns_records?"+params.Encode(), nil, &listResp); err != nil {
		return fmt.Errorf("cloudflare list txt records: %w", err)
	}
	for _, item := range listResp.Result {
		if strings.TrimSpace(item.ID) == "" {
			continue
		}
		if err := p.requestJSON(http.MethodDelete, "/zones/"+zoneID+"/dns_records/"+item.ID, nil, nil); err != nil {
			return fmt.Errorf("cloudflare delete txt record %s: %w", item.ID, err)
		}
	}
	return nil
}

func (p *cloudflareDNSProvider) findZone(effectiveFQDN string) (string, string, error) {
	name := strings.TrimSuffix(strings.TrimSpace(effectiveFQDN), ".")
	parts := strings.Split(name, ".")
	for i := 0; i < len(parts)-1; i++ {
		candidate := strings.Join(parts[i:], ".")
		params := url.Values{}
		params.Set("name", candidate)
		params.Set("status", "active")
		var zoneResp struct {
			Result []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"result"`
		}
		if err := p.requestJSON(http.MethodGet, "/zones?"+params.Encode(), nil, &zoneResp); err != nil {
			return "", "", fmt.Errorf("cloudflare zone lookup for %s: %w", candidate, err)
		}
		if len(zoneResp.Result) > 0 && strings.TrimSpace(zoneResp.Result[0].ID) != "" {
			return strings.TrimSpace(zoneResp.Result[0].Name), strings.TrimSpace(zoneResp.Result[0].ID), nil
		}
	}
	return "", "", fmt.Errorf("cloudflare zone not found for %s", effectiveFQDN)
}

func (p *cloudflareDNSProvider) requestJSON(method, path string, payload any, out any) error {
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, "https://api.cloudflare.com/client/v4"+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	rawBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}

	var envelope struct {
		Success bool `json:"success"`
		Errors  []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if len(rawBody) > 0 {
		_ = json.Unmarshal(rawBody, &envelope)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 || !envelope.Success {
		message := strings.TrimSpace(resp.Status)
		if len(envelope.Errors) > 0 && strings.TrimSpace(envelope.Errors[0].Message) != "" {
			message = strings.TrimSpace(envelope.Errors[0].Message)
		}
		if message == "" {
			message = "request failed"
		}
		return fmt.Errorf("%s", message)
	}

	if out != nil && len(rawBody) > 0 {
		if err := json.Unmarshal(rawBody, out); err != nil {
			return err
		}
	}
	return nil
}
