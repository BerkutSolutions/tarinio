package events

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

type Type string
type Severity string

const (
	TypeApplyStarted      Type = "apply_started"
	TypeApplySucceeded    Type = "apply_succeeded"
	TypeApplyFailed       Type = "apply_failed"
	TypeReloadFailed      Type = "reload_failed"
	TypeHealthCheckFailed Type = "health_check_failed"
	TypeRollbackPerformed Type = "rollback_performed"
	TypeSecurityWAF       Type = "security_waf"
	TypeSecurityRateLimit Type = "security_rate_limit"
	TypeSecurityAccess    Type = "security_access"

	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

// Event is the bounded normalized operational/security event model for Stage 1.
type Event struct {
	ID                   string         `json:"id"`
	Type                 Type           `json:"type"`
	Severity             Severity       `json:"severity"`
	SiteID               string         `json:"site_id,omitempty"`
	SourceComponent      string         `json:"source_component"`
	OccurredAt           string         `json:"occurred_at"`
	Summary              string         `json:"summary"`
	Details              map[string]any `json:"details,omitempty"`
	RelatedRevisionID    string         `json:"related_revision_id,omitempty"`
	RelatedJobID         string         `json:"related_job_id,omitempty"`
	RelatedCertificateID string         `json:"related_certificate_id,omitempty"`
	RelatedRuleID        string         `json:"related_rule_id,omitempty"`
}

type state struct {
	Events []Event `json:"events"`
}

type RetentionPolicy struct {
	MaxEvents int
	MaxAge    time.Duration
}

type Store struct {
	path string
	mu   sync.Mutex
}

func NewStore(root string) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("events store root is required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create events store root: %w", err)
	}
	return &Store{path: filepath.Join(root, "events.json")}, nil
}

func (s *Store) Create(event Event) (Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	event = normalizeEvent(event)
	if err := validateEvent(event); err != nil {
		return Event{}, err
	}

	current, err := s.loadLocked()
	if err != nil {
		return Event{}, err
	}
	for _, existing := range current.Events {
		if existing.ID == event.ID {
			return Event{}, fmt.Errorf("event %s already exists", event.ID)
		}
	}
	current.Events = append(current.Events, event)
	sortEvents(current.Events)
	if err := s.saveLocked(current); err != nil {
		return Event{}, err
	}
	return event, nil
}

func (s *Store) List() ([]Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	items := append([]Event(nil), current.Events...)
	sortEvents(items)
	return items, nil
}

func (s *Store) Prune(policy RetentionPolicy) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, err := s.loadLocked()
	if err != nil {
		return 0, err
	}

	originalCount := len(current.Events)
	items := append([]Event(nil), current.Events...)
	if policy.MaxAge > 0 {
		cutoff := time.Now().UTC().Add(-policy.MaxAge)
		filtered := make([]Event, 0, len(items))
		for _, item := range items {
			occurredAt, err := parseOccurredAt(item.OccurredAt)
			if err != nil {
				continue
			}
			if occurredAt.Before(cutoff) {
				continue
			}
			filtered = append(filtered, item)
		}
		items = filtered
	}

	sortEvents(items)
	if policy.MaxEvents > 0 && len(items) > policy.MaxEvents {
		items = append([]Event(nil), items[len(items)-policy.MaxEvents:]...)
	}

	current.Events = items
	if err := s.saveLocked(current); err != nil {
		return 0, err
	}
	return originalCount - len(items), nil
}

func (s *Store) loadLocked() (*state, error) {
	content, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &state{}, nil
		}
		return nil, fmt.Errorf("read events store: %w", err)
	}
	var current state
	if err := json.Unmarshal(content, &current); err != nil {
		return nil, fmt.Errorf("decode events store: %w", err)
	}
	sortEvents(current.Events)
	return &current, nil
}

func (s *Store) saveLocked(current *state) error {
	content, err := json.MarshalIndent(current, "", "  ")
	if err != nil {
		return fmt.Errorf("encode events store: %w", err)
	}
	content = append(content, '\n')
	if err := os.WriteFile(s.path, content, 0o644); err != nil {
		return fmt.Errorf("write events store: %w", err)
	}
	return nil
}

func validateEvent(event Event) error {
	if event.ID == "" {
		return errors.New("event id is required")
	}
	if event.Type == "" {
		return errors.New("event type is required")
	}
	switch event.Severity {
	case SeverityInfo, SeverityWarning, SeverityError:
	default:
		return errors.New("event severity must be info, warning, or error")
	}
	if event.SourceComponent == "" {
		return errors.New("event source_component is required")
	}
	if event.OccurredAt == "" {
		return errors.New("event occurred_at is required")
	}
	if _, err := parseOccurredAt(event.OccurredAt); err != nil {
		return errors.New("event occurred_at must be RFC3339 or RFC3339Nano")
	}
	if event.Summary == "" {
		return errors.New("event summary is required")
	}
	return nil
}

func normalizeEvent(event Event) Event {
	event.ID = strings.ToLower(strings.TrimSpace(event.ID))
	event.Type = Type(strings.TrimSpace(string(event.Type)))
	event.Severity = Severity(strings.TrimSpace(string(event.Severity)))
	event.SiteID = strings.ToLower(strings.TrimSpace(event.SiteID))
	event.SourceComponent = strings.TrimSpace(event.SourceComponent)
	event.OccurredAt = strings.TrimSpace(event.OccurredAt)
	event.Summary = strings.TrimSpace(event.Summary)
	event.RelatedRevisionID = strings.ToLower(strings.TrimSpace(event.RelatedRevisionID))
	event.RelatedJobID = strings.ToLower(strings.TrimSpace(event.RelatedJobID))
	event.RelatedCertificateID = strings.ToLower(strings.TrimSpace(event.RelatedCertificateID))
	event.RelatedRuleID = strings.TrimSpace(event.RelatedRuleID)
	if len(event.Details) == 0 {
		event.Details = nil
	}
	return event
}

func sortEvents(items []Event) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].OccurredAt == items[j].OccurredAt {
			return items[i].ID < items[j].ID
		}
		return items[i].OccurredAt < items[j].OccurredAt
	})
}

func parseOccurredAt(value string) (time.Time, error) {
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed, nil
	}
	return time.Parse(time.RFC3339, value)
}
