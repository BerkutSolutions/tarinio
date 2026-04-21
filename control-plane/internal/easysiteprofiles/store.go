package easysiteprofiles

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

type state struct {
	Profiles []EasySiteProfile `json:"easy_site_profiles"`
}

// Store persists Easy site profiles as control-plane state.
type Store struct {
	state *storage.JSONState
	mu    sync.Mutex
}

func NewStore(root string) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("easy site profiles store root is required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create easy site profiles store root: %w", err)
	}
	return &Store{state: storage.NewFileJSONState(filepath.Join(root, "easy_site_profiles.json"))}, nil
}

func NewPostgresStore(root string, backend storage.Backend) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("easy site profiles store root is required")
	}
	return &Store{
		state: storage.NewBackendJSONState(backend, "easysiteprofiles/easy_site_profiles.json", filepath.Join(root, "easy_site_profiles.json")),
	}, nil
}

func (s *Store) Create(profile EasySiteProfile) (EasySiteProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile = normalizeProfile(profile)
	if err := validateProfile(profile); err != nil {
		return EasySiteProfile{}, err
	}

	current, err := s.loadLocked()
	if err != nil {
		return EasySiteProfile{}, err
	}
	for _, existing := range current.Profiles {
		if existing.SiteID == profile.SiteID {
			return EasySiteProfile{}, fmt.Errorf("easy site profile for site %s already exists", profile.SiteID)
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	profile.CreatedAt = now
	profile.UpdatedAt = now
	current.Profiles = append(current.Profiles, profile)
	sortProfiles(current.Profiles)
	if err := s.saveLocked(current); err != nil {
		return EasySiteProfile{}, err
	}
	return profile, nil
}

func (s *Store) List() ([]EasySiteProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	items := append([]EasySiteProfile(nil), current.Profiles...)
	for i := range items {
		items[i] = normalizeProfile(items[i])
	}
	sortProfiles(items)
	return items, nil
}

func (s *Store) Update(profile EasySiteProfile) (EasySiteProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile = normalizeProfile(profile)
	if err := validateProfile(profile); err != nil {
		return EasySiteProfile{}, err
	}

	current, err := s.loadLocked()
	if err != nil {
		return EasySiteProfile{}, err
	}
	for i := range current.Profiles {
		if current.Profiles[i].SiteID != profile.SiteID {
			continue
		}
		profile.CreatedAt = current.Profiles[i].CreatedAt
		profile.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		current.Profiles[i] = profile
		sortProfiles(current.Profiles)
		if err := s.saveLocked(current); err != nil {
			return EasySiteProfile{}, err
		}
		return profile, nil
	}
	return EasySiteProfile{}, fmt.Errorf("easy site profile for site %s not found", profile.SiteID)
}

func (s *Store) Delete(siteID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	siteID = normalizeID(siteID)
	if siteID == "" {
		return errors.New("easy site profile site_id is required")
	}

	current, err := s.loadLocked()
	if err != nil {
		return err
	}
	for i := range current.Profiles {
		if current.Profiles[i].SiteID != siteID {
			continue
		}
		current.Profiles = append(current.Profiles[:i], current.Profiles[i+1:]...)
		sortProfiles(current.Profiles)
		return s.saveLocked(current)
	}
	return fmt.Errorf("easy site profile for site %s not found", siteID)
}

func (s *Store) Get(siteID string) (EasySiteProfile, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	siteID = normalizeID(siteID)
	if siteID == "" {
		return EasySiteProfile{}, false, errors.New("easy site profile site_id is required")
	}
	current, err := s.loadLocked()
	if err != nil {
		return EasySiteProfile{}, false, err
	}
	for _, item := range current.Profiles {
		if item.SiteID == siteID {
			return normalizeProfile(item), true, nil
		}
	}
	return EasySiteProfile{}, false, nil
}

func (s *Store) loadLocked() (*state, error) {
	content, err := s.state.Load()
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return &state{}, nil
		}
		return nil, fmt.Errorf("read easy site profiles store: %w", err)
	}
	var current state
	if err := json.Unmarshal(content, &current); err != nil {
		return nil, fmt.Errorf("decode easy site profiles store: %w", err)
	}
	sortProfiles(current.Profiles)
	return &current, nil
}

func (s *Store) saveLocked(current *state) error {
	content, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return fmt.Errorf("encode easy site profiles store: %w", err)
	}
	content = append(content, '\n')
	if err := s.state.Save(content); err != nil {
		return fmt.Errorf("write easy site profiles store: %w", err)
	}
	return nil
}

func sortProfiles(items []EasySiteProfile) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].SiteID < items[j].SiteID
	})
}
