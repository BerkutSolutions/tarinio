package events

import (
	"testing"
	"time"
)

func TestStore_CreateAndList(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	created, err := store.Create(Event{
		ID:              "EVT-1",
		Type:            TypeApplyStarted,
		Severity:        SeverityInfo,
		SourceComponent: "control-plane",
		OccurredAt:      "2026-04-01T10:00:00Z",
		Summary:         "apply started",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if created.ID != "evt-1" {
		t.Fatalf("unexpected normalization: %+v", created)
	}

	items, err := store.List()
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(items) != 1 || items[0].Type != TypeApplyStarted {
		t.Fatalf("unexpected items: %+v", items)
	}
}

func TestStore_PruneByCountAndAge(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	inputs := []Event{
		{ID: "evt-1", Type: TypeApplyStarted, Severity: SeverityInfo, SourceComponent: "apply-runner", OccurredAt: time.Now().UTC().Add(-48 * time.Hour).Format(time.RFC3339Nano), Summary: "old"},
		{ID: "evt-2", Type: TypeApplySucceeded, Severity: SeverityInfo, SourceComponent: "apply-runner", OccurredAt: time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339Nano), Summary: "mid"},
		{ID: "evt-3", Type: TypeApplyFailed, Severity: SeverityError, SourceComponent: "apply-runner", OccurredAt: time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339Nano), Summary: "new"},
	}
	for _, item := range inputs {
		if _, err := store.Create(item); err != nil {
			t.Fatalf("create failed: %v", err)
		}
	}

	pruned, err := store.Prune(RetentionPolicy{MaxEvents: 1, MaxAge: 24 * time.Hour})
	if err != nil {
		t.Fatalf("prune failed: %v", err)
	}
	if pruned != 2 {
		t.Fatalf("expected 2 pruned events, got %d", pruned)
	}

	items, err := store.List()
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(items) != 1 || items[0].ID != "evt-3" {
		t.Fatalf("unexpected retained events: %+v", items)
	}
}
