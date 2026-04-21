package audits

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

type Status string

const (
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
)

type AuditEvent struct {
	ID                string         `json:"id"`
	ActorUserID       string         `json:"actor_user_id,omitempty"`
	ActorIP           string         `json:"actor_ip,omitempty"`
	Action            string         `json:"action"`
	ResourceType      string         `json:"resource_type"`
	ResourceID        string         `json:"resource_id"`
	SiteID            string         `json:"site_id,omitempty"`
	RelatedRevisionID string         `json:"related_revision_id,omitempty"`
	RelatedJobID      string         `json:"related_job_id,omitempty"`
	Status            Status         `json:"status"`
	OccurredAt        string         `json:"occurred_at"`
	Summary           string         `json:"summary"`
	Details           map[string]any `json:"details_json,omitempty"`
}

type Query struct {
	Action       string
	ActorUserID  string
	ActorIP      string
	ResourceType string
	ResourceID   string
	SiteID       string
	Status       string
	From         string
	To           string
	Limit        int
	Offset       int
}

type ListResult struct {
	Items  []AuditEvent `json:"items"`
	Total  int          `json:"total"`
	Limit  int          `json:"limit"`
	Offset int          `json:"offset"`
}

type state struct {
	Events []AuditEvent `json:"audit_events"`
}

type Store struct {
	state *storage.JSONState
	mu    sync.Mutex
}

func NewStore(root string) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("audit store root is required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create audit store root: %w", err)
	}
	return &Store{state: storage.NewFileJSONState(filepath.Join(root, "audit_events.json"))}, nil
}

func NewPostgresStore(root string, backend storage.Backend) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("audit store root is required")
	}
	return &Store{
		state: storage.NewBackendJSONState(backend, "audits/audit_events.json", filepath.Join(root, "audit_events.json")),
	}, nil
}

func (s *Store) Append(event AuditEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	event = normalizeEvent(event)
	if err := validateEvent(event); err != nil {
		return err
	}
	current, err := s.loadLocked()
	if err != nil {
		return err
	}
	current.Events = append(current.Events, event)
	sortEvents(current.Events)
	return s.saveLocked(current)
}

func (s *Store) List(query Query) (ListResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return ListResult{}, err
	}
	filtered := make([]AuditEvent, 0, len(current.Events))
	from, _ := parseTime(query.From)
	to, _ := parseTime(query.To)
	for _, item := range current.Events {
		if query.Action != "" && item.Action != strings.TrimSpace(query.Action) {
			continue
		}
		if query.ActorUserID != "" && item.ActorUserID != strings.TrimSpace(query.ActorUserID) {
			continue
		}
		if query.ActorIP != "" && item.ActorIP != strings.TrimSpace(query.ActorIP) {
			continue
		}
		if query.ResourceType != "" && item.ResourceType != strings.TrimSpace(query.ResourceType) {
			continue
		}
		if query.ResourceID != "" && item.ResourceID != strings.TrimSpace(query.ResourceID) {
			continue
		}
		if query.SiteID != "" && item.SiteID != strings.TrimSpace(query.SiteID) {
			continue
		}
		if query.Status != "" && string(item.Status) != strings.TrimSpace(query.Status) {
			continue
		}
		occurredAt, err := time.Parse(time.RFC3339Nano, item.OccurredAt)
		if err != nil {
			continue
		}
		if !from.IsZero() && occurredAt.Before(from) {
			continue
		}
		if !to.IsZero() && occurredAt.After(to) {
			continue
		}
		filtered = append(filtered, item)
	}
	total := len(filtered)
	limit := query.Limit
	if limit <= 0 {
		limit = 100
	}
	offset := query.Offset
	if offset < 0 {
		offset = 0
	}
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	page := append([]AuditEvent(nil), filtered[offset:end]...)
	return ListResult{Items: page, Total: total, Limit: limit, Offset: offset}, nil
}

func (s *Store) loadLocked() (*state, error) {
	content, err := s.state.Load()
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return &state{}, nil
		}
		return nil, fmt.Errorf("read audit store: %w", err)
	}
	var current state
	if err := json.Unmarshal(content, &current); err != nil {
		return nil, fmt.Errorf("decode audit store: %w", err)
	}
	sortEvents(current.Events)
	return &current, nil
}

func (s *Store) saveLocked(current *state) error {
	content, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return fmt.Errorf("encode audit store: %w", err)
	}
	content = append(content, '\n')
	if err := s.state.Save(content); err != nil {
		return fmt.Errorf("write audit store: %w", err)
	}
	return nil
}

func normalizeEvent(event AuditEvent) AuditEvent {
	event.ID = strings.TrimSpace(event.ID)
	event.ActorUserID = strings.TrimSpace(event.ActorUserID)
	event.ActorIP = strings.TrimSpace(event.ActorIP)
	event.Action = strings.TrimSpace(event.Action)
	event.ResourceType = strings.TrimSpace(event.ResourceType)
	event.ResourceID = strings.TrimSpace(event.ResourceID)
	event.SiteID = strings.TrimSpace(event.SiteID)
	event.RelatedRevisionID = strings.TrimSpace(event.RelatedRevisionID)
	event.RelatedJobID = strings.TrimSpace(event.RelatedJobID)
	event.Summary = strings.TrimSpace(event.Summary)
	return event
}

func validateEvent(event AuditEvent) error {
	if event.ID == "" {
		return errors.New("audit event id is required")
	}
	if event.Action == "" {
		return errors.New("audit event action is required")
	}
	if event.ResourceType == "" {
		return errors.New("audit event resource_type is required")
	}
	if event.ResourceID == "" {
		return errors.New("audit event resource_id is required")
	}
	if event.Status != StatusSucceeded && event.Status != StatusFailed {
		return errors.New("audit event status is invalid")
	}
	if event.OccurredAt == "" {
		return errors.New("audit event occurred_at is required")
	}
	if event.Summary == "" {
		return errors.New("audit event summary is required")
	}
	return nil
}

func sortEvents(items []AuditEvent) {
	sort.Slice(items, func(i, j int) bool {
		left, lerr := time.Parse(time.RFC3339Nano, items[i].OccurredAt)
		right, rerr := time.Parse(time.RFC3339Nano, items[j].OccurredAt)
		if lerr != nil || rerr != nil {
			return items[i].ID > items[j].ID
		}
		if left.Equal(right) {
			return items[i].ID > items[j].ID
		}
		return left.After(right)
	})
}

func parseTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339Nano, value)
}
