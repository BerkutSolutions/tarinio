package jobs

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
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
)

type Type string

const (
	TypeCertificateIssue Type = "certificate_issue"
	TypeCertificateRenew Type = "certificate_renew"
	TypeCompile          Type = "compile"
	TypeApply            Type = "apply"
)

type Job struct {
	ID                  string `json:"id"`
	Type                Type   `json:"type"`
	TargetCertificateID string `json:"target_certificate_id"`
	TargetRevisionID    string `json:"target_revision_id"`
	Status              Status `json:"status"`
	Result              string `json:"result"`
	CreatedAt           string `json:"created_at"`
	StartedAt           string `json:"started_at,omitempty"`
	FinishedAt          string `json:"finished_at,omitempty"`
}

type state struct {
	Jobs []Job `json:"jobs"`
}

type Store struct {
	path string
	mu   sync.Mutex
}

func NewStore(root string) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("jobs store root is required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create jobs store root: %w", err)
	}
	return &Store{path: filepath.Join(root, "jobs.json")}, nil
}

func (s *Store) Create(job Job) (Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	job.ID = normalizeID(job.ID)
	job.TargetCertificateID = normalizeID(job.TargetCertificateID)
	job.TargetRevisionID = normalizeID(job.TargetRevisionID)
	if err := validateJob(job); err != nil {
		return Job{}, err
	}
	job.Status = StatusPending
	job.Result = ""
	job.CreatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	job.StartedAt = ""
	job.FinishedAt = ""

	current, err := s.loadLocked()
	if err != nil {
		return Job{}, err
	}
	for _, existing := range current.Jobs {
		if existing.ID == job.ID {
			return Job{}, fmt.Errorf("job %s already exists", job.ID)
		}
	}
	current.Jobs = append(current.Jobs, job)
	sortJobs(current.Jobs)
	if err := s.saveLocked(current); err != nil {
		return Job{}, err
	}
	return job, nil
}

func (s *Store) MarkRunning(jobID string) (Job, error) {
	return s.update(jobID, func(job *Job) {
		job.Status = StatusRunning
		job.StartedAt = time.Now().UTC().Format(time.RFC3339Nano)
		job.Result = ""
	})
}

func (s *Store) MarkSucceeded(jobID string, result string) (Job, error) {
	return s.update(jobID, func(job *Job) {
		job.Status = StatusSucceeded
		job.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
		job.Result = strings.TrimSpace(result)
	})
}

func (s *Store) MarkFailed(jobID string, result string) (Job, error) {
	return s.update(jobID, func(job *Job) {
		job.Status = StatusFailed
		job.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
		job.Result = strings.TrimSpace(result)
	})
}

func (s *Store) Get(jobID string) (Job, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobID = normalizeID(jobID)
	current, err := s.loadLocked()
	if err != nil {
		return Job{}, false, err
	}
	for _, job := range current.Jobs {
		if job.ID == jobID {
			return job, true, nil
		}
	}
	return Job{}, false, nil
}

func (s *Store) List() ([]Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	items := append([]Job(nil), current.Jobs...)
	sortJobs(items)
	return items, nil
}

func (s *Store) DeleteByRevision(revisionID string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	revisionID = normalizeID(revisionID)
	if revisionID == "" {
		return 0, errors.New("revision id is required")
	}

	current, err := s.loadLocked()
	if err != nil {
		return 0, err
	}
	filtered := make([]Job, 0, len(current.Jobs))
	deleted := 0
	for _, job := range current.Jobs {
		if job.TargetRevisionID == revisionID {
			deleted++
			continue
		}
		filtered = append(filtered, job)
	}
	current.Jobs = filtered
	sortJobs(current.Jobs)
	if err := s.saveLocked(current); err != nil {
		return 0, err
	}
	return deleted, nil
}

func (s *Store) DeleteByTypes(types []Type) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(types) == 0 {
		return 0, nil
	}
	allowed := make(map[Type]struct{}, len(types))
	for _, item := range types {
		allowed[item] = struct{}{}
	}

	current, err := s.loadLocked()
	if err != nil {
		return 0, err
	}
	filtered := make([]Job, 0, len(current.Jobs))
	deleted := 0
	for _, job := range current.Jobs {
		if _, ok := allowed[job.Type]; ok {
			deleted++
			continue
		}
		filtered = append(filtered, job)
	}
	current.Jobs = filtered
	sortJobs(current.Jobs)
	if err := s.saveLocked(current); err != nil {
		return 0, err
	}
	return deleted, nil
}

func (s *Store) update(jobID string, mutate func(*Job)) (Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobID = normalizeID(jobID)
	current, err := s.loadLocked()
	if err != nil {
		return Job{}, err
	}
	for i := range current.Jobs {
		if current.Jobs[i].ID != jobID {
			continue
		}
		mutate(&current.Jobs[i])
		sortJobs(current.Jobs)
		if err := s.saveLocked(current); err != nil {
			return Job{}, err
		}
		return current.Jobs[i], nil
	}
	return Job{}, fmt.Errorf("job %s not found", jobID)
}

func (s *Store) loadLocked() (*state, error) {
	content, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &state{}, nil
		}
		return nil, fmt.Errorf("read jobs store: %w", err)
	}

	var current state
	if err := json.Unmarshal(content, &current); err != nil {
		return nil, fmt.Errorf("decode jobs store: %w", err)
	}
	sortJobs(current.Jobs)
	return &current, nil
}

func (s *Store) saveLocked(current *state) error {
	content, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return fmt.Errorf("encode jobs store: %w", err)
	}
	content = append(content, '\n')
	if err := os.WriteFile(s.path, content, 0o644); err != nil {
		return fmt.Errorf("write jobs store: %w", err)
	}
	return nil
}

func validateJob(job Job) error {
	if job.ID == "" {
		return errors.New("job id is required")
	}
	switch job.Type {
	case TypeCertificateIssue, TypeCertificateRenew:
		if job.TargetCertificateID == "" {
			return errors.New("job target_certificate_id is required")
		}
	case TypeCompile, TypeApply:
		if job.TargetRevisionID == "" {
			return errors.New("job target_revision_id is required")
		}
	default:
		return errors.New("job type must be certificate_issue, certificate_renew, compile, or apply")
	}
	return nil
}

func normalizeID(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func sortJobs(items []Job) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt == items[j].CreatedAt {
			return items[i].ID < items[j].ID
		}
		return items[i].CreatedAt < items[j].CreatedAt
	})
}
