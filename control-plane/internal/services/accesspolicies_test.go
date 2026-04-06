package services

import (
	"context"
	"fmt"
	"testing"

	"waf/control-plane/internal/accesspolicies"
	"waf/control-plane/internal/sites"
)

type fakeAccessPolicyStore struct {
	items []accesspolicies.AccessPolicy
}

func (f *fakeAccessPolicyStore) Create(item accesspolicies.AccessPolicy) (accesspolicies.AccessPolicy, error) {
	f.items = append(f.items, item)
	return item, nil
}

func (f *fakeAccessPolicyStore) List() ([]accesspolicies.AccessPolicy, error) {
	return append([]accesspolicies.AccessPolicy(nil), f.items...), nil
}

func (f *fakeAccessPolicyStore) Update(item accesspolicies.AccessPolicy) (accesspolicies.AccessPolicy, error) {
	for i := range f.items {
		if f.items[i].ID == item.ID {
			f.items[i] = item
			return item, nil
		}
	}
	return accesspolicies.AccessPolicy{}, fmt.Errorf("access policy %s not found", item.ID)
}

func (f *fakeAccessPolicyStore) Delete(id string) error {
	return nil
}

func TestAccessPolicyService_CreateValidatesSite(t *testing.T) {
	service := NewAccessPolicyService(
		&fakeAccessPolicyStore{},
		&fakeSiteReader{items: []sites.Site{{ID: "site-a"}}},
		nil,
	)

	created, err := service.Create(context.Background(), accesspolicies.AccessPolicy{
		ID:     "access-a",
		SiteID: "site-a",
	})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if created.ID != "access-a" {
		t.Fatalf("unexpected policy: %+v", created)
	}
}

func TestAccessPolicyService_CreateRejectsMissingSite(t *testing.T) {
	service := NewAccessPolicyService(
		&fakeAccessPolicyStore{},
		&fakeSiteReader{},
		nil,
	)

	if _, err := service.Create(context.Background(), accesspolicies.AccessPolicy{ID: "access-a", SiteID: "site-a"}); err == nil {
		t.Fatal("expected missing site error")
	}
}

func TestAccessPolicyService_UpsertCreatesWhenMissing(t *testing.T) {
	store := &fakeAccessPolicyStore{}
	service := NewAccessPolicyService(
		store,
		&fakeSiteReader{items: []sites.Site{{ID: "site-a"}}},
		nil,
	)

	out, err := service.Upsert(context.Background(), accesspolicies.AccessPolicy{
		SiteID:    "site-a",
		AllowList: []string{"10.0.0.1"},
	})
	if err != nil {
		t.Fatalf("upsert failed: %v", err)
	}
	if out.ID == "" {
		t.Fatalf("expected generated id, got %+v", out)
	}
	if len(store.items) != 1 {
		t.Fatalf("expected 1 item in store, got %d", len(store.items))
	}
}

func TestAccessPolicyService_UpsertUpdatesExistingBySiteID(t *testing.T) {
	store := &fakeAccessPolicyStore{
		items: []accesspolicies.AccessPolicy{
			{ID: "legacy-access-id", SiteID: "site-a", AllowList: []string{"10.0.0.1"}},
		},
	}
	service := NewAccessPolicyService(
		store,
		&fakeSiteReader{items: []sites.Site{{ID: "site-a"}}},
		nil,
	)

	out, err := service.Upsert(context.Background(), accesspolicies.AccessPolicy{
		ID:       "different-id",
		SiteID:   "site-a",
		DenyList: []string{"192.168.1.1"},
	})
	if err != nil {
		t.Fatalf("upsert failed: %v", err)
	}
	if out.ID != "legacy-access-id" {
		t.Fatalf("expected existing id preserved, got %+v", out)
	}
	if len(store.items) != 1 || store.items[0].ID != "legacy-access-id" {
		t.Fatalf("expected in-place update, got %+v", store.items)
	}
	if len(store.items[0].DenyList) != 1 || store.items[0].DenyList[0] != "192.168.1.1" {
		t.Fatalf("expected denylist updated, got %+v", store.items[0])
	}
}
