package services

import (
	"crypto/sha256"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"waf/control-plane/internal/events"
)

type EventStore interface {
	Create(event events.Event) (events.Event, error)
	List() ([]events.Event, error)
	Prune(policy events.RetentionPolicy) (int, error)
}

type EventService struct {
	store     EventStore
	retention events.RetentionPolicy
	collector RuntimeSecurityEventCollector

	collectorMu              sync.Mutex
	collectorNextAttemptAt   time.Time
	collectorLastError       string
	collectorLastLoggedError time.Time
}

type EventServiceOption func(*EventService)

func WithRuntimeSecurityCollector(collector RuntimeSecurityEventCollector) EventServiceOption {
	return func(s *EventService) {
		s.collector = collector
	}
}

func NewEventService(store EventStore, options ...EventServiceOption) *EventService {
	s := &EventService{
		store: store,
		retention: events.RetentionPolicy{
			MaxEvents: 1000,
			MaxAge:    30 * 24 * time.Hour,
		},
	}
	for _, option := range options {
		if option != nil {
			option(s)
		}
	}
	return s
}

func (s *EventService) Emit(event events.Event) (events.Event, error) {
	if strings.TrimSpace(event.OccurredAt) == "" {
		event.OccurredAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	if strings.TrimSpace(event.ID) == "" {
		event.ID = newEventID(event)
	}
	created, err := s.store.Create(event)
	if err != nil {
		return events.Event{}, err
	}
	_, _ = s.store.Prune(s.retention)
	return created, nil
}

func (s *EventService) List() ([]events.Event, error) {
	s.collectRuntimeSecurityEventsBestEffort()
	return s.store.List()
}

func (s *EventService) SetRetention(policy events.RetentionPolicy) {
	s.retention = policy
}

func newEventID(event events.Event) string {
	sum := sha256.Sum256([]byte(
		string(event.Type) + ":" +
			string(event.Severity) + ":" +
			event.SourceComponent + ":" +
			event.RelatedRevisionID + ":" +
			event.RelatedJobID + ":" +
			event.OccurredAt + ":" +
			event.Summary,
	))
	return fmt.Sprintf("evt-%x", sum[:6])
}

func (s *EventService) collectRuntimeSecurityEventsBestEffort() {
	if s.collector == nil {
		return
	}
	if !s.shouldCollectRuntimeSecurityEvents() {
		return
	}
	items, err := s.collector.Collect()
	if err != nil {
		s.recordCollectorFailure(err)
		return
	}
	s.recordCollectorSuccess()
	for _, item := range items {
		if _, err := s.Emit(item); err != nil && !strings.Contains(strings.ToLower(err.Error()), "already exists") {
			log.Printf("event service security event emit failed: %v", err)
		}
	}
}

func (s *EventService) shouldCollectRuntimeSecurityEvents() bool {
	s.collectorMu.Lock()
	defer s.collectorMu.Unlock()
	now := time.Now()
	if !s.collectorNextAttemptAt.IsZero() && now.Before(s.collectorNextAttemptAt) {
		return false
	}
	s.collectorNextAttemptAt = now.Add(5 * time.Second)
	return true
}

func (s *EventService) recordCollectorSuccess() {
	s.collectorMu.Lock()
	defer s.collectorMu.Unlock()
	s.collectorLastError = ""
	s.collectorNextAttemptAt = time.Now().Add(5 * time.Second)
}

func (s *EventService) recordCollectorFailure(err error) {
	message := strings.TrimSpace(err.Error())
	now := time.Now()

	s.collectorMu.Lock()
	defer s.collectorMu.Unlock()
	s.collectorNextAttemptAt = now.Add(30 * time.Second)
	shouldLog := message != s.collectorLastError || now.Sub(s.collectorLastLoggedError) >= time.Minute
	s.collectorLastError = message
	if shouldLog {
		s.collectorLastLoggedError = now
		log.Printf("event service runtime security collector failed: %v", err)
	}
}
