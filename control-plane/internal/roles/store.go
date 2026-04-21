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
	"waf/control-plane/internal/storage"
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
	state *storage.JSONState
	mu    sync.Mutex
}

func NewStore(root string) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("roles store root is required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create roles store root: %w", err)
	}
	store := &Store{state: storage.NewFileJSONState(filepath.Join(root, "roles.json"))}
	if err := store.seedDefaults(); err != nil {
		return nil, err
	}
	return store, nil
}

func NewPostgresStore(root string, backend storage.Backend) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("roles store root is required")
	}
	store := &Store{
		state: storage.NewBackendJSONState(backend, "roles/roles.json", filepath.Join(root, "roles.json")),
	}
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
		normalizedID := rbac.NormalizeRoleID(roleID)
		role, ok := byID[normalizedID]
		if !ok {
			if normalizedID != "admin" {
				continue
			}
			for _, permission := range rbac.AllPermissions() {
				set[permission] = struct{}{}
			}
			continue
		}
		for _, permission := range role.Permissions {
			set[permission] = struct{}{}
		}
		if normalizedID == "admin" {
			for _, permission := range rbac.AllPermissions() {
				set[permission] = struct{}{}
			}
		}
	}
	out := make([]rbac.Permission, 0, len(set))
	for permission := range set {
		out = append(out, permission)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func (s *Store) Create(role Role) (Role, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return Role{}, err
	}
	role, err = normalizeRole(role)
	if err != nil {
		return Role{}, err
	}
	for _, item := range current.Roles {
		if item.ID == role.ID {
			return Role{}, fmt.Errorf("role %s already exists", role.ID)
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	role.CreatedAt = now
	role.UpdatedAt = now
	current.Roles = append(current.Roles, role)
	sortRoles(current.Roles)
	if err := s.saveLocked(current); err != nil {
		return Role{}, err
	}
	return role, nil
}

func (s *Store) Update(role Role) (Role, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return Role{}, err
	}
	role, err = normalizeRole(role)
	if err != nil {
		return Role{}, err
	}
	for i := range current.Roles {
		if current.Roles[i].ID != role.ID {
			continue
		}
		role.CreatedAt = current.Roles[i].CreatedAt
		role.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		current.Roles[i] = role
		sortRoles(current.Roles)
		if err := s.saveLocked(current); err != nil {
			return Role{}, err
		}
		return role, nil
	}
	return Role{}, fmt.Errorf("role %s not found", role.ID)
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return err
	}
	normalizedID := rbac.NormalizeRoleID(id)
	for i := range current.Roles {
		if current.Roles[i].ID != normalizedID {
			continue
		}
		current.Roles = append(current.Roles[:i], current.Roles[i+1:]...)
		return s.saveLocked(current)
	}
	return fmt.Errorf("role %s not found", normalizedID)
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
	now := time.Now().UTC().Format(time.RFC3339)
	updated := false
	for _, definition := range rbac.DefaultRoleDefinitions() {
		found := false
		for i := range current.Roles {
			if current.Roles[i].ID != definition.ID {
				continue
			}
			found = true
			mergedPermissions := mergePermissions(current.Roles[i].Permissions, definition.Permissions)
			if current.Roles[i].Name != definition.Name || !equalPermissions(current.Roles[i].Permissions, mergedPermissions) {
				current.Roles[i].Name = definition.Name
				current.Roles[i].Permissions = mergedPermissions
				current.Roles[i].UpdatedAt = now
				updated = true
			}
			break
		}
		if found {
			continue
		}
		updated = true
		current.Roles = append(current.Roles, Role{
			ID:          definition.ID,
			Name:        definition.Name,
			Permissions: rbac.SortedPermissions(definition.Permissions),
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}
	sortRoles(current.Roles)
	if !updated {
		return nil
	}
	return s.saveLocked(current)
}

func (s *Store) loadLocked() (*state, error) {
	content, err := s.state.Load()
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
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
	if err := s.state.Save(content); err != nil {
		return fmt.Errorf("write roles store: %w", err)
	}
	return nil
}

func sortRoles(items []Role) {
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
}

func normalizeRole(role Role) (Role, error) {
	role.ID = rbac.NormalizeRoleID(role.ID)
	role.Name = strings.TrimSpace(role.Name)
	if role.ID == "" {
		return Role{}, errors.New("role id is required")
	}
	if role.Name == "" {
		role.Name = role.ID
	}
	role.Permissions = rbac.SortedPermissions(role.Permissions)
	for _, permission := range role.Permissions {
		if !rbac.IsKnownPermission(permission) {
			return Role{}, fmt.Errorf("unknown permission %s", permission)
		}
	}
	if role.ID == "admin" {
		role.Permissions = rbac.SortedPermissions(rbac.AllPermissions())
	}
	return role, nil
}

func mergePermissions(current []rbac.Permission, required []rbac.Permission) []rbac.Permission {
	merged := make([]rbac.Permission, 0, len(current)+len(required))
	seen := make(map[rbac.Permission]struct{}, len(current)+len(required))
	for _, permission := range current {
		if !rbac.IsKnownPermission(permission) {
			continue
		}
		if _, ok := seen[permission]; ok {
			continue
		}
		seen[permission] = struct{}{}
		merged = append(merged, permission)
	}
	for _, permission := range required {
		if _, ok := seen[permission]; ok {
			continue
		}
		seen[permission] = struct{}{}
		merged = append(merged, permission)
	}
	return rbac.SortedPermissions(merged)
}

func equalPermissions(left []rbac.Permission, right []rbac.Permission) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
