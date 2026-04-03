package ratelimitpolicies

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

// Limits is the minimal Stage 1 rate-limit declaration mapped later by the compiler.
type Limits struct {
	RequestsPerSecond int `json:"requests_per_second"`
	Burst             int `json:"burst"`
}

// RateLimitPolicy is the minimal Stage 1 control-plane rate-limit entity.
type RateLimitPolicy struct {
	ID        string `json:"id"`
	SiteID    string `json:"site_id"`
	Enabled   bool   `json:"enabled"`
	Limits    Limits `json:"limits"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type state struct {
	Policies []RateLimitPolicy `json:"rate_limit_policies"`
}

// Store persists rate-limit policies without runtime coupling.
type Store struct {
	path string
	mu   sync.Mutex
}

func NewStore(root string) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("rate limit policies store root is required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create rate limit policies store root: %w", err)
	}
	return &Store{path: filepath.Join(root, "rate_limit_policies.json")}, nil
}

func (s *Store) Create(item RateLimitPolicy) (RateLimitPolicy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item = normalizeRateLimitPolicy(item)
	if err := validateRateLimitPolicy(item); err != nil {
		return RateLimitPolicy{}, err
	}

	current, err := s.loadLocked()
	if err != nil {
		return RateLimitPolicy{}, err
	}
	for _, existing := range current.Policies {
		if existing.ID == item.ID {
			return RateLimitPolicy{}, fmt.Errorf("rate limit policy %s already exists", item.ID)
		}
		if existing.SiteID == item.SiteID {
			return RateLimitPolicy{}, fmt.Errorf("rate limit policy for site %s already exists", item.SiteID)
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	item.CreatedAt = now
	item.UpdatedAt = now
	current.Policies = append(current.Policies, item)
	sortPolicies(current.Policies)
	if err := s.saveLocked(current); err != nil {
		return RateLimitPolicy{}, err
	}
	return item, nil
}

func (s *Store) List() ([]RateLimitPolicy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	items := append([]RateLimitPolicy(nil), current.Policies...)
	sortPolicies(items)
	return items, nil
}

func (s *Store) Update(item RateLimitPolicy) (RateLimitPolicy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item = normalizeRateLimitPolicy(item)
	if err := validateRateLimitPolicy(item); err != nil {
		return RateLimitPolicy{}, err
	}

	current, err := s.loadLocked()
	if err != nil {
		return RateLimitPolicy{}, err
	}
	for _, existing := range current.Policies {
		if existing.SiteID == item.SiteID && existing.ID != item.ID {
			return RateLimitPolicy{}, fmt.Errorf("rate limit policy for site %s already exists", item.SiteID)
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
			return RateLimitPolicy{}, err
		}
		return item, nil
	}
	return RateLimitPolicy{}, fmt.Errorf("rate limit policy %s not found", item.ID)
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id = normalizeID(id)
	if id == "" {
		return errors.New("rate limit policy id is required")
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
	return fmt.Errorf("rate limit policy %s not found", id)
}

func (s *Store) loadLocked() (*state, error) {
	content, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &state{}, nil
		}
		return nil, fmt.Errorf("read rate limit policies store: %w", err)
	}

	var current state
	if err := json.Unmarshal(content, &current); err != nil {
		return nil, fmt.Errorf("decode rate limit policies store: %w", err)
	}
	sortPolicies(current.Policies)
	return &current, nil
}

func (s *Store) saveLocked(current *state) error {
	content, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return fmt.Errorf("encode rate limit policies store: %w", err)
	}
	content = append(content, '\n')
	if err := os.WriteFile(s.path, content, 0o644); err != nil {
		return fmt.Errorf("write rate limit policies store: %w", err)
	}
	return nil
}

func validateRateLimitPolicy(item RateLimitPolicy) error {
	if item.ID == "" {
		return errors.New("rate limit policy id is required")
	}
	if item.SiteID == "" {
		return errors.New("rate limit policy site_id is required")
	}
	if item.Enabled {
		if item.Limits.RequestsPerSecond <= 0 {
			return errors.New("rate limit policy requests_per_second must be greater than zero when enabled")
		}
		if item.Limits.Burst < 0 {
			return errors.New("rate limit policy burst must be zero or greater")
		}
		return nil
	}
	if item.Limits.RequestsPerSecond < 0 {
		return errors.New("rate limit policy requests_per_second must be zero or greater")
	}
	if item.Limits.Burst < 0 {
		return errors.New("rate limit policy burst must be zero or greater")
	}
	return nil
}

func normalizeRateLimitPolicy(item RateLimitPolicy) RateLimitPolicy {
	item.ID = normalizeID(item.ID)
	item.SiteID = normalizeID(item.SiteID)
	return item
}

func normalizeID(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func sortPolicies(items []RateLimitPolicy) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
}
