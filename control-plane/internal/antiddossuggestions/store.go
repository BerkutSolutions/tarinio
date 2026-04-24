package antiddossuggestions

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
	Items []Suggestion `json:"anti_ddos_rule_suggestions"`
}

type Store struct {
	state *storage.JSONState
	mu    sync.Mutex
}

func NewStore(root string) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("anti-ddos rule suggestions store root is required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create anti-ddos rule suggestions store root: %w", err)
	}
	return &Store{state: storage.NewFileJSONState(filepath.Join(root, "anti_ddos_rule_suggestions.json"))}, nil
}

func NewPostgresStore(root string, backend storage.Backend) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("anti-ddos rule suggestions store root is required")
	}
	return &Store{
		state: storage.NewBackendJSONState(backend, "antiddos/anti_ddos_rule_suggestions.json", filepath.Join(root, "anti_ddos_rule_suggestions.json")),
	}, nil
}

func (s *Store) List() ([]Suggestion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	items := append([]Suggestion(nil), current.Items...)
	sortSuggestions(items)
	return items, nil
}

func (s *Store) Upsert(item Suggestion) (Suggestion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item = NormalizeSuggestion(item)
	if err := ValidateSuggestion(item); err != nil {
		return Suggestion{}, err
	}

	current, err := s.loadLocked()
	if err != nil {
		return Suggestion{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for i := range current.Items {
		if current.Items[i].ID != item.ID {
			continue
		}
		item.CreatedAt = current.Items[i].CreatedAt
		if item.CreatedAt == "" {
			item.CreatedAt = now
		}
		item.UpdatedAt = now
		current.Items[i] = item
		sortSuggestions(current.Items)
		if err := s.saveLocked(current); err != nil {
			return Suggestion{}, err
		}
		return item, nil
	}
	item.CreatedAt = now
	item.UpdatedAt = now
	current.Items = append(current.Items, item)
	sortSuggestions(current.Items)
	if err := s.saveLocked(current); err != nil {
		return Suggestion{}, err
	}
	return item, nil
}

func (s *Store) SetStatus(id, status string) (Suggestion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id = strings.TrimSpace(strings.ToLower(id))
	status = strings.TrimSpace(strings.ToLower(status))
	if id == "" {
		return Suggestion{}, errors.New("anti-ddos rule suggestion id is required")
	}
	if status != StatusSuggested && status != StatusShadow {
		return Suggestion{}, errors.New("anti-ddos rule suggestion status must be suggested or shadow")
	}

	current, err := s.loadLocked()
	if err != nil {
		return Suggestion{}, err
	}
	for i := range current.Items {
		if current.Items[i].ID != id {
			continue
		}
		current.Items[i].Status = status
		current.Items[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		sortSuggestions(current.Items)
		if err := s.saveLocked(current); err != nil {
			return Suggestion{}, err
		}
		return current.Items[i], nil
	}
	return Suggestion{}, fmt.Errorf("anti-ddos rule suggestion %s not found", id)
}

func (s *Store) loadLocked() (*state, error) {
	content, err := s.state.Load()
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return &state{}, nil
		}
		return nil, fmt.Errorf("read anti-ddos rule suggestions store: %w", err)
	}
	var current state
	if err := json.Unmarshal(content, &current); err != nil {
		return nil, fmt.Errorf("decode anti-ddos rule suggestions store: %w", err)
	}
	for i := range current.Items {
		current.Items[i] = NormalizeSuggestion(current.Items[i])
	}
	sortSuggestions(current.Items)
	return &current, nil
}

func (s *Store) saveLocked(current *state) error {
	content, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return fmt.Errorf("encode anti-ddos rule suggestions store: %w", err)
	}
	content = append(content, '\n')
	if err := s.state.Save(content); err != nil {
		return fmt.Errorf("write anti-ddos rule suggestions store: %w", err)
	}
	return nil
}

func sortSuggestions(items []Suggestion) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].UpdatedAt == items[j].UpdatedAt {
			return items[i].ID < items[j].ID
		}
		return items[i].UpdatedAt > items[j].UpdatedAt
	})
}
