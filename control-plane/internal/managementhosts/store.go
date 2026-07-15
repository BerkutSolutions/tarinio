package managementhosts

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"waf/control-plane/internal/storage"
)

// Settings is the persisted, explicit ownership of WAF self-management paths.
// Version provides optimistic concurrency for settings clients.
type Settings struct {
	Hosts               []string `json:"management_hosts"`
	BlockDirectIPAccess bool     `json:"block_direct_ip_access"`
	Version             int64    `json:"version"`
	Migrated            bool     `json:"migrated"`
	SetupRequired       bool     `json:"management_hosts_setup_required"`
	UpdatedAt           string   `json:"updated_at"`
}

type Store struct {
	state *storage.JSONState
	mu    sync.Mutex
}

// ErrVersionConflict lets HTTP/API callers distinguish a stale optimistic-lock
// write from an invalid management-host configuration.
var ErrVersionConflict = errors.New("management hosts version conflict")

func NewStore(root string, backend storage.Backend) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("management hosts store root is required")
	}
	path := filepath.Join(root, "management_hosts.json")
	if storage.IsNilBackend(backend) {
		if err := os.MkdirAll(root, 0o755); err != nil {
			return nil, err
		}
		return &Store{state: storage.NewFileJSONState(path)}, nil
	}
	return &Store{state: storage.NewBackendJSONState(backend, "settings/management_hosts.json", path)}, nil
}

func (s *Store) Get() (Settings, error) { s.mu.Lock(); defer s.mu.Unlock(); return s.loadLocked() }

func (s *Store) Update(hosts []string, version int64) (Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, err := s.loadLocked()
	if err != nil {
		return Settings{}, err
	}
	if version != current.Version {
		return Settings{}, ErrVersionConflict
	}
	canonical, err := NormalizeHosts(hosts)
	if err != nil {
		return Settings{}, err
	}
	if len(canonical) == 0 {
		return Settings{}, errors.New("at least one management host is required")
	}
	current.Hosts, current.Version, current.Migrated, current.SetupRequired, current.UpdatedAt = canonical, current.Version+1, true, false, time.Now().UTC().Format(time.RFC3339Nano)
	return current, s.saveLocked(current)
}

// UpdateDirectIPAccess changes the WAF-wide default-server policy without
// touching the independently managed control-plane host ownership.
func (s *Store) UpdateDirectIPAccess(block bool) (Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, err := s.loadLocked()
	if err != nil {
		return Settings{}, err
	}
	if current.BlockDirectIPAccess == block {
		return current, nil
	}
	current.BlockDirectIPAccess = block
	current.Version++
	current.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	return current, s.saveLocked(current)
}

// Bootstrap migrates only an unambiguous legacy public host. It is idempotent.
func (s *Store) Bootstrap(host string) (Settings, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, err := s.loadLocked()
	if err != nil {
		return Settings{}, false, err
	}
	if current.Migrated || len(current.Hosts) != 0 {
		return current, false, nil
	}
	hosts, err := NormalizeHosts([]string{host})
	if err != nil || len(hosts) == 0 {
		return current, false, nil
	}
	current.Hosts, current.Version, current.Migrated, current.SetupRequired, current.UpdatedAt = hosts, 1, true, false, time.Now().UTC().Format(time.RFC3339Nano)
	return current, true, s.saveLocked(current)
}

func (s *Store) loadLocked() (Settings, error) {
	content, err := s.state.Load()
	if errors.Is(err, storage.ErrNotFound) {
		return Settings{SetupRequired: true}, nil
	}
	if err != nil {
		return Settings{}, err
	}
	var out Settings
	if err := json.Unmarshal(content, &out); err != nil {
		return Settings{}, err
	}
	hosts, err := NormalizeHosts(out.Hosts)
	if err != nil {
		return Settings{}, err
	}
	out.Hosts = hosts
	out.SetupRequired = !out.Migrated || len(out.Hosts) == 0
	return out, nil
}
func (s *Store) saveLocked(item Settings) error {
	content, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return err
	}
	return s.state.Save(append(content, '\n'))
}

func NormalizeHosts(hosts []string) ([]string, error) {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(hosts))
	for _, raw := range hosts {
		host := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(raw), "."))
		if host == "" {
			continue
		}
		if net.ParseIP(host) == nil {
			// localhost is a supported legacy/dev bootstrap host. Keeping it
			// valid makes an upgrade deterministic instead of silently dropping
			// an existing local management endpoint.
			if host == "localhost" {
				goto normalized
			}
			if len(host) > 253 || strings.ContainsAny(host, "/:@") || !strings.Contains(host, ".") {
				return nil, fmt.Errorf("invalid management host %q", raw)
			}
		}
	normalized:
		if _, ok := seen[host]; !ok {
			seen[host] = struct{}{}
			out = append(out, host)
		}
	}
	sort.Strings(out)
	return out, nil
}
