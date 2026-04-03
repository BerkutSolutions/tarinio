package upstreams

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

// Upstream is the minimal Stage 1 control-plane upstream entity.
type Upstream struct {
	ID        string `json:"id"`
	SiteID    string `json:"site_id"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Scheme    string `json:"scheme"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type state struct {
	Upstreams []Upstream `json:"upstreams"`
}

// Store persists upstreams without any runtime dependency.
type Store struct {
	path string
	mu   sync.Mutex
}

func NewStore(root string) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("upstreams store root is required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create upstreams store root: %w", err)
	}
	return &Store{path: filepath.Join(root, "upstreams.json")}, nil
}

func (s *Store) Create(item Upstream) (Upstream, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item = normalizeUpstream(item)
	if err := validateUpstream(item); err != nil {
		return Upstream{}, err
	}

	current, err := s.loadLocked()
	if err != nil {
		return Upstream{}, err
	}
	for _, existing := range current.Upstreams {
		if existing.ID == item.ID {
			return Upstream{}, fmt.Errorf("upstream %s already exists", item.ID)
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	item.CreatedAt = now
	item.UpdatedAt = now
	current.Upstreams = append(current.Upstreams, item)
	sortUpstreams(current.Upstreams)
	if err := s.saveLocked(current); err != nil {
		return Upstream{}, err
	}
	return item, nil
}

func (s *Store) List() ([]Upstream, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	items := append([]Upstream(nil), current.Upstreams...)
	sortUpstreams(items)
	return items, nil
}

func (s *Store) Update(item Upstream) (Upstream, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item = normalizeUpstream(item)
	if err := validateUpstream(item); err != nil {
		return Upstream{}, err
	}

	current, err := s.loadLocked()
	if err != nil {
		return Upstream{}, err
	}
	for i := range current.Upstreams {
		if current.Upstreams[i].ID != item.ID {
			continue
		}
		item.CreatedAt = current.Upstreams[i].CreatedAt
		item.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		current.Upstreams[i] = item
		sortUpstreams(current.Upstreams)
		if err := s.saveLocked(current); err != nil {
			return Upstream{}, err
		}
		return item, nil
	}
	return Upstream{}, fmt.Errorf("upstream %s not found", item.ID)
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id = normalizeID(id)
	if id == "" {
		return errors.New("upstream id is required")
	}

	current, err := s.loadLocked()
	if err != nil {
		return err
	}
	for i := range current.Upstreams {
		if current.Upstreams[i].ID != id {
			continue
		}
		current.Upstreams = append(current.Upstreams[:i], current.Upstreams[i+1:]...)
		sortUpstreams(current.Upstreams)
		return s.saveLocked(current)
	}
	return fmt.Errorf("upstream %s not found", id)
}

func (s *Store) loadLocked() (*state, error) {
	content, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &state{}, nil
		}
		return nil, fmt.Errorf("read upstreams store: %w", err)
	}

	var current state
	if err := json.Unmarshal(content, &current); err != nil {
		return nil, fmt.Errorf("decode upstreams store: %w", err)
	}
	sortUpstreams(current.Upstreams)
	return &current, nil
}

func (s *Store) saveLocked(current *state) error {
	content, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return fmt.Errorf("encode upstreams store: %w", err)
	}
	content = append(content, '\n')
	if err := os.WriteFile(s.path, content, 0o644); err != nil {
		return fmt.Errorf("write upstreams store: %w", err)
	}
	return nil
}

func validateUpstream(item Upstream) error {
	if item.ID == "" {
		return errors.New("upstream id is required")
	}
	if item.SiteID == "" {
		return errors.New("upstream site_id is required")
	}
	if item.Host == "" {
		return errors.New("upstream host is required")
	}
	if item.Port <= 0 || item.Port > 65535 {
		return errors.New("upstream port must be between 1 and 65535")
	}
	switch item.Scheme {
	case "http", "https":
	default:
		return errors.New("upstream scheme must be http or https")
	}
	return nil
}

func normalizeUpstream(item Upstream) Upstream {
	item.ID = normalizeID(item.ID)
	item.SiteID = normalizeID(item.SiteID)
	item.Host = strings.TrimSpace(item.Host)
	item.Scheme = strings.ToLower(strings.TrimSpace(item.Scheme))
	return item
}

func normalizeID(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func sortUpstreams(items []Upstream) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
}
