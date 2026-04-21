package sites

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

	"waf/control-plane/internal/storage"
)

// Site is the minimal Stage 1 control-plane site entity used by the first CRUD API.
type Site struct {
	ID          string `json:"id"`
	PrimaryHost string `json:"primary_host"`
	Enabled     bool   `json:"enabled"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type state struct {
	Sites []Site `json:"sites"`
}

// Store persists sites as control-plane state without any runtime dependency.
type Store struct {
	state *storage.JSONState
	mu    sync.Mutex
}

func NewStore(root string) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("sites store root is required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create sites store root: %w", err)
	}
	return &Store{
		state: storage.NewFileJSONState(filepath.Join(root, "sites.json")),
	}, nil
}

func NewPostgresStore(root string, backend storage.Backend) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("sites store root is required")
	}
	return &Store{
		state: storage.NewBackendJSONState(backend, "sites/sites.json", filepath.Join(root, "sites.json")),
	}, nil
}

func (s *Store) Create(site Site) (Site, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	site = normalizeSite(site)
	if err := validateSite(site, false); err != nil {
		return Site{}, err
	}

	current, err := s.loadLocked()
	if err != nil {
		return Site{}, err
	}
	for _, existing := range current.Sites {
		if existing.ID == site.ID {
			return Site{}, fmt.Errorf("site %s already exists", site.ID)
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	site.CreatedAt = now
	site.UpdatedAt = now
	current.Sites = append(current.Sites, site)
	sortSites(current.Sites)
	if err := s.saveLocked(current); err != nil {
		return Site{}, err
	}
	return site, nil
}

func (s *Store) List() ([]Site, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	items := append([]Site(nil), current.Sites...)
	sortSites(items)
	return items, nil
}

func (s *Store) Update(site Site) (Site, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	site = normalizeSite(site)
	if err := validateSite(site, true); err != nil {
		return Site{}, err
	}

	current, err := s.loadLocked()
	if err != nil {
		return Site{}, err
	}
	for i := range current.Sites {
		if current.Sites[i].ID != site.ID {
			continue
		}
		site.CreatedAt = current.Sites[i].CreatedAt
		site.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		current.Sites[i] = site
		sortSites(current.Sites)
		if err := s.saveLocked(current); err != nil {
			return Site{}, err
		}
		return site, nil
	}
	return Site{}, fmt.Errorf("site %s not found", site.ID)
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id = normalizeID(id)
	if id == "" {
		return errors.New("site id is required")
	}

	current, err := s.loadLocked()
	if err != nil {
		return err
	}
	for i := range current.Sites {
		if current.Sites[i].ID != id {
			continue
		}
		current.Sites = append(current.Sites[:i], current.Sites[i+1:]...)
		sortSites(current.Sites)
		return s.saveLocked(current)
	}
	return fmt.Errorf("site %s not found", id)
}

func (s *Store) loadLocked() (*state, error) {
	content, err := s.state.Load()
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return &state{}, nil
		}
		return nil, fmt.Errorf("read sites store: %w", err)
	}

	var current state
	if err := json.Unmarshal(content, &current); err != nil {
		return nil, fmt.Errorf("decode sites store: %w", err)
	}
	sortSites(current.Sites)
	return &current, nil
}

func (s *Store) saveLocked(current *state) error {
	content, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return fmt.Errorf("encode sites store: %w", err)
	}
	content = append(content, '\n')
	if err := s.state.Save(content); err != nil {
		return fmt.Errorf("write sites store: %w", err)
	}
	return nil
}

func validateSite(site Site, requireExisting bool) error {
	if site.ID == "" {
		return errors.New("site id is required")
	}
	if strings.TrimSpace(site.PrimaryHost) == "" {
		return errors.New("site primary_host is required")
	}
	return nil
}

func normalizeSite(site Site) Site {
	site.ID = normalizeID(site.ID)
	site.PrimaryHost = strings.TrimSpace(site.PrimaryHost)
	return site
}

func normalizeID(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func sortSites(items []Site) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
}
