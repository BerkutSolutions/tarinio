package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type TLSAutoRenewSettings struct {
	Enabled         bool   `json:"enabled"`
	RenewBeforeDays int    `json:"renew_before_days"`
	UpdatedAt       string `json:"updated_at,omitempty"`
}

type tlsAutoRenewSettingsStore struct {
	path string
	mu   sync.Mutex
}

func newTLSAutoRenewSettingsStore(root string) (*tlsAutoRenewSettingsStore, error) {
	if root == "" {
		return nil, errors.New("tls auto-renew settings root is required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create tls auto-renew settings root: %w", err)
	}
	return &tlsAutoRenewSettingsStore{path: filepath.Join(root, "settings.json")}, nil
}

func defaultTLSAutoRenewSettings() TLSAutoRenewSettings {
	return TLSAutoRenewSettings{Enabled: false, RenewBeforeDays: 30}
}

func (s *tlsAutoRenewSettingsStore) Get() (TLSAutoRenewSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	payload, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return defaultTLSAutoRenewSettings(), nil
		}
		return TLSAutoRenewSettings{}, fmt.Errorf("read tls auto-renew settings: %w", err)
	}
	var out TLSAutoRenewSettings
	if err := json.Unmarshal(payload, &out); err != nil {
		return TLSAutoRenewSettings{}, fmt.Errorf("decode tls auto-renew settings: %w", err)
	}
	if err := normalizeTLSAutoRenewSettings(&out); err != nil {
		return TLSAutoRenewSettings{}, err
	}
	return out, nil
}

func (s *tlsAutoRenewSettingsStore) Put(value TLSAutoRenewSettings) (TLSAutoRenewSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := normalizeTLSAutoRenewSettings(&value); err != nil {
		return TLSAutoRenewSettings{}, err
	}
	value.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return TLSAutoRenewSettings{}, fmt.Errorf("encode tls auto-renew settings: %w", err)
	}
	body = append(body, '\n')
	if err := os.WriteFile(s.path, body, 0o644); err != nil {
		return TLSAutoRenewSettings{}, fmt.Errorf("write tls auto-renew settings: %w", err)
	}
	return value, nil
}

func normalizeTLSAutoRenewSettings(value *TLSAutoRenewSettings) error {
	if value == nil {
		return errors.New("tls auto-renew settings are required")
	}
	if value.RenewBeforeDays <= 0 {
		value.RenewBeforeDays = 30
	}
	if value.RenewBeforeDays < 1 || value.RenewBeforeDays > 365 {
		return errors.New("renew_before_days must be between 1 and 365")
	}
	return nil
}
