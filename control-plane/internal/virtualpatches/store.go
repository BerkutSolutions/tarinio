package virtualpatches

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
	Patches []VirtualPatch `json:"virtual_patches"`
}

// Store persists virtual patches as control-plane state.
type Store struct {
	state *storage.JSONState
	mu    sync.Mutex
}

func NewStore(root string) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("virtual patches store root is required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create virtual patches store root: %w", err)
	}
	return &Store{
		state: storage.NewFileJSONState(filepath.Join(root, "virtual_patches.json")),
	}, nil
}

func NewPostgresStore(root string, backend storage.Backend) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("virtual patches store root is required")
	}
	return &Store{
		state: storage.NewBackendJSONState(backend, "virtualpatches/virtual_patches.json", filepath.Join(root, "virtual_patches.json")),
	}, nil
}

// Create adds a new virtual patch. ID must be set by the caller.
func (s *Store) Create(patch VirtualPatch) (VirtualPatch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	patch = Normalize(patch)
	if err := Validate(patch); err != nil {
		return VirtualPatch{}, err
	}
	if strings.TrimSpace(patch.ID) == "" {
		return VirtualPatch{}, errors.New("virtual patch id is required")
	}

	current, err := s.loadLocked()
	if err != nil {
		return VirtualPatch{}, err
	}
	for _, existing := range current.Patches {
		if existing.ID == patch.ID {
			return VirtualPatch{}, fmt.Errorf("virtual patch %s already exists", patch.ID)
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	patch.CreatedAt = now
	current.Patches = append(current.Patches, patch)
	sortPatches(current.Patches)
	if err := s.saveLocked(current); err != nil {
		return VirtualPatch{}, err
	}
	return patch, nil
}

// List returns all patches for the given site (or all patches if siteID is empty).
func (s *Store) List(siteID string) ([]VirtualPatch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	out := make([]VirtualPatch, 0, len(current.Patches))
	for _, p := range current.Patches {
		if siteID != "" && p.SiteID != siteID {
			continue
		}
		out = append(out, p)
	}
	sortPatches(out)
	return out, nil
}

// ListActive returns non-expired patches for the given site.
func (s *Store) ListActive(siteID string) ([]VirtualPatch, error) {
	all, err := s.List(siteID)
	if err != nil {
		return nil, err
	}
	out := make([]VirtualPatch, 0, len(all))
	for _, p := range all {
		if !p.IsExpired() {
			out = append(out, p)
		}
	}
	return out, nil
}

// Delete removes a patch by ID.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("virtual patch id is required")
	}

	current, err := s.loadLocked()
	if err != nil {
		return err
	}
	for i := range current.Patches {
		if current.Patches[i].ID != id {
			continue
		}
		current.Patches = append(current.Patches[:i], current.Patches[i+1:]...)
		sortPatches(current.Patches)
		return s.saveLocked(current)
	}
	return fmt.Errorf("virtual patch %s not found", id)
}

func (s *Store) loadLocked() (*state, error) {
	content, err := s.state.Load()
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return &state{}, nil
		}
		return nil, fmt.Errorf("read virtual patches store: %w", err)
	}
	var current state
	if err := json.Unmarshal(content, &current); err != nil {
		return nil, fmt.Errorf("decode virtual patches store: %w", err)
	}
	sortPatches(current.Patches)
	return &current, nil
}

func (s *Store) saveLocked(current *state) error {
	content, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return fmt.Errorf("encode virtual patches store: %w", err)
	}
	content = append(content, '\n')
	if err := s.state.Save(content); err != nil {
		return fmt.Errorf("write virtual patches store: %w", err)
	}
	return nil
}

func sortPatches(items []VirtualPatch) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
}
