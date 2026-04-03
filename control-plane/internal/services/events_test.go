package services

import (
	"testing"

	"waf/control-plane/internal/events"
)

type fakeEventStore struct {
	items  []events.Event
	pruned []events.RetentionPolicy
}

func (f *fakeEventStore) Create(event events.Event) (events.Event, error) {
	f.items = append(f.items, event)
	return event, nil
}

func (f *fakeEventStore) List() ([]events.Event, error) {
	return append([]events.Event(nil), f.items...), nil
}

func (f *fakeEventStore) Prune(policy events.RetentionPolicy) (int, error) {
	f.pruned = append(f.pruned, policy)
	return 0, nil
}

func TestEventService_EmitGeneratesIDAndTimestamp(t *testing.T) {
	store := &fakeEventStore{}
	service := NewEventService(store)

	event, err := service.Emit(events.Event{
		Type:            events.TypeApplyStarted,
		Severity:        events.SeverityInfo,
		SourceComponent: "control-plane",
		Summary:         "apply started",
	})
	if err != nil {
		t.Fatalf("emit failed: %v", err)
	}
	if event.ID == "" || event.OccurredAt == "" {
		t.Fatalf("expected generated event metadata, got %+v", event)
	}
	if len(store.pruned) != 1 {
		t.Fatalf("expected prune to be called, got %+v", store.pruned)
	}
}
