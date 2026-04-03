package antiddos

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type state struct {
	Settings Settings `json:"anti_ddos_settings"`
}

type Store struct {
	path string
	mu   sync.Mutex
}

func NewStore(root string) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("anti-ddos store root is required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create anti-ddos store root: %w", err)
	}
	return &Store{path: filepath.Join(root, "anti_ddos_settings.json")}, nil
}

func (s *Store) Get() (Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return Settings{}, err
	}
	item := NormalizeSettings(current.Settings)
	if err := ValidateSettings(item); err != nil {
		return Settings{}, err
	}
	return item, nil
}

func (s *Store) Upsert(item Settings) (Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item = NormalizeSettings(item)
	if err := ValidateSettings(item); err != nil {
		return Settings{}, err
	}

	current, err := s.loadLocked()
	if err != nil {
		return Settings{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	item.CreatedAt = strings.TrimSpace(current.Settings.CreatedAt)
	if item.CreatedAt == "" {
		item.CreatedAt = now
	}
	item.UpdatedAt = now
	current.Settings = item
	if err := s.saveLocked(current); err != nil {
		return Settings{}, err
	}
	return item, nil
}

func (s *Store) loadLocked() (*state, error) {
	content, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &state{Settings: DefaultSettings()}, nil
		}
		return nil, fmt.Errorf("read anti-ddos store: %w", err)
	}
	var current state
	if err := json.Unmarshal(content, &current); err != nil {
		return nil, fmt.Errorf("decode anti-ddos store: %w", err)
	}
	current.Settings = NormalizeSettings(current.Settings)
	return &current, nil
}

func (s *Store) saveLocked(current *state) error {
	content, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return fmt.Errorf("encode anti-ddos store: %w", err)
	}
	content = append(content, '\n')
	if err := os.WriteFile(s.path, content, 0o644); err != nil {
		return fmt.Errorf("write anti-ddos store: %w", err)
	}
	return nil
}
