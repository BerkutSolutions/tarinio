package certificates

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

// Certificate stores control-plane certificate metadata only.
type Certificate struct {
	ID         string   `json:"id"`
	CommonName string   `json:"common_name"`
	SANList    []string `json:"san_list"`
	NotBefore  string   `json:"not_before"`
	NotAfter   string   `json:"not_after"`
	Status     string   `json:"status"`
	CreatedAt  string   `json:"created_at"`
	UpdatedAt  string   `json:"updated_at"`
}

type state struct {
	Certificates []Certificate `json:"certificates"`
}

// Store persists certificate metadata without private keys or runtime coupling.
type Store struct {
	path string
	mu   sync.Mutex
}

func NewStore(root string) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("certificates store root is required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create certificates store root: %w", err)
	}
	return &Store{path: filepath.Join(root, "certificates.json")}, nil
}

func (s *Store) Create(item Certificate) (Certificate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item = normalizeCertificate(item)
	if err := validateCertificate(item); err != nil {
		return Certificate{}, err
	}

	current, err := s.loadLocked()
	if err != nil {
		return Certificate{}, err
	}
	for _, existing := range current.Certificates {
		if existing.ID == item.ID {
			return Certificate{}, fmt.Errorf("certificate %s already exists", item.ID)
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	item.CreatedAt = now
	item.UpdatedAt = now
	current.Certificates = append(current.Certificates, item)
	sortCertificates(current.Certificates)
	if err := s.saveLocked(current); err != nil {
		return Certificate{}, err
	}
	return item, nil
}

func (s *Store) List() ([]Certificate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	items := append([]Certificate(nil), current.Certificates...)
	sortCertificates(items)
	return items, nil
}

func (s *Store) Update(item Certificate) (Certificate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item = normalizeCertificate(item)
	if err := validateCertificate(item); err != nil {
		return Certificate{}, err
	}

	current, err := s.loadLocked()
	if err != nil {
		return Certificate{}, err
	}
	for i := range current.Certificates {
		if current.Certificates[i].ID != item.ID {
			continue
		}
		item.CreatedAt = current.Certificates[i].CreatedAt
		item.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		current.Certificates[i] = item
		sortCertificates(current.Certificates)
		if err := s.saveLocked(current); err != nil {
			return Certificate{}, err
		}
		return item, nil
	}
	return Certificate{}, fmt.Errorf("certificate %s not found", item.ID)
}

func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id = normalizeID(id)
	if id == "" {
		return errors.New("certificate id is required")
	}

	current, err := s.loadLocked()
	if err != nil {
		return err
	}
	for i := range current.Certificates {
		if current.Certificates[i].ID != id {
			continue
		}
		current.Certificates = append(current.Certificates[:i], current.Certificates[i+1:]...)
		sortCertificates(current.Certificates)
		return s.saveLocked(current)
	}
	return fmt.Errorf("certificate %s not found", id)
}

func (s *Store) loadLocked() (*state, error) {
	content, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &state{}, nil
		}
		return nil, fmt.Errorf("read certificates store: %w", err)
	}

	var current state
	if err := json.Unmarshal(content, &current); err != nil {
		return nil, fmt.Errorf("decode certificates store: %w", err)
	}
	sortCertificates(current.Certificates)
	return &current, nil
}

func (s *Store) saveLocked(current *state) error {
	content, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return fmt.Errorf("encode certificates store: %w", err)
	}
	content = append(content, '\n')
	if err := os.WriteFile(s.path, content, 0o644); err != nil {
		return fmt.Errorf("write certificates store: %w", err)
	}
	return nil
}

func validateCertificate(item Certificate) error {
	if item.ID == "" {
		return errors.New("certificate id is required")
	}
	if item.CommonName == "" {
		return errors.New("certificate common_name is required")
	}
	switch item.Status {
	case "active", "expired", "revoked":
	default:
		return errors.New("certificate status must be active, expired, or revoked")
	}
	if item.NotBefore != "" {
		if _, err := time.Parse(time.RFC3339, item.NotBefore); err != nil {
			return errors.New("certificate not_before must be RFC3339")
		}
	}
	if item.NotAfter != "" {
		if _, err := time.Parse(time.RFC3339, item.NotAfter); err != nil {
			return errors.New("certificate not_after must be RFC3339")
		}
	}
	return nil
}

func normalizeCertificate(item Certificate) Certificate {
	item.ID = normalizeID(item.ID)
	item.CommonName = strings.TrimSpace(item.CommonName)
	item.Status = strings.ToLower(strings.TrimSpace(item.Status))
	item.NotBefore = strings.TrimSpace(item.NotBefore)
	item.NotAfter = strings.TrimSpace(item.NotAfter)

	sans := make([]string, 0, len(item.SANList))
	for _, san := range item.SANList {
		san = strings.TrimSpace(san)
		if san == "" {
			continue
		}
		sans = append(sans, san)
	}
	sort.Strings(sans)
	item.SANList = slices.Compact(sans)
	return item
}

func normalizeID(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func sortCertificates(items []Certificate) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
}
