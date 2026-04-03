package tlsconfigs

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// TLSConfig binds a site to certificate metadata in control-plane state.
type TLSConfig struct {
	SiteID        string `json:"site_id"`
	CertificateID string `json:"certificate_id"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

type state struct {
	TLSConfigs []TLSConfig `json:"tls_configs"`
}

// Store persists TLS configs keyed by site id without runtime coupling.
type Store struct {
	path string
	mu   sync.Mutex
}

func NewStore(root string) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("tls configs store root is required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create tls configs store root: %w", err)
	}
	return &Store{path: filepath.Join(root, "tls_configs.json")}, nil
}

func (s *Store) Create(item TLSConfig) (TLSConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item = normalizeTLSConfig(item)
	if err := validateTLSConfig(item); err != nil {
		return TLSConfig{}, err
	}

	current, err := s.loadLocked()
	if err != nil {
		return TLSConfig{}, err
	}
	for _, existing := range current.TLSConfigs {
		if existing.SiteID == item.SiteID {
			return TLSConfig{}, fmt.Errorf("tls config for site %s already exists", item.SiteID)
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	item.CreatedAt = now
	item.UpdatedAt = now
	current.TLSConfigs = append(current.TLSConfigs, item)
	sortTLSConfigs(current.TLSConfigs)
	if err := s.saveLocked(current); err != nil {
		return TLSConfig{}, err
	}
	return item, nil
}

func (s *Store) List() ([]TLSConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	items := append([]TLSConfig(nil), current.TLSConfigs...)
	sortTLSConfigs(items)
	return items, nil
}

func (s *Store) Update(item TLSConfig) (TLSConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item = normalizeTLSConfig(item)
	if err := validateTLSConfig(item); err != nil {
		return TLSConfig{}, err
	}

	current, err := s.loadLocked()
	if err != nil {
		return TLSConfig{}, err
	}
	for i := range current.TLSConfigs {
		if current.TLSConfigs[i].SiteID != item.SiteID {
			continue
		}
		item.CreatedAt = current.TLSConfigs[i].CreatedAt
		item.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		current.TLSConfigs[i] = item
		sortTLSConfigs(current.TLSConfigs)
		if err := s.saveLocked(current); err != nil {
			return TLSConfig{}, err
		}
		return item, nil
	}
	return TLSConfig{}, fmt.Errorf("tls config for site %s not found", item.SiteID)
}

func (s *Store) Delete(siteID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	siteID = normalizeID(siteID)
	if siteID == "" {
		return errors.New("tls config site_id is required")
	}

	current, err := s.loadLocked()
	if err != nil {
		return err
	}
	for i := range current.TLSConfigs {
		if current.TLSConfigs[i].SiteID != siteID {
			continue
		}
		current.TLSConfigs = append(current.TLSConfigs[:i], current.TLSConfigs[i+1:]...)
		sortTLSConfigs(current.TLSConfigs)
		return s.saveLocked(current)
	}
	return fmt.Errorf("tls config for site %s not found", siteID)
}

func (s *Store) loadLocked() (*state, error) {
	content, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &state{}, nil
		}
		return nil, fmt.Errorf("read tls configs store: %w", err)
	}

	var current state
	if err := json.Unmarshal(content, &current); err != nil {
		return nil, fmt.Errorf("decode tls configs store: %w", err)
	}
	sortTLSConfigs(current.TLSConfigs)
	return &current, nil
}

func (s *Store) saveLocked(current *state) error {
	content, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return fmt.Errorf("encode tls configs store: %w", err)
	}
	content = append(content, '\n')
	if err := os.WriteFile(s.path, content, 0o644); err != nil {
		return fmt.Errorf("write tls configs store: %w", err)
	}
	return nil
}

func validateTLSConfig(item TLSConfig) error {
	if item.SiteID == "" {
		return errors.New("tls config site_id is required")
	}
	if item.CertificateID == "" {
		return errors.New("tls config certificate_id is required")
	}
	return nil
}

func normalizeTLSConfig(item TLSConfig) TLSConfig {
	item.SiteID = normalizeID(item.SiteID)
	item.CertificateID = normalizeID(item.CertificateID)
	return item
}

func normalizeID(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func sortTLSConfigs(items []TLSConfig) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].SiteID < items[j].SiteID
	})
}
