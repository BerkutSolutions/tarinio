package vaultkv

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	Address       string
	Token         string
	Mount         string
	PathPrefix    string
	TLSSkipVerify bool
	HTTPClient    *http.Client
}

func (c Client) secretPath(name string) string {
	base := strings.Trim(strings.TrimSpace(c.PathPrefix), "/")
	name = strings.Trim(strings.TrimSpace(name), "/")
	if base == "" {
		return name
	}
	if name == "" {
		return base
	}
	return base + "/" + name
}

func (c Client) apiURL(name string) string {
	address := strings.TrimRight(strings.TrimSpace(c.Address), "/")
	mount := strings.Trim(strings.TrimSpace(c.Mount), "/")
	if mount == "" {
		mount = "secret"
	}
	return address + "/v1/" + mount + "/data/" + c.secretPath(name)
}

func (c Client) client() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	if c.TLSSkipVerify {
		return &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
	}
	return &http.Client{Timeout: 10 * time.Second}
}

func (c Client) Get(name string, key string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, c.apiURL(name), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-Vault-Token", strings.TrimSpace(c.Token))
	resp, err := c.client().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return "", nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		return "", fmt.Errorf("vault get failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var payload struct {
		Data struct {
			Data map[string]any `json:"data"`
		} `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload); err != nil {
		return "", err
	}
	if payload.Data.Data == nil {
		return "", nil
	}
	value, _ := payload.Data.Data[key].(string)
	return strings.TrimSpace(value), nil
}

func (c Client) Put(name string, values map[string]string) error {
	data := make(map[string]string, len(values))
	for key, value := range values {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		data[trimmedKey] = strings.TrimSpace(value)
	}
	body, err := json.Marshal(map[string]any{
		"data": data,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, c.apiURL(name), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Vault-Token", strings.TrimSpace(c.Token))
	resp, err := c.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		content, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		return fmt.Errorf("vault put failed: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(content)))
	}
	return nil
}
