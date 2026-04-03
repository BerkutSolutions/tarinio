package services

import (
	"context"
	"testing"

	"waf/control-plane/internal/accesspolicies"
	"waf/control-plane/internal/sites"
)

func TestManualBanService_BanCreatesPolicyWhenMissing(t *testing.T) {
	store := &fakeAccessPolicyStore{}
	service := NewManualBanService(store, &fakeSiteReader{items: []sites.Site{{ID: "site-a"}}}, nil)

	policy, err := service.Ban(context.Background(), "site-a", "10.0.0.1")
	if err != nil {
		t.Fatalf("ban failed: %v", err)
	}
	if policy.ID != "site-a-access" || len(policy.DenyList) != 1 || policy.DenyList[0] != "10.0.0.1" {
		t.Fatalf("unexpected policy: %+v", policy)
	}
}

func TestManualBanService_BanIsIdempotent(t *testing.T) {
	store := &fakeAccessPolicyStore{
		items: []accesspolicies.AccessPolicy{{
			ID:       "access-a",
			SiteID:   "site-a",
			Enabled:  true,
			DenyList: []string{"10.0.0.1"},
		}},
	}
	service := NewManualBanService(store, &fakeSiteReader{items: []sites.Site{{ID: "site-a"}}}, nil)

	policy, err := service.Ban(context.Background(), "site-a", "10.0.0.1")
	if err != nil {
		t.Fatalf("ban failed: %v", err)
	}
	if len(policy.DenyList) != 1 {
		t.Fatalf("expected no duplicate deny entries, got %+v", policy.DenyList)
	}
}

func TestManualBanService_UnbanIsIdempotent(t *testing.T) {
	store := &fakeAccessPolicyStore{
		items: []accesspolicies.AccessPolicy{{
			ID:       "access-a",
			SiteID:   "site-a",
			Enabled:  true,
			DenyList: []string{"10.0.0.1"},
		}},
	}
	service := NewManualBanService(store, &fakeSiteReader{items: []sites.Site{{ID: "site-a"}}}, nil)

	policy, err := service.Unban(context.Background(), "site-a", "10.0.0.2")
	if err != nil {
		t.Fatalf("unban failed: %v", err)
	}
	if len(policy.DenyList) != 1 || policy.DenyList[0] != "10.0.0.1" {
		t.Fatalf("expected unchanged denylist, got %+v", policy.DenyList)
	}
}

func TestManualBanService_UnbanWithoutExistingPolicyIsNoOp(t *testing.T) {
	store := &fakeAccessPolicyStore{}
	service := NewManualBanService(store, &fakeSiteReader{items: []sites.Site{{ID: "site-a"}}}, nil)

	policy, err := service.Unban(context.Background(), "site-a", "10.0.0.1")
	if err != nil {
		t.Fatalf("unban failed: %v", err)
	}
	if policy.SiteID != "site-a" {
		t.Fatalf("expected site-a policy placeholder, got %+v", policy)
	}
	if len(policy.DenyList) != 0 {
		t.Fatalf("expected empty denylist, got %+v", policy.DenyList)
	}
}

func TestManualBanService_UnbanRemovesEntry(t *testing.T) {
	store := &fakeAccessPolicyStore{
		items: []accesspolicies.AccessPolicy{{
			ID:       "access-a",
			SiteID:   "site-a",
			Enabled:  true,
			DenyList: []string{"10.0.0.1", "10.0.0.2"},
		}},
	}
	service := NewManualBanService(store, &fakeSiteReader{items: []sites.Site{{ID: "site-a"}}}, nil)

	policy, err := service.Unban(context.Background(), "site-a", "10.0.0.1")
	if err != nil {
		t.Fatalf("unban failed: %v", err)
	}
	if len(policy.DenyList) != 1 || policy.DenyList[0] != "10.0.0.2" {
		t.Fatalf("expected filtered denylist, got %+v", policy.DenyList)
	}
}
