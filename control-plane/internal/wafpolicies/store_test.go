package wafpolicies

import "testing"

func TestStore_CreateListUpdateDelete(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}

	created, err := store.Create(WAFPolicy{
		ID:                 "waf-a",
		SiteID:             "site-a",
		Enabled:            true,
		Mode:               ModeDetection,
		CRSEnabled:         true,
		CustomRuleIncludes: []string{"rules/a.conf"},
		RuleOverrides:      []RuleOverride{{RuleID: "941100", Enabled: false}},
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if created.CreatedAt == "" || created.UpdatedAt == "" {
		t.Fatal("expected timestamps")
	}

	items, err := store.List()
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(items) != 1 || items[0].ID != "waf-a" {
		t.Fatalf("unexpected items: %+v", items)
	}

	updated, err := store.Update(WAFPolicy{
		ID:                 "waf-a",
		SiteID:             "site-a",
		Enabled:            true,
		Mode:               ModePrevention,
		CRSEnabled:         false,
		CustomRuleIncludes: []string{"rules/b.conf"},
		RuleOverrides:      []RuleOverride{{RuleID: "942100", Enabled: false}},
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if updated.Mode != ModePrevention || updated.CRSEnabled {
		t.Fatalf("unexpected updated item: %+v", updated)
	}

	if err := store.Delete("waf-a"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	items, err = store.List()
	if err != nil {
		t.Fatalf("list after delete failed: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty list, got %+v", items)
	}
}

func TestStore_RejectsInvalidFields(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}

	if _, err := store.Create(WAFPolicy{ID: "waf-a", SiteID: "site-a", Enabled: true, Mode: "bad"}); err == nil {
		t.Fatal("expected mode validation error")
	}
	if _, err := store.Create(WAFPolicy{ID: "waf-a", SiteID: "site-a", RuleOverrides: []RuleOverride{{RuleID: ""}}}); err == nil {
		t.Fatal("expected rule override validation error")
	}
}

func TestStore_RejectsSecondPolicyForSameSite(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}
	if _, err := store.Create(WAFPolicy{ID: "waf-a", SiteID: "site-a"}); err != nil {
		t.Fatalf("first create failed: %v", err)
	}
	if _, err := store.Create(WAFPolicy{ID: "waf-b", SiteID: "site-a"}); err == nil {
		t.Fatal("expected duplicate site policy error")
	}
}
