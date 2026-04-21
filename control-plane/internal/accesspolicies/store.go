package accesspolicies

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"waf/control-plane/internal/storage"
)

// AccessPolicy is the minimal Stage 1 control-plane access policy entity.
type AccessPolicy struct {
	ID        string   `json:"id"`
	SiteID    string   `json:"site_id"`
	Enabled   bool     `json:"enabled"`
	AllowList []string `json:"allowlist"`
	DenyList  []string `json:"denylist"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

type state struct {
	Policies []AccessPolicy `json:"access_policies"`
}

// Store persists access policies without runtime coupling.
type Store struct {
	state *storage.JSONState
	mu    sync.Mutex
}

func NewStore(root string) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("access policies store root is required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create access policies store root: %w", err)
	}
	return &Store{state: storage.NewFileJSONState(filepath.Join(root, "access_policies.json"))}, nil
}

func NewPostgresStore(root string, backend storage.Backend) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("access policies store root is required")
	}
	return &Store{
		state: storage.NewBackendJSONState(backend, "accesspolicies/access_policies.json", filepath.Join(root, "access_policies.json")),
	}, nil
}

func (s *Store) Create(item AccessPolicy) (AccessPolicy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item = normalizeAccessPolicy(item)
	if err := validateAccessPolicy(item); err != nil {
		return AccessPolicy{}, err
	}

	current, err := s.loadLocked()
	if err != nil {
		return AccessPolicy{}, err
	}
	for _, existing := range current.Policies {
		if existing.ID == item.ID {
			return AccessPolicy{}, fmt.Errorf("access policy %s already exists", item.ID)
		}
		if existing.SiteID == item.SiteID {
			return AccessPolicy{}, fmt.Errorf("access policy for site %s already exists", item.SiteID)
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	item.CreatedAt = now
	item.UpdatedAt = now
	current.Policies = append(current.Policies, item)
	sortPolicies(current.Policies)
	if err := s.saveLocked(current); err != nil {
		return AccessPolicy{}, err
	}
	return item, nil
}

func (s *Store) List() ([]AccessPolicy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	items := append([]AccessPolicy(nil), current.Policies...)
	sortPolicies(items)
	return items, nil
}

func (s *Store) Update(item AccessPolicy) (AccessPolicy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item = normalizeAccessPolicy(item)
	if err := validateAccessPolicy(item); err != nil {
		return AccessPolicy{}, err
	}

	current, err := s.loadLocked()
	if err != nil {
		return AccessPolicy{}, err
	}
	for _, existing := range current.Policies {
		if existing.SiteID == item.SiteID && existing.ID != item.ID {
			return AccessPolicy{}, fmt.Errorf("access policy for site %s already exists", item.SiteID)
		}
	}
	for i := range current.Policies {
		if current.Policies[i].ID != item.ID {
			continue
		}
		item.CreatedAt = current.Policies[i].CreatedAt
		item.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		current.Policies[i] = item
		sortPolicies(current.Policies)
		if err := s.saveLocked(current); err != nil {
			return AccessPolicy{}, err
		}
		return item, nil
	}
	return AccessPolicy{}, fmt.Errorf("access policy %s not found", item.ID)
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id = normalizeID(id)
	if id == "" {
		return errors.New("access policy id is required")
	}

	current, err := s.loadLocked()
	if err != nil {
		return err
	}
	for i := range current.Policies {
		if current.Policies[i].ID != id {
			continue
		}
		current.Policies = append(current.Policies[:i], current.Policies[i+1:]...)
		sortPolicies(current.Policies)
		return s.saveLocked(current)
	}
	return fmt.Errorf("access policy %s not found", id)
}

func (s *Store) loadLocked() (*state, error) {
	content, err := s.state.Load()
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return &state{}, nil
		}
		return nil, fmt.Errorf("read access policies store: %w", err)
	}

	var current state
	if err := json.Unmarshal(content, &current); err != nil {
		return nil, fmt.Errorf("decode access policies store: %w", err)
	}
	sortPolicies(current.Policies)
	return &current, nil
}

func (s *Store) saveLocked(current *state) error {
	content, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return fmt.Errorf("encode access policies store: %w", err)
	}
	content = append(content, '\n')
	if err := s.state.Save(content); err != nil {
		return fmt.Errorf("write access policies store: %w", err)
	}
	return nil
}

func validateAccessPolicy(item AccessPolicy) error {
	if item.ID == "" {
		return errors.New("access policy id is required")
	}
	if item.SiteID == "" {
		return errors.New("access policy site_id is required")
	}
	for _, value := range item.AllowList {
		if err := validateCIDROrIP(value); err != nil {
			return fmt.Errorf("access policy allowlist contains invalid value %q", value)
		}
	}
	for _, value := range item.DenyList {
		if err := validateCIDROrIP(value); err != nil {
			return fmt.Errorf("access policy denylist contains invalid value %q", value)
		}
	}
	return nil
}

func normalizeAccessPolicy(item AccessPolicy) AccessPolicy {
	item.ID = normalizeID(item.ID)
	item.SiteID = normalizeID(item.SiteID)
	item.AllowList = normalizeCIDRList(item.AllowList)
	item.DenyList = normalizeCIDRList(item.DenyList)
	return item
}

func normalizeCIDRList(values []string) []string {
	items := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		items = append(items, value)
	}
	sort.Strings(items)
	return slices.Compact(items)
}

func validateCIDROrIP(value string) error {
	if ip := net.ParseIP(value); ip != nil {
		return nil
	}
	if _, _, err := net.ParseCIDR(value); err == nil {
		return nil
	}
	return errors.New("must be valid ip or cidr")
}

func normalizeID(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func sortPolicies(items []AccessPolicy) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
}
