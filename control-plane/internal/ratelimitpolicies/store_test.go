package ratelimitpolicies

import "testing"

func TestStore_CreateNormalizesAndListsDeterministically(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	created, err := store.Create(RateLimitPolicy{
		ID:      "  POLICY-A ",
		SiteID:  " Site-A ",
		Enabled: true,
		Limits: Limits{
			RequestsPerSecond: 20,
			Burst:             10,
		},
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if created.ID != "policy-a" || created.SiteID != "site-a" {
		t.Fatalf("unexpected normalization: %+v", created)
	}

	items, err := store.List()
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(items) != 1 || items[0].ID != "policy-a" {
		t.Fatalf("unexpected items: %+v", items)
	}
}

func TestStore_RejectsInvalidEnabledLimits(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	_, err = store.Create(RateLimitPolicy{
		ID:      "policy-a",
		SiteID:  "site-a",
		Enabled: true,
		Limits: Limits{
			RequestsPerSecond: 0,
			Burst:             5,
		},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}
