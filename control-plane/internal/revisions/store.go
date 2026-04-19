package revisions

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type Status string

const (
	StatusPending  Status = "pending"
	StatusInactive Status = "inactive"
	StatusActive   Status = "active"
	StatusFailed   Status = "failed"
)

// Revision stores the minimal control-plane metadata for one compiled bundle.
type Revision struct {
	ID              string `json:"id"`
	Version         int    `json:"version"`
	CreatedAt       string `json:"created_at"`
	Checksum        string `json:"checksum"`
	BundlePath      string `json:"bundle_path"`
	Status          Status `json:"status"`
	LastApplyJobID  string `json:"last_apply_job_id,omitempty"`
	LastApplyStatus string `json:"last_apply_status,omitempty"`
	LastApplyResult string `json:"last_apply_result,omitempty"`
	LastApplyAt     string `json:"last_apply_at,omitempty"`
}

type state struct {
	CurrentActiveRevisionID string     `json:"current_active_revision_id,omitempty"`
	Revisions               []Revision `json:"revisions"`
}

// Store persists revision metadata and lifecycle state for the control plane.
// It is not readable by runtime and contains no domain-model entities.
type Store struct {
	path string
	mu   sync.Mutex
}

func NewStore(root string) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("store root is required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create store root: %w", err)
	}
	return &Store{
		path: filepath.Join(root, "revisions.json"),
	}, nil
}

func (s *Store) SavePending(revision Revision) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := validateRevision(revision); err != nil {
		return err
	}
	revision.Status = StatusPending

	current, err := s.loadLocked()
	if err != nil {
		return err
	}
	for _, existing := range current.Revisions {
		if existing.ID == revision.ID {
			return fmt.Errorf("revision %s already exists", revision.ID)
		}
	}
	current.Revisions = append(current.Revisions, revision)
	sortRevisions(current.Revisions)
	return s.saveLocked(current)
}

func (s *Store) MarkActive(revisionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return err
	}

	index := indexByID(current.Revisions, revisionID)
	if index == -1 {
		return fmt.Errorf("revision %s not found", revisionID)
	}

	for i := range current.Revisions {
		if current.Revisions[i].Status == StatusActive {
			current.Revisions[i].Status = StatusInactive
		}
	}
	current.Revisions[index].Status = StatusActive
	current.CurrentActiveRevisionID = revisionID
	sortRevisions(current.Revisions)
	return s.saveLocked(current)
}

func (s *Store) MarkFailed(revisionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return err
	}

	index := indexByID(current.Revisions, revisionID)
	if index == -1 {
		return fmt.Errorf("revision %s not found", revisionID)
	}

	current.Revisions[index].Status = StatusFailed
	if current.CurrentActiveRevisionID == revisionID {
		current.CurrentActiveRevisionID = ""
	}
	sortRevisions(current.Revisions)
	return s.saveLocked(current)
}

func (s *Store) Get(revisionID string) (Revision, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return Revision{}, false, err
	}
	for _, revision := range current.Revisions {
		if revision.ID == revisionID {
			return revision, true, nil
		}
	}
	return Revision{}, false, nil
}

func (s *Store) ResetStatuses() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return err
	}
	for i := range current.Revisions {
		if current.Revisions[i].Status == StatusActive {
			continue
		}
		current.Revisions[i].Status = StatusInactive
	}
	sortRevisions(current.Revisions)
	return s.saveLocked(current)
}

func (s *Store) RecordApplyResult(revisionID string, jobID string, status string, result string, appliedAt string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return err
	}

	index := indexByID(current.Revisions, revisionID)
	if index == -1 {
		return fmt.Errorf("revision %s not found", revisionID)
	}

	current.Revisions[index].LastApplyJobID = strings.TrimSpace(jobID)
	current.Revisions[index].LastApplyStatus = strings.TrimSpace(status)
	current.Revisions[index].LastApplyResult = strings.TrimSpace(result)
	current.Revisions[index].LastApplyAt = strings.TrimSpace(appliedAt)
	sortRevisions(current.Revisions)
	return s.saveLocked(current)
}

func (s *Store) List() ([]Revision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	out := append([]Revision(nil), current.Revisions...)
	sortRevisions(out)
	return out, nil
}

func (s *Store) CurrentActive() (Revision, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return Revision{}, false, err
	}
	if current.CurrentActiveRevisionID == "" {
		return Revision{}, false, nil
	}
	for _, revision := range current.Revisions {
		if revision.ID == current.CurrentActiveRevisionID {
			return revision, true, nil
		}
	}
	return Revision{}, false, nil
}

func (s *Store) Delete(revisionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	revisionID = strings.ToLower(strings.TrimSpace(revisionID))
	if revisionID == "" {
		return errors.New("revision id is required")
	}

	current, err := s.loadLocked()
	if err != nil {
		return err
	}
	if current.CurrentActiveRevisionID == revisionID {
		return fmt.Errorf("revision %s is active and cannot be deleted", revisionID)
	}
	for i := range current.Revisions {
		if current.Revisions[i].ID != revisionID {
			continue
		}
		current.Revisions = append(current.Revisions[:i], current.Revisions[i+1:]...)
		sortRevisions(current.Revisions)
		return s.saveLocked(current)
	}
	return fmt.Errorf("revision %s not found", revisionID)
}

func (s *Store) loadLocked() (*state, error) {
	content, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &state{}, nil
		}
		return nil, fmt.Errorf("read store: %w", err)
	}

	var current state
	if err := json.Unmarshal(content, &current); err != nil {
		return nil, fmt.Errorf("decode store: %w", err)
	}
	sortRevisions(current.Revisions)
	return &current, nil
}

func (s *Store) saveLocked(current *state) error {
	content, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return fmt.Errorf("encode store: %w", err)
	}
	content = append(content, '\n')
	if err := os.WriteFile(s.path, content, 0o644); err != nil {
		return fmt.Errorf("write store: %w", err)
	}
	return nil
}

func validateRevision(revision Revision) error {
	if strings.TrimSpace(revision.ID) == "" {
		return errors.New("revision id is required")
	}
	if revision.Version <= 0 {
		return errors.New("revision version must be positive")
	}
	if strings.TrimSpace(revision.CreatedAt) == "" {
		return errors.New("revision created_at is required")
	}
	if strings.TrimSpace(revision.Checksum) == "" {
		return errors.New("revision checksum is required")
	}
	if strings.TrimSpace(revision.BundlePath) == "" {
		return errors.New("revision bundle path is required")
	}
	return nil
}

func sortRevisions(revisions []Revision) {
	sort.Slice(revisions, func(i, j int) bool {
		if revisions[i].Version == revisions[j].Version {
			return revisions[i].ID < revisions[j].ID
		}
		return revisions[i].Version < revisions[j].Version
	})
}

func indexByID(revisions []Revision, revisionID string) int {
	for i, revision := range revisions {
		if revision.ID == revisionID {
			return i
		}
	}
	return -1
}
