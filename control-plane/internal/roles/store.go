package roles

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

	"waf/control-plane/internal/rbac"
)

type Role struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Permissions []rbac.Permission `json:"permissions"`
	CreatedAt   string            `json:"created_at"`
	UpdatedAt   string            `json:"updated_at"`
}

type state struct {
	Roles []Role `json:"roles"`
}

type Store struct {
	path string
	mu   sync.Mutex
}

func NewStore(root string) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("roles store root is required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create roles store root: %w", err)
	}
	store := &Store{path: filepath.Join(root, "roles.json")}
	if err := store.seedDefaults(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) Get(id string) (Role, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return Role{}, false, err
	}
	id = rbac.NormalizeRoleID(id)
	for _, item := range current.Roles {
		if item.ID == id {
			return item, true, nil
		}
	}
	return Role{}, false, nil
}

func (s *Store) PermissionsForRoleIDs(roleIDs []string) []rbac.Permission {
	current, err := s.List()
	if err != nil {
		return nil
	}
	byID := make(map[string]Role, len(current))
	for _, item := range current {
		byID[item.ID] = item
	}
	set := map[rbac.Permission]struct{}{}
	for _, roleID := range roleIDs {
		role, ok := byID[rbac.NormalizeRoleID(roleID)]
		if !ok {
			continue
		}
		for _, permission := range role.Permissions {
			set[permission] = struct{}{}
		}
	}
	out := make([]rbac.Permission, 0, len(set))
	for permission := range set {
		out = append(out, permission)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func (s *Store) List() ([]Role, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	items := append([]Role(nil), current.Roles...)
	sortRoles(items)
	return items, nil
}

func (s *Store) seedDefaults() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return err
	}
	if len(current.Roles) > 0 {
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	defaults := rbac.DefaultRolePermissions()
	for _, roleID := range []string{"admin", "operator", "viewer"} {
		current.Roles = append(current.Roles, Role{
			ID:          roleID,
			Name:        roleID,
			Permissions: rbac.SortedPermissions(defaults[roleID]),
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}
	sortRoles(current.Roles)
	return s.saveLocked(current)
}

func (s *Store) loadLocked() (*state, error) {
	content, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &state{}, nil
		}
		return nil, fmt.Errorf("read roles store: %w", err)
	}
	var current state
	if err := json.Unmarshal(content, &current); err != nil {
		return nil, fmt.Errorf("decode roles store: %w", err)
	}
	sortRoles(current.Roles)
	return &current, nil
}

func (s *Store) saveLocked(current *state) error {
	content, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return fmt.Errorf("encode roles store: %w", err)
	}
	content = append(content, '\n')
	if err := os.WriteFile(s.path, content, 0o644); err != nil {
		return fmt.Errorf("write roles store: %w", err)
	}
	return nil
}

func sortRoles(items []Role) {
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
}
