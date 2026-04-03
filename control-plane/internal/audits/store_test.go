package audits

import "testing"

func TestStore_ListFiltersAndPagination(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	_ = store.Append(AuditEvent{
		ID:           "a1",
		ActorUserID:  "admin",
		Action:       "site.create",
		ResourceType: "site",
		ResourceID:   "site-a",
		Status:       StatusSucceeded,
		OccurredAt:   "2026-04-01T10:00:00Z",
		Summary:      "site created",
	})
	_ = store.Append(AuditEvent{
		ID:           "a2",
		ActorUserID:  "admin",
		Action:       "site.delete",
		ResourceType: "site",
		ResourceID:   "site-b",
		Status:       StatusFailed,
		OccurredAt:   "2026-04-01T11:00:00Z",
		Summary:      "site delete failed",
	})

	result, err := store.List(Query{Action: "site.delete", Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 || result.Items[0].ID != "a2" {
		t.Fatalf("unexpected filtered result: %+v", result)
	}
}
