package wafpolicies

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
)

type Mode string

const (
	ModeDetection  Mode = "detection"
	ModePrevention Mode = "prevention"
)

type RuleOverride struct {
	RuleID  string `json:"rule_id"`
	Enabled bool   `json:"enabled"`
}

// WAFPolicy is the minimal Stage 1 control-plane WAF policy entity.
type WAFPolicy struct {
	ID                 string         `json:"id"`
	SiteID             string         `json:"site_id"`
	Enabled            bool           `json:"enabled"`
	Mode               Mode           `json:"mode"`
	CRSEnabled         bool           `json:"crs_enabled"`
	CustomRuleIncludes []string       `json:"custom_rule_includes"`
	RuleOverrides      []RuleOverride `json:"rule_overrides"`
	CreatedAt          string         `json:"created_at"`
	UpdatedAt          string         `json:"updated_at"`
}

type state struct {
	Policies []WAFPolicy `json:"waf_policies"`
}

// Store persists WAF policies without compiler or runtime coupling.
type Store struct {
	path string
	mu   sync.Mutex
}

func NewStore(root string) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("waf policies store root is required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create waf policies store root: %w", err)
	}
	return &Store{path: filepath.Join(root, "waf_policies.json")}, nil
}

func (s *Store) Create(item WAFPolicy) (WAFPolicy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item = normalizeWAFPolicy(item)
	if err := validateWAFPolicy(item); err != nil {
		return WAFPolicy{}, err
	}

	current, err := s.loadLocked()
	if err != nil {
		return WAFPolicy{}, err
	}
	for _, existing := range current.Policies {
		if existing.ID == item.ID {
			return WAFPolicy{}, fmt.Errorf("waf policy %s already exists", item.ID)
		}
		if existing.SiteID == item.SiteID {
			return WAFPolicy{}, fmt.Errorf("waf policy for site %s already exists", item.SiteID)
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	item.CreatedAt = now
	item.UpdatedAt = now
	current.Policies = append(current.Policies, item)
	sortPolicies(current.Policies)
	if err := s.saveLocked(current); err != nil {
		return WAFPolicy{}, err
	}
	return item, nil
}

func (s *Store) List() ([]WAFPolicy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	items := append([]WAFPolicy(nil), current.Policies...)
	sortPolicies(items)
	return items, nil
}

func (s *Store) Update(item WAFPolicy) (WAFPolicy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item = normalizeWAFPolicy(item)
	if err := validateWAFPolicy(item); err != nil {
		return WAFPolicy{}, err
	}

	current, err := s.loadLocked()
	if err != nil {
		return WAFPolicy{}, err
	}

	for _, existing := range current.Policies {
		if existing.SiteID == item.SiteID && existing.ID != item.ID {
			return WAFPolicy{}, fmt.Errorf("waf policy for site %s already exists", item.SiteID)
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
			return WAFPolicy{}, err
		}
		return item, nil
	}

	return WAFPolicy{}, fmt.Errorf("waf policy %s not found", item.ID)
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id = normalizeID(id)
	if id == "" {
		return errors.New("waf policy id is required")
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
	return fmt.Errorf("waf policy %s not found", id)
}

func (s *Store) loadLocked() (*state, error) {
	content, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &state{}, nil
		}
		return nil, fmt.Errorf("read waf policies store: %w", err)
	}

	var current state
	if err := json.Unmarshal(content, &current); err != nil {
		return nil, fmt.Errorf("decode waf policies store: %w", err)
	}
	sortPolicies(current.Policies)
	return &current, nil
}

func (s *Store) saveLocked(current *state) error {
	content, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return fmt.Errorf("encode waf policies store: %w", err)
	}
	content = append(content, '\n')
	if err := os.WriteFile(s.path, content, 0o644); err != nil {
		return fmt.Errorf("write waf policies store: %w", err)
	}
	return nil
}

func validateWAFPolicy(item WAFPolicy) error {
	if item.ID == "" {
		return errors.New("waf policy id is required")
	}
	if item.SiteID == "" {
		return errors.New("waf policy site_id is required")
	}
	if item.Enabled {
		switch item.Mode {
		case ModeDetection, ModePrevention:
		default:
			return errors.New("waf policy mode must be detection or prevention")
		}
	}
	for _, include := range item.CustomRuleIncludes {
		if strings.TrimSpace(include) == "" {
			return errors.New("waf policy custom_rule_includes must not contain empty values")
		}
	}
	for _, override := range item.RuleOverrides {
		if strings.TrimSpace(override.RuleID) == "" {
			return errors.New("waf policy rule_overrides must define rule_id")
		}
	}
	return nil
}

func normalizeWAFPolicy(item WAFPolicy) WAFPolicy {
	item.ID = normalizeID(item.ID)
	item.SiteID = normalizeID(item.SiteID)
	item.Mode = Mode(strings.ToLower(strings.TrimSpace(string(item.Mode))))

	includes := make([]string, 0, len(item.CustomRuleIncludes))
	for _, include := range item.CustomRuleIncludes {
		include = strings.TrimSpace(include)
		if include == "" {
			continue
		}
		includes = append(includes, include)
	}
	sort.Strings(includes)
	item.CustomRuleIncludes = slices.Compact(includes)

	overrides := make([]RuleOverride, 0, len(item.RuleOverrides))
	for _, override := range item.RuleOverrides {
		override.RuleID = strings.TrimSpace(override.RuleID)
		overrides = append(overrides, override)
	}
	sort.Slice(overrides, func(i, j int) bool {
		return overrides[i].RuleID < overrides[j].RuleID
	})
	item.RuleOverrides = overrides
	return item
}

func normalizeID(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func sortPolicies(items []WAFPolicy) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
}
